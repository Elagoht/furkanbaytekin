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
documents/                  routes that are not HTML: /rss, /robots.txt, /sitemap.xml, /llms.txt, and /healthz
templates/                  the HTML, one file per fragment
static/                     CSS, icons and the web manifest, at /static/
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

A page's optional `person` block adds a schema.org `Person` to its head, with
site.json's person as the base. Editing a section's data shows up on reload
under `collage dev`, and so does adding, removing or reordering sections: a page's
sections are read from its JSON on every render.

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
| `/blogs/{slug}/view` | POST: counts a view in the CMS, answers `{"views": n}` | — |
| `/rss` | the latest posts as RSS 2.0, linked from every page's head | 30 minutes |
| `/sitemap.xml` | the fixed pages and every post, with its update date | 1 hour |
| `/robots.txt` | everything allowed but search, and the sitemap | 1 hour |
| `/llms.txt` | [llms.txt](https://llmstxt.org): who the site is about, its pages, every post | 1 hour |

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

- **opti-image** fetches images with a declared `width` and `height` from the
  site (`site.json`'s `url`) and the CMS (`BLOG_API_URL`) — the list is built in
  `main.go` from those two, not written in the config — resizes them, with
  `"webp": "auto"`: PNG sources become WebP, photographs JPEG,, and serves them from `/_image/`
  (`collage export` writes them into `dist/_image/`).
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
collage build              # -> bin/furkanbaytekin, built for linux/amd64
collage build -i           # also offers a Dockerfile and a systemd unit in bin/
HOST=0.0.0.0 PORT=8080 COLLAGE_CSRF_KEY=... ./bin/furkanbaytekin
```

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
