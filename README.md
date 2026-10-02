# furkanbaytekin

A [collage](https://github.com/Elagoht/collage) project, scaffolded by
`collage new`.

## Running

```
go mod tidy
cp .env.example .env.development
collage dev
```

The site is at http://localhost:3000. `collage dev` reads `.env.development`
(or `.env` when there is none) and runs `go run .` in development mode:
templates and static files are read from disk on every request, so editing one
and the browser reloads itself. A change to Go code rebuilds and restarts the
program.

A variable set in your shell wins over the file, so `PORT=4000 collage dev`
still works. Only `collage dev` reads these files; the built binary takes its
environment from wherever it runs.

```
go test ./...
```

## What is here

```
main.go                     configuration, the plugins, the static mount, and the collage CLI contract
routes.go                   every page, document and action, in one app.Register call
main_test.go, blog_test.go  the tests, driving app.Handler() with no server and no port
data/content/               the site's data as JSON: site.json (header, footer, person), blog.json, one file per page
data/blog/                  the Bloggo CMS client and the Markdown renderer
pages/<area>/               one file per page: its layout, content, path and caching
fragments/pages/<area>/     each page's content and the data it reads, mirroring pages/
fragments/layouts/          Master(), the shell every page renders inside
fragments/sections/         the blocks a section page is made of; fragments/seo, its head tags
actions/                    POST /blogs/{slug}/view and POST /api/webhook; their handlers in actions/funcs/
documents/                  routes that are not HTML: /llms.txt and /healthz, and what the feed and sitemap plugins list
templates/                  the HTML, one file per fragment; templates/og/, the share cards
static/                     CSS, icons and the web manifest, at /static/
fonts/                      Outfit as TrueType, which the share cards are drawn in (OFL.txt)
og_test.go                  the share cards' tests
frontmatter/                the YAML front matter a page's or a post's .md opens with
canonical.go                the Link header naming a page canonical for its .md
```

The areas are `landing` (`/`, `/about`), `blog` and `errors` (the not-found page).

Every page is a JSON file in `data/content/` — `/` is `home.json`, `/about` is
`about.json` — given its path in `pages/landing/` and registered in `routes.go`, and declared `Static()`,
so `collage export` writes it to a file. Each entry of a page's `sections` is
one fragment from `fragments/sections/`, with its template in
`templates/fragments/sections/` and its stylesheet in `static/sections/`:

| type | what it is |
| --- | --- |
| `hero` | avatar, name, titles, bio, buttons, social links |
| `hobbies` | icon cards |
| `stats` | linked numbers |
| `profile` | image, heading and a paragraph |
| `chips` | a labelled row of tags |
| `stack` | groups of tags |
| `expertise` | a two-column list |
| `experience` | jobs, with `**bold**` phrases in their points |
| `education` | schools |
| `separator` | a hairline, no data |

The not-found page's words are `site.json`'s `notFound`.

The previous site served its icons, its manifest and `/rss.xml` at the root.
Those addresses answer `301` to where this one serves them
(`pages/landing/home.go`'s `legacyAddresses`): browsers ask for `/favicon.ico`
whatever a page links, and a feed reader keeps the address it subscribed to.

`/about.md` is `/about` as Markdown, for a reader that wants the text alone: its
`seo` title and description and its address as front matter, then each section
written by its type (`fragments/sections/markdown.go`) — a list for chips and
stacks, a heading per job — with no separators. A section type has to say how it
is written as Markdown to compile at all. The page links it with
`rel="alternate"`, it names the page as canonical in a `Link` header
(`canonical.go`, as a post's `.md` does), and llms.txt links it. Another section
page gets one by registering `sectionMarkdown` for it in `pages/landing/`.

A page's optional `person` block adds a schema.org `Person` to its head, with
site.json's person as the base. Editing a section's data shows up on reload
under `collage dev`, and so does adding, removing or reordering sections: a page's
sections are read from its JSON on every render.

## Head, feed and sitemap

The canonical URL, Open Graph and Twitter tags in every page's head are
[elagoht/meta](https://github.com/Elagoht/collage-meta)'s, from the page's own
address against `Config.BaseURL`, which is `site.json`'s `url`. A page's JSON
gives its title and description (`seo`); a post gives its dates and author
too. The feed's title and description are `blog.json`'s `feed`, read once when the
site starts: change them and restart.

## Share cards

The image a link shows on X, LinkedIn, Slack or a messenger is drawn by this site,
by [elagoht/ogimage](https://github.com/Elagoht/collage-ogimage), from an HTML
template in `templates/og/`:

- `og/post.html` is a post's: its title, category, date, read time and cover,
  set in `fragments/pages/blog/post.go` with `ogimage.Set`, after `meta.Set` so
  the card's `og:image` is the one the head keeps;
- `og/default.html` is every other page's, from the title and description in its
  head, with no code.

They are drawn in Outfit, from `fonts/`, in `site.css`'s dark palette; the
avatar comes from `static/icons/`, and a cover from the CMS. A card is drawn on
its first request, kept under `$CACHE_DIR/ogimage`, and served at a URL made from
what is on it, `/_og/<hash>.png`: a retitled post gets a new card at a new URL,
and the old one still answers for shares made before.

Under `collage dev`, `/_og-preview/` shows the card each page carried on its last
render, and an edit to a card template shows on the page's next reload. To draw
one card without a page:

```sh
echo '{"title":"A title","label":"Software","fields":{"date":"Oct 2, 2026","readTime":"7"}}' > card.json
go run . ogimage og/post.html card.json > card.png
```

The template's CSS is a subset — flex boxes and text, no `position` or grid; the
plugin's README lists it — and a property outside it stops the site at startup,
naming the template and line.

## The blog

Posts come live from the Bloggo CMS at `BLOG_API_URL`, authenticated with
`BLOG_TRUSTED_FRONTEND_KEY` in the `x-trusted-frontend` header — both from the
environment (see `.env.example`), and the key never leaves the server. The
words around the posts are in `data/content/blog.json`.

| route | what it is | cached |
| --- | --- | --- |
| `/blogs` | the list, with category and tag filters and pages | 5 minutes, per `page`, `category`, `tag` |
| `/blogs/search?search=` | the same list, searched | never: its key would be whatever anyone types |
| `/blogs/{slug}` | a post: Markdown with highlighted code, a table of contents, related posts | 10 minutes |
| `/blogs/{slug}.md` | the same post as Markdown: its properties as YAML front matter (title, description, url, author, category, tags, dates, read time, cover), then the body as the CMS holds it, with its images and uploads made absolute; the page links it with `rel="alternate"`, and it names the page as canonical in a `Link` header (`canonical.go`), so a search engine indexes the post once | 10 minutes, dropped with the page |
| `/blogs/{slug}/view` | POST: counts a view in the CMS, answers `{"views": n}` | — |
| `/rss` | [elagoht/feed](https://github.com/Elagoht/collage-feed): the latest posts as RSS 2.0, linked from every page's head | until a post changes |
| `/sitemap.xml` | [elagoht/sitemap](https://github.com/Elagoht/collage-sitemap): every page but search, and every post with its update date | until a post changes |
| `/robots.txt` | [elagoht/robots](https://github.com/Elagoht/collage-robots): everything allowed but search, and the sitemap | fixed at start |
| `/llms.txt` | [llms.txt](https://llmstxt.org): who the site is about, its pages, every post, linked as its `.md` | 1 hour |

The times above are a fallback. Bloggo reports every change to
`POST /api/webhook` with `X-Webhook-Secret` (`WEBHOOK_SECRET`), and the pages
the change made wrong are dropped from the cache at once: a post's own page and
every list, feed and sitemap for a post; everything showing categories or tags
for those; post pages for an author; all of it for the panel's manual sync.
Each page says what it read through its data handler's tags (`blog/tags.go`),
and the disk cache keeps those tags in its entries, so a webhook after a
restart still reaches pages cached before it.

A post's Markdown may contain raw HTML — embeds and live demos — and it is
rendered as written: the CMS is trusted like the templates are. The post page's
script counts the view and rolls the counter to the live number, so a cached
page still shows a current one. Every image the CMS stores is 1280×720, so covers
and the CMS images inside a post declare that size and opti-image serves them
from `/_image/`; a post image hosted anywhere else is linked as it is.

These pages need the server, so `collage export` skips them.

Three plugins are installed, configured in `plugins-config.json`:

- **opti-image** resizes images with a declared `width` and `height`: the
  site's own, under `/static/`, read from the static files with no request made
  (`Files` in `main.go`), and the CMS's (`BLOG_API_URL`), fetched — the only
  origin it fetches from, built in `main.go`, not written in the config. With
  `"webp": "auto"` PNG sources become WebP and photographs JPEG, served from
  `/_image/` (`collage export` writes them into `dist/_image/`). A page's own
  image is written as a path — `"src": "/static/icons/…"` — not as the site's
  address, which the site would have to be up at to fetch from itself.
- **jsonld** adds a `WebSite` node; the home page emits a `Person`.
- **minimizer** strips whitespace from pages and assets.

## Set `COLLAGE_CSRF_KEY` before deploying

The site has no forms yet, but collage also namespaces its disk cache by this
key: without one, a key is generated per process and the rendered pages under
`.cache` are thrown away on every restart. It also signs the tokens
`{{csrfToken}}` puts in forms, once there are any.

```
openssl rand -hex 32
```

Put the result in `.env.development` for development, and in your production
environment's secrets for the deployed binary.

## Deploying

There is no `collage start`: this is a Go program, so production is a binary.

```
collage build -os linux -arch amd64 -o bin/furkanbaytekin-linux-amd64   # or -arch arm64
collage build -i           # also offers a Dockerfile and a systemd unit in bin/
HOST=127.0.0.1 PORT=8080 COLLAGE_CSRF_KEY=... ./bin/furkanbaytekin-linux-amd64
```

`collage build` alone builds for the machine it runs on, so a binary built on a
Mac does not run on a Linux server: name the target. `HOST=127.0.0.1` when a
reverse proxy on the same machine (Caddy, a Cloudflare tunnel) is what the
public reaches; `0.0.0.0` only when the proxy is elsewhere, as in a container.
The binary reads `.env` from its working directory, and a variable already set
in the environment wins. `CACHE_DIR` has to outlive a restart and a deploy: a
cached page names share cards and images the next process must still be able
to draw.

Templates and static files are embedded, so the binary is the whole site —
nothing to copy next to it. It drains in-flight requests on `SIGTERM`, answers
`/healthz` for a liveness check, and caches rendered pages and resized images
under `.cache` — or `CACHE_DIR` — so a restart does not re-render everything.

`collage export` writes the static pages to `dist/` for any static host. Pages
that need a server — a form, or a `Dynamic()` page — are skipped and named, and
still work when the binary serves them.

## The collage CLI contract

- **`collage dev`** runs `go run .` here with `COLLAGE_DEV=1` set. `main.go`
  reads that and turns on development mode.
- **`collage export`** runs `go run . -collage-build -out <dir>`. `main.go`
  parses those flags and renders through `collage.NewBuilder`, collage's own
  builder.

If you rewrite `main.go`, keep both halves working, or `collage dev` and
`collage export` stop doing anything useful here.
