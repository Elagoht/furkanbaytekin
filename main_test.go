package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"furkanbaytekin/fragments/sections"

	"github.com/Elagoht/collage/pkg/collage"
	"github.com/Elagoht/collage/pkg/collagetest"
)

// Testing a collage application needs no server and no port: app.Handler() is an
// ordinary http.Handler, and collagetest drives it the way a browser does. Everything
// below goes through the same newApp main uses, so what these tests exercise is the
// site that actually runs rather than a second wiring that can drift from it.

// client is a browser of its own on a fresh copy of the site, in front of a fake CMS.
func client(t *testing.T) *collagetest.Client {
	t.Helper()
	fakeCMS(t)
	return clientOf(t, newTestApp(t))
}

// newTestApp builds the application main builds, against whatever CMS the test set up.
func newTestApp(t *testing.T) *collage.App {
	t.Helper()
	app, err := newApp(false, 0)
	if err != nil {
		t.Fatalf("newApp() = %v, want nil", err)
	}
	return app
}

func clientOf(t *testing.T, app *collage.App) *collagetest.Client {
	t.Helper()
	return collagetest.New(t, app.Handler())
}

func TestPagesRender(t *testing.T) {
	c := client(t)
	for target, want := range map[string]string{
		"/":      "Furkan Baytekin",
		"/about": "Product-oriented developer",
	} {
		res := c.Get(target)
		if res.Status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, res.Status)
			continue
		}
		if !strings.Contains(res.Body, want) {
			t.Errorf("GET %s does not contain %q", target, want)
		}
		// Counted, not merely found: a layout writing one and a page hoisting
		// another is two titles, which is a page that looks fine and is not.
		if n := strings.Count(res.Body, "<title>"); n != 1 {
			t.Errorf("GET %s has %d titles, want exactly 1", target, n)
		}
	}
}

func TestStylesheetIsContentAddressed(t *testing.T) {
	c := client(t)
	page := c.Get("/")

	link := regexp.MustCompile(`href="(/static/site\.[0-9a-f]+\.css)"`).FindStringSubmatch(page.Body)
	if link == nil {
		t.Fatalf("the page does not link a content-addressed stylesheet:\n%s", page.Body)
	}

	css := c.Get(link[1]).WantStatus(http.StatusOK)
	if css.Body == "" {
		t.Error("the stylesheet served no bytes")
	}
	if cc := css.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}
}

// The home page is home.json: every section it lists renders, in its order,
// with the page's own title and description in place of the site's defaults.
func TestHomeRendersItsSectionsInOrder(t *testing.T) {
	body := client(t).Get("/").WantStatus(http.StatusOK).Body

	last := -1
	for _, want := range []string{`class="hero"`, `class="section-sep"`, `class="hobbies"`, `class="stats"`} {
		at := strings.Index(body[last+1:], want)
		if at < 0 {
			t.Fatalf("GET / has no %s after byte %d", want, last)
		}
		last += 1 + at
	}
	for _, want := range []string{
		"<title>Furkan Baytekin | The Open Sourcerer - Fullstack Developer</title>",
		`<link rel="canonical" href="https://furkanbaytekin.dev/">`,
		"Collecting Vinyl Records",
		"YouTube Videos",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET / does not contain %q", want)
		}
	}
	if n := strings.Count(body, `name="description"`); n != 1 {
		t.Errorf("GET / has %d descriptions, want exactly 1", n)
	}
}

func TestCopyrightCarriesTheCurrentYear(t *testing.T) {
	body := client(t).Get("/").Body
	want := "Copyleft © " + strconv.Itoa(time.Now().Year()) + " All Wrongs Reversed"
	if !strings.Contains(body, want) {
		t.Errorf("GET / does not contain %q", want)
	}
}

func TestAboutRendersItsSections(t *testing.T) {
	body := client(t).Get("/about").WantStatus(http.StatusOK).Body

	last := -1
	for _, want := range []string{`class="about-page"`, `class="languages"`, `class="stack"`, `class="expertise"`, `class="experience"`, `class="education"`} {
		at := strings.Index(body[last+1:], want)
		if at < 0 {
			t.Fatalf("GET /about has no %s after byte %d", want, last)
		}
		last += 1 + at
	}
	for _, want := range []string{
		"<title>About - Furkan Baytekin</title>",
		"<b>modular monolith</b>",
		"Gazi University",
		`"knowsAbout":[`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("GET /about does not contain %q", want)
		}
	}
	if avatar := regexp.MustCompile(`<img class="about-avatar" src="(/_image/[^"]+)"`).FindStringSubmatch(body); avatar == nil {
		t.Error("the about avatar is not served from /_image/")
	}
}

