// Command furkanbaytekin serves this collage application, or — when invoked with
// -collage-build — renders it to static files instead of serving it.
//
// # The collage CLI contract
//
// "collage dev" runs `go run .` here with COLLAGE_DEV=1 set, and the variables
// of .env.development (or .env) added, and "collage export" runs
// `go run . -collage-build -out <dir>`. This file honours both by reading that
// variable and those flags below. Development mode reloads templates from disk
// on every request; it does not hot-reload Go code, so a change to any .go file
// here still needs a restart.
//
// "collage build" needs nothing from this file: it compiles the program, which
// is something go build does without being told anything.
//
// If you rewrite this file, keep both halves working, or "collage dev" and
// "collage export" stop doing anything useful here.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	feed "github.com/Elagoht/collage-feed"
	jsonld "github.com/Elagoht/collage-jsonld"
	meta "github.com/Elagoht/collage-meta"
	minimizer "github.com/Elagoht/collage-minimizer"
	ogimage "github.com/Elagoht/collage-ogimage"
	optiimage "github.com/Elagoht/collage-opti-image"
	robots "github.com/Elagoht/collage-robots"
	sitemap "github.com/Elagoht/collage-sitemap"
	"github.com/Elagoht/collage/pkg/collage"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"
	"furkanbaytekin/documents"
	"furkanbaytekin/fragments/sections"
)

// Templates and static files are embedded, so this binary runs from anywhere:
// a container with a different WORKDIR, a systemd unit, a copy on a server.
//
// It costs nothing in development. With DevMode on, collage prefers the
// directory on disk whenever it is there — which it is while you are working
// in this project — so editing a template is still visible on the next
// request, embedded copy or not.
//
//go:embed all:templates
var templatesFS embed.FS

//go:embed all:static
var staticFS embed.FS

// The faces share cards are drawn in: the site's own Outfit, as static TrueType
// files, which is what the card renderer reads (the pages load it from Google
// Fonts instead). Outside static/, because no page asks for them.
//
//go:embed fonts/*.ttf
var fontsFS embed.FS

func main() {
	// Before the flags: -port's default is read from PORT. "collage dev" reads
	// its own environment file, and says so with COLLAGE_DEV.
	if os.Getenv("COLLAGE_DEV") != "1" {
		if err := loadEnvFile(".env"); err != nil {
			log.Fatalf("furkanbaytekin: %v", err)
		}
	}

	buildFlag := flag.Bool("collage-build", false, "render the app to static files instead of serving it")
	outFlag := flag.String("out", "dist", "output directory for -collage-build")
	cleanFlag := flag.Bool("clean", false, "remove -out's existing contents before building")
	portFlag := flag.Int("port", envInt("PORT", 3000), "port to listen on (env PORT)")
	flag.Parse()

	devMode := os.Getenv("COLLAGE_DEV") == "1"

	app, err := newApp(devMode, *portFlag)
	if err != nil {
		log.Fatalf("furkanbaytekin: %v", err)
	}

	// A word after the flags is a command: a plugin's, or collage's own
	// collage-inspect and collage-check, which "collage inspect" and "collage
	// check" run as `go run . <command>`. A word nobody claims is a usage error
	// rather than a server started by accident.
	if args := flag.Args(); len(args) > 0 {
		code, err := collage.DispatchCommands(context.Background(), app, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "furkanbaytekin: %v\n", err)
		}
		os.Exit(code)
	}

	if *buildFlag {
		if err := staticBuild(app, *outFlag, *cleanFlag); err != nil {
			log.Fatalf("furkanbaytekin: static build: %v", err)
		}
		return
	}

	if err := app.ListenAndServe(); err != nil {
		log.Fatalf("furkanbaytekin: %v", err)
	}
}

