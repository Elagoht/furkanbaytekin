package sections

import (
	"strings"
	"testing"

	"furkanbaytekin/data/content"
)

// Every page the site has writes as Markdown: the home page's hero too, which
// is its heading, with its titles, bio and links.
func TestEveryPageWritesAsMarkdown(t *testing.T) {
	store := content.NewStore(content.Embedded())
	for _, name := range []string{"home", "about"} {
		page, err := store.Page(name)
		if err != nil {
			t.Fatal(err)
		}
		md, err := WriteSections(page.Sections, "https://site.example")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(md, "](/") {
			t.Errorf("%s has a relative link or image:\n%s", name, md)
		}
		if name == "home" && !strings.HasPrefix(md, "# ") {
			t.Errorf("home does not open with its hero's heading:\n%s", md)
		}
	}
}

func TestMarkdownPathsAreAbsolute(t *testing.T) {
	w := &Markdown{base: "https://site.example"}
	for href, want := range map[string]string{
		"/about":                 "[a](https://site.example/about)",
		"https://x.example/y":    "[a](https://x.example/y)",
		"//evil.example/":        "[a](//evil.example/)",
		"mailto:me@site.example": "[a](mailto:me@site.example)",
		"":                       "a",
	} {
		if got := w.link("a", href); got != want {
			t.Errorf("link(%q) = %q, want %q", href, got, want)
		}
	}
	if got := w.link("[x]", "/y"); got != `[\[x\]](https://site.example/y)` {
		t.Errorf("brackets in a label: %q", got)
	}
}

// Two lists in a row are two lists: a renderer reads the same marker after a
// blank line as the same list.
func TestListsInARowStayApart(t *testing.T) {
	w := &Markdown{}
	w.list([]string{"a"})
	w.list([]string{"b"})
	w.list([]string{"c"})
	w.heading(2, "H")
	w.list([]string{"d"})
	if got, want := w.b.String(), "- a\n\n* b\n\n- c\n\n## H\n\n- d\n"; got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
