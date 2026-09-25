package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"furkanbaytekin/fragments/sections"
)

// Testing a collage application needs no server and no port: app.Handler() is an
// ordinary http.Handler, so net/http/httptest drives it directly. Everything below
// goes through the same newApp main uses, so what these tests exercise is the site
// that actually runs rather than a second wiring that can drift from it.

func handler(t *testing.T) http.Handler {
	t.Helper()
	fakeCMS(t)
	app, err := newApp(false, 0)
	if err != nil {
		t.Fatalf("newApp() = %v, want nil", err)
	}
	return app.Handler()
}

// get returns the response to a GET of target, and its body.
func get(t *testing.T, h http.Handler, target string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec, rec.Body.String()
}

func TestPagesRender(t *testing.T) {
	h := handler(t)
	for target, want := range map[string]string{
		"/":      "Furkan Baytekin",
		"/about": "Product-oriented developer",
	} {
		rec, body := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, rec.Code)
			continue
		}
		if !strings.Contains(body, want) {
			t.Errorf("GET %s does not contain %q", target, want)
		}
		// Counted, not merely found: a layout writing one and a page hoisting
		// another is two titles, which is a page that looks fine and is not.
		if n := strings.Count(body, "<title>"); n != 1 {
			t.Errorf("GET %s has %d titles, want exactly 1", target, n)
		}
	}
}

func TestStylesheetIsContentAddressed(t *testing.T) {
	_, body := get(t, handler(t), "/")

	link := regexp.MustCompile(`href="(/static/site\.[0-9a-f]+\.css)"`).FindStringSubmatch(body)
	if link == nil {
		t.Fatalf("the page does not link a content-addressed stylesheet:\n%s", body)
	}

	rec, css := get(t, handler(t), link[1])
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", link[1], rec.Code)
	}
	if css == "" {
		t.Error("the stylesheet served no bytes")
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}
}

// The home page is home.json: every section it lists renders, in its order,
// with the page's own title and description in place of the site's defaults.
func TestHomeRendersItsSectionsInOrder(t *testing.T) {
	rec, body := get(t, handler(t), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}

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
	_, body := get(t, handler(t), "/")
	want := "Copyleft © " + strconv.Itoa(time.Now().Year()) + " All Wrongs Reversed"
	if !strings.Contains(body, want) {
		t.Errorf("GET / does not contain %q", want)
	}
}

func TestAboutRendersItsSections(t *testing.T) {
	rec, body := get(t, handler(t), "/about")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /about = %d, want 200", rec.Code)
	}

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
	_, body := get(t, handler(t), "/")

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
	_, body := get(t, handler(t), "/")

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
	_, body := get(t, handler(t), "/")
	if strings.Contains(body, "\n  ") {
		t.Error("the page still carries its template indentation")
	}
}

// The favicons are this site's own files, not links to another host.
func TestFaviconsAreServedLocally(t *testing.T) {
	_, body := get(t, handler(t), "/")

	icons := regexp.MustCompile(`rel="(?:icon|apple-touch-icon)" href="([^"]+)"`).FindAllStringSubmatch(body, -1)
	if len(icons) == 0 {
		t.Fatal("the page links no icons")
	}
	h := handler(t)
	for _, icon := range icons {
		if rec, _ := get(t, h, icon[1]); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", icon[1], rec.Code)
		}
	}
}

// The health check answers without rendering a template, which is the point of it:
// it cannot start failing because a page did.
func TestHealthCheck(t *testing.T) {
	rec, body := get(t, handler(t), "/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", rec.Code)
	}
	var health struct{ Status string }
	if err := json.Unmarshal([]byte(body), &health); err != nil || health.Status != "ok" {
		t.Errorf("body = %q, want a status of ok", body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// An address that is nothing answers 404 with this site's own page. A static
// export writes the same page as 404.html.
func TestNotFoundPage(t *testing.T) {
	rec, body := get(t, handler(t), "/there-is-nothing-here")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !strings.Contains(body, "Page not found") || !strings.Contains(body, "<title>Page Not Found</title>") {
		t.Errorf("body = %q, want this site's own not-found page", body)
	}
}

// A method no page answers is a 405 naming what the URL does accept.
func TestUnsupportedMethodIsRefused(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	handler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Errorf("Allow = %q, want it to name what the URL accepts", allow)
	}
}