func TestEmphasisEscapesAndBolds(t *testing.T) {
	for in, want := range map[string]string{
		"a **b** c":       "a <b>b</b> c",
		"<x> **y**":       "&lt;x&gt; <b>y</b>",
		"unpaired ** one": "unpaired ** one",
	} {
		if got := string(sections.Emphasis(in)); got != want {
			t.Errorf("Emphasis(%q) = %q, want %q", in, got, want)
		}
	}
}

// jsonld: the home page describes its person, and the plugin adds the site.
func TestHomeHasStructuredData(t *testing.T) {
	body := client(t).Get("/").Body

	blocks := regexp.MustCompile(`<script type="application/ld\+json">(.*?)</script>`).FindAllStringSubmatch(body, -1)
	types := map[string]bool{}
	for _, block := range blocks {
		var node struct {
			Type string `json:"@type"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(block[1]), &node); err != nil {
			t.Fatalf("a JSON-LD block does not parse: %v\n%s", err, block[1])
		}
		types[node.Type] = true
	}
	for _, want := range []string{"Person", "WebSite"} {
		if !types[want] {
			t.Errorf("no %s node among %v", want, types)
		}
	}
}

// opti-image: the avatar declares its size, so its src is replaced with a copy
// this server produces and serves under /_image/.
func TestAvatarIsServedByThisSite(t *testing.T) {
	body := client(t).Get("/").Body

	avatar := regexp.MustCompile(`<img class="hero-avatar" src="([^"]+)"`).FindStringSubmatch(body)
	if avatar == nil {
		t.Fatalf("no avatar on the page:\n%s", body)
	}
	if !strings.HasPrefix(avatar[1], "/_image/") {
		t.Errorf("avatar src = %q, want a /_image/ URL", avatar[1])
	}
}

// minimizer: indentation between tags is collapsed.
func TestPagesAreMinified(t *testing.T) {
	body := client(t).Get("/").Body
	if strings.Contains(body, "\n  ") {
		t.Error("the page still carries its template indentation")
	}
}

// The favicons are this site's own files, not links to another host.
func TestFaviconsAreServedLocally(t *testing.T) {
	c := client(t)
	icons := regexp.MustCompile(`rel="(?:icon|apple-touch-icon)" href="([^"]+)"`).FindAllStringSubmatch(c.Get("/").Body, -1)
	if len(icons) == 0 {
		t.Fatal("the page links no icons")
	}
	for _, icon := range icons {
		if res := c.Get(icon[1]); res.Status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", icon[1], res.Status)
		}
	}
}

// The health check answers without rendering a template, which is the point of it:
// it cannot start failing because a page did.
func TestHealthCheck(t *testing.T) {
	res := client(t).Get("/healthz").WantStatus(http.StatusOK)

	var health struct{ Status string }
	if err := json.Unmarshal([]byte(res.Body), &health); err != nil || health.Status != "ok" {
		t.Errorf("body = %q, want a status of ok", res.Body)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// An address that is nothing answers 404 with this site's own page. A static
// export writes the same page as 404.html.
func TestNotFoundPage(t *testing.T) {
	body := client(t).Get("/there-is-nothing-here").WantStatus(http.StatusNotFound).Body

	if !strings.Contains(body, "Page not found") || !strings.Contains(body, "<title>Page Not Found</title>") {
		t.Errorf("body = %q, want this site's own not-found page", body)
	}
}

// A method no page answers is a 405 naming what the URL does accept.
func TestUnsupportedMethodIsRefused(t *testing.T) {
	c := client(t)
	res := c.Do(c.Request(http.MethodDelete, "/", nil)).WantStatus(http.StatusMethodNotAllowed)

	if allow := res.Header.Get("Allow"); !strings.Contains(allow, "GET") {
		t.Errorf("Allow = %q, want it to name what the URL accepts", allow)
	}
}

// Every link a template builds by name — pageURL, actionURL — names a route that
// exists, checked without rendering anything, as "collage check" does.
func TestLinksResolve(t *testing.T) {
	fakeCMS(t)
	if findings := newTestApp(t).Check(); len(findings) > 0 {
		t.Errorf("broken links: %v", findings)
	}
}