// newApp builds the application: its configuration, its routes, and its static
// mount.
//
// It is separate from main so that the tests can build the same application
// and drive it through app.Handler(), with no server listening and no port to
// pick. What they exercise is then the site that actually runs, rather than a
// second wiring that can drift from it.
func newApp(devMode bool, port int) (*collage.App, error) {
	// Plugin configuration, keyed by plugin name. A missing file is not an
	// error: every plugin then runs on its defaults.
	pluginConfig, err := collage.LoadPluginConfig("plugins-config.json")
	if err != nil {
		return nil, fmt.Errorf("plugin configuration: %w", err)
	}

	store := contentFiles(devMode)
	site, err := store.Site()
	if err != nil {
		return nil, err
	}

	client, err := blog.New(
		envString("BLOG_API_URL", "https://myblogcms.furkanbaytekin.dev/api"),
		os.Getenv("BLOG_TRUSTED_FRONTEND_KEY"),
	)
	if err != nil {
		return nil, err
	}

	// opti-image fetches from the CMS, and from nowhere else: the site's own
	// images it reads from the static files (Files, below), with no request to
	// itself. The CMS is named where it is configured already, so the list
	// cannot drift from it; plugins-config.json holds the rest.
	origins, err := imageOrigins(client.Origin())
	if err != nil {
		return nil, err
	}

	// The feed, read once here: its title and description are blog.json's.
	posts, err := documents.Feed(store, client)
	if err != nil {
		return nil, err
	}

	// The static files are mounted below, and share cards read the avatar
	// from them.
	assets, err := staticFiles(devMode)
	if err != nil {
		return nil, err
	}

	// Rendered pages and produced images live under one directory, so one
	// setting moves or isolates both.
	cacheDir := envString("CACHE_DIR", ".cache")

	app, err := collage.New(&collage.Config{
		DevMode: devMode,
		// The site's public origin, from site.json: the canonical URLs, the
		// sitemap and the feed are absolute against it.
		BaseURL: site.URL,
		Server: collage.ServerConfig{
			Host: envString("HOST", "localhost"),
			Port: port,
		},
		Template: collage.TemplateConfig{
			FS:        templatesFS,
			Root:      "templates",
			Extension: ".html",
			Funcs:     template.FuncMap{"emphasis": sections.Emphasis},
		},
		Cache: collage.CacheConfig{
			// In development collage never reads from the cache — a cached
			// page would hide the template you just edited — and uses memory
			// instead of disk. In production rendered pages are kept under
			// .cache, namespaced by a hash of this binary.
			Enabled:    true,
			Type:       "disk",
			Dir:        cacheDir,
			DefaultTTL: 5 * time.Minute,
		},
		PluginConfig: pluginConfig,
		// opti-image, jsonld and minimizer take the rest of their settings
		// from plugins-config.json. opti-image and minimizer have to be here
		// rather than registered later: they mount filesystems while the
		// application is built. jsonld, meta and feed reach the document head
		// through {{hoist "head"}} in the layout. The SEO plugins take
		// BaseURL above.
		Plugins: []collage.Plugin{
			optiimage.NewWith(optiimage.Config{
				AllowedOrigins: origins,
				// "/static/avatar.png" is resized from the file /static/
				// serves, so a test or an export needs no network, and the
				// site does not depend on its own public address being up.
				Files:    map[string]fs.FS{"/static/": assets},
				CacheDir: filepath.Join(cacheDir, "opti-image"),
			}),
			jsonld.New(),
			minimizer.New(),
			meta.New(meta.Options{
				SiteName: site.Name,
				Locales:  map[string]string{"en": "en_US"},
			}),
			// After meta, so the card's og:image is declared after meta's:
			// a post's card replaces its cover, and every other page gets
			// the default card from its title and description.
			ogimage.NewWith(ogimage.Config{
				Templates: cardTemplates(devMode),
				Root:      ".",
				Default:   "og/default.html",
				SiteName:  site.Name,
				// The avatar is read from the static files, and a
				// cover from the CMS.
				Files:        map[string]fs.FS{"/static/": assets},
				ImageOrigins: []string{client.Origin()},
				Fonts: []ogimage.Font{
					{Family: "Outfit", Weight: 400, File: "fonts/Outfit-Regular.ttf"},
					{Family: "Outfit", Weight: 600, File: "fonts/Outfit-SemiBold.ttf"},
					{Family: "Outfit", Weight: 700, File: "fonts/Outfit-Bold.ttf"},
				},
				FontFiles: fontsFS,
				// Beside the page cache: a cached page names a card that a
				// restarted process must still be able to draw.
				Dir: filepath.Join(cacheDir, "ogimage"),
			}),
			sitemap.New(sitemap.Options{
				// Every search is a fresh render and says nothing the list
				// does not.
				Exclude: []string{"blogs-search"},
				LastMod: documents.PostModified(client),
			}),
			robots.New(robots.Options{
				Rules:    []robots.Rule{{Allow: []string{"/"}, Disallow: []string{"/blogs/search"}}},
				Sitemaps: []string{site.URL + "/sitemap.xml"},
			}),
			feed.New(posts),
		},
		Security: collage.SecurityConfig{
			// Signs the forgery tokens forms carry. Unset, one is generated
			// per process: fine in development, wrong to deploy, because every
			// form submitted before a restart is refused after it. Make one
			// with `openssl rand -hex 32`.
			CSRFKey: []byte(os.Getenv("COLLAGE_CSRF_KEY")),
		},
	})
	if err != nil {
		return nil, err
	}

	if err := register(app, routeSources{
		store:         store,
		client:        client,
		webhookSecret: os.Getenv("WEBHOOK_SECRET"),
		log:           slog.Default(),
	}); err != nil {
		return nil, err
	}

	// A page's Markdown names the page as its canonical address.
	if err := app.Use(markdownCanonical(site.URL)); err != nil {
		return nil, fmt.Errorf("markdown canonical: %w", err)
	}

	if err := app.Mount("/static/", assets); err != nil {
		return nil, fmt.Errorf("mount static files: %w", err)
	}

	return app, nil
}

