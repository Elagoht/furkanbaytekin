package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/Elagoht/collage/pkg/collagetest"
)

const (
	testKey           = "test-trusted-frontend-key"
	testWebhookSecret = "test-webhook-secret"
)

// cms records what the fake CMS was asked.
type cms struct {
	mu      sync.Mutex
	queries []string
	views   []string
	paths   []string
	title   string
	// extra is appended to the list of posts, for a post published mid-test.
	extra string
}

func (c *cms) record(list *[]string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	*list = append(*list, value)
}

func (c *cms) post(format string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return fmt.Sprintf(format, c.title)
}

func (c *cms) retitle(title string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.title = title
}

// fakeCMS stands in for Bloggo, so the tests neither need the network nor
// touch the live counters. It refuses any request without the key, as the real
// one does.
func fakeCMS(t *testing.T) *cms {
	t.Helper()
	state := &cms{title: "Hello World"}
	post := `{"slug":"hello-world","title":"%s","description":"The first post.","coverImage":"/uploads/cover/hello-world+1746296252694",
		"readCount":41,"readTime":3,"publishedAt":"2026-08-29 08:52:01","updatedAt":"2026-08-30 10:00:00",
		"author":{"id":2,"name":"Furkan Baytekin"},"category":{"slug":"software","name":"Software"},"tags":[]`
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/posts", func(w http.ResponseWriter, r *http.Request) {
		state.record(&state.queries, r.URL.RawQuery)
		state.mu.Lock()
		extra := state.extra
		state.mu.Unlock()
		io.WriteString(w, `{"data":[`+state.post(post)+`},{"slug":"second","title":"Second","publishedAt":"2026-08-01 00:00:00","category":{"slug":"software","name":"Software"}}`+extra+`],"page":1,"take":6,"total":14}`)
	})
	mux.HandleFunc("GET /api/posts/views", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"hello-world":42}`)
	})
	mux.HandleFunc("GET /api/posts/{slug}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("slug") {
		case "second":
			// The list's other post, so an export builds every page.
			io.WriteString(w, `{"slug":"second","title":"Second","publishedAt":"2026-08-01 00:00:00","updatedAt":"2026-08-01 00:00:00",
				"author":{"id":2,"name":"Furkan Baytekin"},"category":{"slug":"software","name":"Software"},"tags":[],"content":"Short."}`)
			return
		case "hello-world":
		default:
			http.NotFound(w, r)
			return
		}
		io.WriteString(w, state.post(post)+`,"content":"Intro with **bold**.\n\n## First part\n\n`+"```go\\nfmt.Println(1)\\n```"+`\n\n### Detail\n\n![diagram](/uploads/content/d.png)"}`)
	})
	mux.HandleFunc("POST /api/posts/{slug}/view", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("slug") != "hello-world" {
			http.NotFound(w, r)
			return
		}
		var body struct{ UserAgent string }
		json.NewDecoder(r.Body).Decode(&body)
		state.record(&state.views, body.UserAgent)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/categories", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"categories":[{"slug":"software","name":"Software","postCount":14},{"slug":"empty","name":"Empty","postCount":0}]}`)
	})
	mux.HandleFunc("GET /api/tags", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"tags":[]}`)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/uploads/") {
			if r.URL.EscapedPath() == "/uploads/content/d.png" {
				png.Encode(w, image.NewRGBA(image.Rect(0, 0, 1280, 720)))
				return
			}
			// Like the real CMS: the literal "+" and nothing else.
			if r.URL.EscapedPath() != "/uploads/cover/hello-world+1746296252694" {
				http.NotFound(w, r)
				return
			}
			// Red, so a share card can tell it was drawn.
			cover := image.NewRGBA(image.Rect(0, 0, 1280, 720))
			draw.Draw(cover, cover.Bounds(), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
			jpeg.Encode(w, cover, nil)
			return
		}
		state.record(&state.paths, r.URL.EscapedPath())
		if r.Header.Get("x-trusted-frontend") != testKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	// Each app its own cache: apps sharing a directory, a build and a key share
	// rendered pages, and one's /_image/ names mean nothing to another's
	// opti-image.
	t.Setenv("CACHE_DIR", t.TempDir())
	t.Setenv("BLOG_API_URL", server.URL+"/api")
	t.Setenv("BLOG_TRUSTED_FRONTEND_KEY", testKey)
	t.Setenv("WEBHOOK_SECRET", testWebhookSecret)
	return state
}

func TestBlogListsPostsWithPagesAndFilters(t *testing.T) {
	body := client(t).Get("/blogs?page=2").WantStatus(http.StatusOK).Body
	for _, want := range []string{
		"<title>Blog Posts - Written with love</title>",
		`href="/blogs/hello-world">Hello World</a>`,
		"August 29, 2026",
		`href="/blogs?category=software">Software <span>14</span>`,
		`href="/blogs?page=3"`,
		`rel="prev"`,
		`"@type":"Blog"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /blogs does not contain %q", want)
		}
	}
	if strings.Contains(body, ">Empty <span>") {
		t.Error("a category with no posts is offered as a filter")
	}
	if strings.Contains(body, testKey) {
		t.Error("the trusted-frontend key reached the page")
	}
}

