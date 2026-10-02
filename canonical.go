package main

import (
	"net/http"
	"strings"
)

// markdownCanonical names a page as the canonical address of its Markdown, in a
// Link header: /blogs/hello.md names /blogs/hello, /about.md names /about. The
// Markdown is the same page, and a search engine indexing both would split it
// between two URLs; it stays reachable, and llms.txt points readers to it.
//
// Only a 200 in text/markdown gets one — a 404 would name a page that is not
// there either, and a file that merely ends in .md is not a page's Markdown.
//
// A middleware, because collage gives a document no headers of its own; a
// static export, which has no response headers, goes without.
func markdownCanonical(siteURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page, ok := strings.CutSuffix(r.URL.EscapedPath(), ".md")
			if !ok || page == "" || strings.HasSuffix(page, "/") {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(&canonicalWriter{ResponseWriter: w, link: "<" + siteURL + page + `>; rel="canonical"`}, r)
		})
	}
}

// canonicalWriter adds the Link header when the response turns out to be a
// page's Markdown.
type canonicalWriter struct {
	http.ResponseWriter
	link    string
	written bool
}

func (w *canonicalWriter) WriteHeader(status int) {
	if !w.written {
		w.written = true
		if status == http.StatusOK && strings.HasPrefix(w.Header().Get("Content-Type"), "text/markdown") {
			w.Header().Add("Link", w.link)
		}
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *canonicalWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the writer underneath.
func (w *canonicalWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