// staticFiles returns the filesystem "/static/" is served from: the embedded
// copy, except in development, where the directory on disk wins so an edited
// stylesheet shows up without a rebuild.
//
// os.OpenRoot rather than os.DirFS: os.DirFS follows a symlink out of the
// directory, and an os.Root does not.
func staticFiles(devMode bool) (fs.FS, error) {
	if devMode {
		if root, err := os.OpenRoot("static"); err == nil {
			return root.FS(), nil
		}
	}

	// fs.Sub, because the embedded tree contains the "static" directory
	// itself: mounting it whole would serve "/static/static/app.css".
	return fs.Sub(staticFS, "static")
}

// cardTemplates returns the filesystem share card templates are read from, rooted
// at templates/: the embedded copy, except in development, where the directory on
// disk wins so an edited card shows up on the next render, as a page does.
func cardTemplates(devMode bool) fs.FS {
	if devMode {
		if root, err := os.OpenRoot("templates"); err == nil {
			return root.FS()
		}
	}
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic(err) // a constant, valid path
	}
	return sub
}

// imageOrigins is the scheme and host of each of urls.
func imageOrigins(urls ...string) ([]optiimage.Origin, error) {
	origins := make([]optiimage.Origin, 0, len(urls))
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("image origin %q is not an absolute URL", raw)
		}
		origins = append(origins, optiimage.Origin{Scheme: u.Scheme, Host: u.Host})
	}
	return origins, nil
}

// contentFiles returns the store the pages read their JSON from: the embedded
// copy, except in development, where the directory on disk wins so an edited
// file shows up on the next request, as a template does.
func contentFiles(devMode bool) *content.Store {
	if devMode {
		if root, err := os.OpenRoot("data/content"); err == nil {
			return content.NewStore(root.FS())
		}
	}
	return content.NewStore(content.Embedded())
}

// envString returns the environment variable named key, or fallback.
func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// envInt is envString for a number. An unparseable value falls back rather than
// failing: a port is not worth refusing to start over.
func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

// staticBuild renders every statically-buildable page to files under outDir
// through collage's own builder, and prints what was written, skipped and
// failed.
func staticBuild(app *collage.App, outDir string, clean bool) error {
	builder, err := collage.NewBuilder(app, collage.BuildOptions{
		OutDir: outDir,
		Clean:  clean,
	})
	if err != nil {
		return err
	}

	report, buildErr := builder.Build(context.Background())

	collage.PrintBuildReport(os.Stdout, report, buildErr)

	return buildErr
}