func TestBlogSearchIsForwarded(t *testing.T) {
	state := fakeCMS(t)
	clientOf(t, newTestApp(t)).Get("/blogs/search?search=jwt&category=software").WantStatus(http.StatusOK)
	found := false
	for _, q := range state.queries {
		found = found || (strings.Contains(q, "search=jwt") && strings.Contains(q, "category=software"))
	}
	if !found {
		t.Errorf("the CMS was asked %v, want a search for jwt in software", state.queries)
	}
}

func TestBlogPostRendersMarkdown(t *testing.T) {
	body := client(t).Get("/blogs/hello-world").WantStatus(http.StatusOK).Body
	for _, want := range []string{
		"<title>Hello World - Furkan Baytekin</title>",
		"<strong>bold</strong>",
		`<h2 id="first-part">First part</h2>`,
		`class="toc-link" href="#first-part"`,
		`class="toc-item toc-item--level-3"`,
		`background-color:#282a36`,
		`alt="diagram" width="1280" height="720"`,
		`"@type":"BlogPosting"`,
		`data-views="41"`,
		`href="/blogs/second"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /blogs/hello-world does not contain %q", want)
		}
	}
	if strings.Contains(body, "/uploads/content/d.png") {
		t.Error("an image in the post links the CMS instead of this site")
	}
	if strings.Contains(body, `class="blog-title"><a href="/blogs/hello-world"`) {
		t.Error("a post lists itself among its related posts")
	}
}

// A cover whose path has a "+" reaches the CMS as "+", not as the "&#43;"
// html/template writes in the attribute.
func TestCoverWithAPlusIsServed(t *testing.T) {
	c := client(t)
	body := c.Get("/blogs/hello-world").Body
	m := regexp.MustCompile(`blog-post-cover"><img src="(/_image/[^"]+)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("the cover is not served from /_image/:\n%s", body)
	}
	if res := c.Get(m[1]); res.Status != http.StatusOK {
		t.Errorf("GET %s = %d, want 200", m[1], res.Status)
	}
}

func TestUnknownPostIsNotFound(t *testing.T) {
	body := client(t).Get("/blogs/missing").WantStatus(http.StatusNotFound).Body
	if !strings.Contains(body, "Page not found") {
		t.Error("an unknown post is not answered with the site's not-found page")
	}
}

// A slug that could resolve to another CMS endpoint is refused before any
// request, and a CMS redirect is never followed with the key. A dot segment never
// reaches the page at all: collage redirects it to its clean spelling first, which
// for /blogs/.. is the home page.
func TestOddSlugsNeverReachTheCMS(t *testing.T) {
	state := fakeCMS(t)
	app, err := newApp(false, 0)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]int{
		"/blogs/..":              http.StatusMovedPermanently,
		"/blogs/%2E%2E":          http.StatusMovedPermanently,
		"/blogs/..%2Fcategories": http.StatusNotFound,
		"/blogs/a.b":             http.StatusNotFound,
		// The Markdown is a second way to the CMS, behind the same refusal.
		"/blogs/a.b.md":             http.StatusNotFound,
		"/blogs/..%2Fcategories.md": http.StatusNotFound,
		"/blogs/...md":              http.StatusNotFound,
	} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.URL.RawPath, req.URL.Path = target, strings.ReplaceAll(strings.ReplaceAll(target, "%2E", "."), "%2F", "/")
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("GET %s = %d, want %d", target, rec.Code, want)
		}
		if want == http.StatusMovedPermanently && rec.Header().Get("Location") != "/" {
			t.Errorf("GET %s redirects to %q, want /", target, rec.Header().Get("Location"))
		}
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, path := range state.paths {
		if strings.HasPrefix(path, "/api/posts/") && strings.Contains(path, ".") {
			t.Errorf("the CMS was asked for %q", path)
		}
	}
}

