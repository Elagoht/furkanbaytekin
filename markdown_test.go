package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// /blogs/{slug}.md is the post as Markdown: its properties as front matter,
// then the body, with its images and uploads made absolute against the CMS.
func TestPostAsMarkdown(t *testing.T) {
	fakeCMS(t)
	c := clientOf(t, newTestApp(t))
	res := c.Get("/blogs/hello-world.md").WantStatus(http.StatusOK)
	if got := res.Header.Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	cms := strings.TrimSuffix(os.Getenv("BLOG_API_URL"), "/api")
	want := `---
title: "Hello World"
description: "The first post."
slug: "hello-world"
url: "https://furkanbaytekin.dev/blogs/hello-world"
author: "Furkan Baytekin"
category: "Software"
tags: []
published: 2026-08-29T08:52:01Z
updated: 2026-08-30T10:00:00Z
readTime: 3
cover: "` + cms + `/uploads/cover/hello-world+1746296252694"
---

Intro with **bold**.
`
	want += "\n## First part\n\n```go\nfmt.Println(1)\n```\n\n### Detail\n\n![diagram](" + cms + "/uploads/content/d.png)\n"
	// Exactly: whitespace is meaning in Markdown, and nothing on the way — the
	// minimizer, a plugin's hook — may touch it.
	if res.Body != want {
		t.Errorf("GET /blogs/hello-world.md =\n%s\nwant\n%s", res.Body, want)
	}
}

// The page names its Markdown, and the two are dropped together.
func TestPostLinksItsMarkdownAndTheWebhookDropsIt(t *testing.T) {
	cms := fakeCMS(t)
	c := clientOf(t, newTestApp(t))
	page := c.Get("/blogs/hello-world").WantStatus(http.StatusOK).Body
	if !strings.Contains(page, `<link rel="alternate" type="text/markdown" href="/blogs/hello-world.md">`) {
		t.Error("the post does not link its Markdown")
	}
	c.Get("/blogs/hello-world.md").WantStatus(http.StatusOK)
	cms.retitle("Hello: Again")
	webhook(c, testWebhookSecret, `{"event":"post.updated","entity":"post","slug":"hello-world","action":"updated"}`).WantStatus(http.StatusOK)
	// A title with a colon stays one YAML value.
	if body := c.Get("/blogs/hello-world.md").Body; !strings.Contains(body, `title: "Hello: Again"`) {
		t.Errorf("after the webhook:\n%s", body)
	}
}

// The Markdown names the page as its canonical address, so a search engine
// indexes the post once; a 404 names nothing.
func TestPostMarkdownNamesThePageCanonical(t *testing.T) {
	c := client(t)
	res := c.Get("/blogs/hello-world.md").WantStatus(http.StatusOK)
	if got := res.Header.Get("Link"); got != `<https://furkanbaytekin.dev/blogs/hello-world>; rel="canonical"` {
		t.Errorf("Link = %q", got)
	}
	// From the cache too, where the response is not rendered again.
	if got := c.Get("/blogs/hello-world.md").Header.Get("Link"); got == "" {
		t.Error("a cached Markdown response has no Link")
	}
	for _, path := range []string{"/blogs/missing.md", "/blogs/hello-world", "/llms.txt"} {
		if got := c.Get(path).Header.Get("Link"); strings.Contains(got, "canonical") {
			t.Errorf("GET %s: Link = %q", path, got)
		}
	}
}

func TestUnknownPostAsMarkdownIsNotFound(t *testing.T) {
	c := client(t)
	c.Get("/blogs/missing.md").WantStatus(http.StatusNotFound)
	// A page and its Markdown, not a slug that ends in ".md".
	c.Get("/blogs/.md").WantStatus(http.StatusNotFound)
}

// An export writes each post's Markdown, and the about page's, beside the page.
func TestExportWritesPostMarkdown(t *testing.T) {
	fakeCMS(t)
	out := t.TempDir()
	if err := staticBuild(newTestApp(t), out, false); err != nil {
		t.Fatalf("staticBuild() = %v", err)
	}
	for _, file := range []string{"blogs/hello-world.md", "blogs/second.md", "blogs/hello-world/index.html", "about.md"} {
		body, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(file)))
		if err != nil {
			t.Errorf("the export has no %s", file)
			continue
		}
		if strings.HasSuffix(file, ".md") && !strings.HasPrefix(string(body), "---\ntitle: ") {
			t.Errorf("%s does not begin with its front matter:\n%s", file, body)
		}
	}
}

// /about.md is /about as Markdown: front matter, the person's name, then each
// section written by its type, with no trace of the separators between them.
func TestAboutAsMarkdown(t *testing.T) {
	c := client(t)
	res := c.Get("/about.md").WantStatus(http.StatusOK)
	if got := res.Header.Get("Content-Type"); got != "text/markdown; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := res.Header.Get("Link"); got != `<https://furkanbaytekin.dev/about>; rel="canonical"` {
		t.Errorf("Link = %q", got)
	}
	head := `---
title: "About - Furkan Baytekin"
description: "Product-oriented developer who owns end-to-end systems from architecture to production."
url: "https://furkanbaytekin.dev/about"
author: "Furkan Baytekin"
---

# Furkan Baytekin

## About

![Furkan Baytekin](https://furkanbaytekin.dev/`
	if !strings.HasPrefix(res.Body, head) {
		t.Errorf("GET /about.md begins\n%s", res.Body[:min(len(res.Body), 600)])
	}
	for _, want := range []string{
		"\n## Languages\n\nGo, TypeScript, JavaScript,",
		"\n## Tech Stack\n\n- **Frontend**: Next.js, React.js,",
		"\n## What I Know\n\n- System Design & Architecture\n",
		"\n## Experience\n\n### Developer, UNOG\n\n*Sep 2026 - Present*\n\n- Developed open source",
		"\n- Architected the whole platform (**Go backend**,",
		"\n## Education\n\n- **Anadolu University**, Web Design and Development (A.D.), 2021 - 2023\n",
	} {
		if !strings.Contains(res.Body, want) {
			t.Errorf("GET /about.md lacks %q", want)
		}
	}
	if n := strings.Count(res.Body, "\n---\n"); n != 1 {
		t.Errorf("GET /about.md has %d \"---\" lines past the first, want 1: the front matter's end", n)
	}
	if strings.Contains(res.Body, "\n\n\n") {
		t.Error("GET /about.md has more than one blank line in a row")
	}
}

// /about links its Markdown; a page without one links none.
func TestSectionPagesLinkTheirMarkdown(t *testing.T) {
	c := client(t)
	link := `<link rel="alternate" type="text/markdown" href="/about.md">`
	if !strings.Contains(c.Get("/about").Body, link) {
		t.Error("/about does not link its Markdown")
	}
	if strings.Contains(c.Get("/").Body, `type="text/markdown"`) {
		t.Error("/ links a Markdown it does not have")
	}
}
