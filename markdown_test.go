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

func TestUnknownPostAsMarkdownIsNotFound(t *testing.T) {
	c := client(t)
	c.Get("/blogs/missing.md").WantStatus(http.StatusNotFound)
	// A page and its Markdown, not a slug that ends in ".md".
	c.Get("/blogs/.md").WantStatus(http.StatusNotFound)
}

// An export writes each post's Markdown beside its page.
func TestExportWritesPostMarkdown(t *testing.T) {
	fakeCMS(t)
	out := t.TempDir()
	if err := staticBuild(newTestApp(t), out, false); err != nil {
		t.Fatalf("staticBuild() = %v", err)
	}
	for _, file := range []string{"blogs/hello-world.md", "blogs/second.md", "blogs/hello-world/index.html"} {
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