func TestViewIsCountedWithTheReadersAgent(t *testing.T) {
	state := fakeCMS(t)
	c := clientOf(t, newTestApp(t))
	req := c.Request(http.MethodPost, "/blogs/hello-world/view", nil)
	req.Header.Set("User-Agent", "Reader/1.0")
	res := c.Do(req).WantStatus(http.StatusOK)

	var answer struct{ Views int }
	if err := json.Unmarshal([]byte(res.Body), &answer); err != nil || answer.Views != 42 {
		t.Errorf("body = %q, want the new count, 42", res.Body)
	}
	if len(state.views) != 1 || state.views[0] != "Reader/1.0" {
		t.Errorf("the CMS counted %v, want one view by Reader/1.0", state.views)
	}
}

func TestFeedIsRSS(t *testing.T) {
	res := client(t).Get("/rss").WantStatus(http.StatusOK)
	body := res.Body
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/rss+xml") {
		t.Errorf("Content-Type = %q, want application/rss+xml", ct)
	}
	var feed struct {
		Items []struct {
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal([]byte(body), &feed); err != nil {
		t.Fatalf("the feed does not parse: %v", err)
	}
	if len(feed.Items) != 2 || feed.Items[0].Link != "https://furkanbaytekin.dev/blogs/hello-world" {
		t.Errorf("items = %+v, want the two posts with absolute links", feed.Items)
	}
}

func TestRobotsPointsAtTheSitemap(t *testing.T) {
	res := client(t).Get("/robots.txt")
	body := res.Body
	if res.Status != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("GET /robots.txt = %d %q, want 200 text/plain", res.Status, res.Header.Get("Content-Type"))
	}
	for _, want := range []string{"User-agent: *", "Disallow: /blogs/search", "Sitemap: https://furkanbaytekin.dev/sitemap.xml"} {
		if !strings.Contains(body, want) {
			t.Errorf("robots.txt does not contain %q:\n%s", want, body)
		}
	}
}

