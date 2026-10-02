package pages

import (
	"net/http"
	"strings"
)

// MarkdownCanonical names a post's page as the canonical address of its
// Markdown, in a Link header on every /blogs/{slug}.md that answers 200: the
// Markdown is the same post, and a search engine indexing both would split the
// post between two URLs. It stays reachable — llms.txt points readers to it.
//
// A middleware, because collage gives a document no headers of its own. It
// matches what PostMarkdown's pattern does, and a static export, which has no
// response headers, goes without.
func MarkdownCanonical(siteURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page, ok := markdownPage(r.URL.EscapedPath())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(&canonicalWriter{ResponseWriter: w, link: "<" + siteURL + page + `>; rel="canonical"`}, r)
		})
	}
}

// markdownPage is the page path of a post's Markdown path, escaped as it came:
// "/blogs/hello.md" is "/blogs/hello".
func markdownPage(escaped string) (string, bool) {
	slug, ok := strings.CutPrefix(escaped, "/blogs/")
	if !ok || strings.Contains(slug, "/") {
		return "", false
	}
	slug, ok = strings.CutSuffix(slug, ".md")
	if !ok || slug == "" {
		return "", false
	}
	return "/blogs/" + slug, true
}

// canonicalWriter adds the Link header to a 200, and to nothing else: a 404's
// canonical would name a page that is not there either.
type canonicalWriter struct {
	http.ResponseWriter
	link    string
	written bool
}

func (w *canonicalWriter) WriteHeader(status int) {
	if !w.written {
		w.written = true
		if status == http.StatusOK {
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