func TestSitemapListsPagesAndEveryPost(t *testing.T) {
	body := client(t).Get("/sitemap.xml").WantStatus(http.StatusOK).Body
	var set struct {
		URLs []struct {
			Loc     string `xml:"loc"`
			LastMod string `xml:"lastmod"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(body), &set); err != nil {
		t.Fatalf("the sitemap does not parse: %v", err)
	}
	got := map[string]string{}
	for _, u := range set.URLs {
		got[u.Loc] = u.LastMod
	}
	for _, want := range []string{"https://furkanbaytekin.dev/", "https://furkanbaytekin.dev/about", "https://furkanbaytekin.dev/blogs", "https://furkanbaytekin.dev/blogs/second"} {
		if _, ok := got[want]; !ok {
			t.Errorf("the sitemap does not list %s", want)
		}
	}
	if !strings.HasPrefix(got["https://furkanbaytekin.dev/blogs/hello-world"], "2026-08-30") {
		t.Errorf("hello-world lastmod = %q, want its update date", got["https://furkanbaytekin.dev/blogs/hello-world"])
	}
	if len(set.URLs) != 5 {
		t.Errorf("the sitemap has %d URLs, want 5: a post repeated across CMS pages is listed once", len(set.URLs))
	}
}

func TestLLMsTxt(t *testing.T) {
	body := client(t).Get("/llms.txt").WantStatus(http.StatusOK).Body
	for _, want := range []string{
		"# Furkan Baytekin\n\n> ",
		"## Pages",
		"- [About](https://furkanbaytekin.dev/about): ",
		"## Blog posts",
		"- [Hello World](https://furkanbaytekin.dev/blogs/hello-world): The first post.",
		"- [RSS feed](https://furkanbaytekin.dev/rss)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("llms.txt does not contain %q:\n%s", want, body)
		}
	}
}

func TestPagesAdvertiseTheFeed(t *testing.T) {
	body := client(t).Get("/").Body
	if n := strings.Count(body, `type="application/rss+xml"`); n != 1 {
		t.Errorf("the page links its RSS feed %d times, want once", n)
	}
	if !strings.Contains(body, `href="/rss"`) && !strings.Contains(body, `href="https://furkanbaytekin.dev/rss"`) {
		t.Error("the page's feed link does not point at /rss")
	}
}

// webhook posts body to /api/webhook as Bloggo does, with secret in its header.
func webhook(c *collagetest.Client, secret, body string) *collagetest.Response {
	req := c.Request(http.MethodPost, "/api/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}
	return c.Do(req)
}

// The whole point: a cached post stays as it was until Bloggo says it changed,
// and is rendered again the moment it does.
func TestWebhookInvalidatesTheChangedPost(t *testing.T) {
	state := fakeCMS(t)
	c := clientOf(t, newTestApp(t))

	for _, target := range []string{"/blogs/hello-world", "/blogs", "/rss"} {
		if !strings.Contains(c.Get(target).Body, "Hello World") {
			t.Fatalf("GET %s does not show the post", target)
		}
	}
	state.retitle("Hello Again")
	if !strings.Contains(c.Get("/blogs/hello-world").Body, "<title>Hello World") {
		t.Fatal("the post was not served from the cache, so this test proves nothing")
	}

	webhook(c, testWebhookSecret, `{"event":"post.updated","entity":"post","id":1,"slug":"hello-world","action":"updated","timestamp":"2026-09-24T12:00:00Z","data":{"title":"Hello Again"}}`).WantStatus(http.StatusOK)
	for _, target := range []string{"/blogs/hello-world", "/blogs", "/rss"} {
		if !strings.Contains(c.Get(target).Body, "Hello Again") {
			t.Errorf("GET %s still shows the old title after the webhook", target)
		}
	}
}

func TestWebhookRefusesAWrongSecret(t *testing.T) {
	c := client(t)
	body := `{"event":"cms.sync","entity":"cms","action":"sync"}`
	for _, secret := range []string{"", "wrong"} {
		if res := webhook(c, secret, body); res.Status != http.StatusUnauthorized {
			t.Errorf("webhook with secret %q = %d, want 401", secret, res.Status)
		}
	}
}

// Bloggo retries anything but a 2xx five times, so an event this site does not
// show is acknowledged, not refused.
func TestWebhookAcknowledgesWhatItDoesNotUse(t *testing.T) {
	res := webhook(client(t), testWebhookSecret, `{"event":"keyvalue.updated","entity":"keyvalue","action":"updated","data":{"site_name":"x"}}`).WantStatus(http.StatusOK)
	var answer struct {
		Success     bool
		Invalidated []string
	}
	if err := json.Unmarshal([]byte(res.Body), &answer); err != nil || !answer.Success || len(answer.Invalidated) != 0 {
		t.Errorf("body = %s, want success invalidating nothing", res.Body)
	}
}

// A restart forgets which keys carry which tags; the disk cache keeps them in
// each entry, so a webhook after a restart still reaches what the process
// before it cached.
func TestWebhookAfterARestartReachesWhatWasCachedBefore(t *testing.T) {
	state := fakeCMS(t)
	t.Setenv("COLLAGE_CSRF_KEY", "same-key-across-the-restart")

	clientOf(t, newTestApp(t)).Get("/blogs/hello-world")
	state.retitle("Hello Again")

	after := clientOf(t, newTestApp(t))
	if !strings.Contains(after.Get("/blogs/hello-world").Body, "<title>Hello World") {
		t.Fatal("the restarted app did not serve the earlier cache, so this test proves nothing")
	}
	webhook(after, testWebhookSecret, `{"event":"post.updated","entity":"post","slug":"hello-world","action":"updated"}`)
	if !strings.Contains(after.Get("/blogs/hello-world").Body, "Hello Again") {
		t.Error("the webhook did not reach a page cached before the restart")
	}
}

// The sitemap is made once and kept; a post published in the CMS reaches it when
// the webhook says so, not when the binary is next deployed.
func TestWebhookRemakesTheSitemap(t *testing.T) {
	state := fakeCMS(t)
	c := clientOf(t, newTestApp(t))
	if !strings.Contains(c.Get("/sitemap.xml").Body, "/blogs/hello-world</loc>") {
		t.Fatal("the sitemap does not list the post, so this test proves nothing")
	}
	state.mu.Lock()
	state.extra = `,{"slug":"third","title":"Third","publishedAt":"2026-09-01 00:00:00","category":{"slug":"software","name":"Software"}}`
	state.mu.Unlock()
	if strings.Contains(c.Get("/sitemap.xml").Body, "/blogs/third</loc>") {
		t.Fatal("the sitemap was not served from the cache, so this test proves nothing")
	}

	webhook(c, testWebhookSecret, `{"event":"post.created","entity":"post","slug":"third","action":"created"}`).WantStatus(http.StatusOK)
	if !strings.Contains(c.Get("/sitemap.xml").Body, "/blogs/third</loc>") {
		t.Error("a post published after the sitemap was made is not in it after the webhook")
	}
}
