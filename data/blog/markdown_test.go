package blog

import (
	"strings"
	"testing"
)

// Source makes a post's images and uploads absolute, where the source spells
// them and nowhere else: not in a code block, not a link to another site.
func TestSourceMakesCMSPathsAbsolute(t *testing.T) {
	client, err := New("https://cms.example/api", "key")
	if err != nil {
		t.Fatal(err)
	}
	src := "![a](/uploads/a.png) and [file](/uploads/f.pdf) and [x](https://x.example/uploads/y)\n\n" +
		"[ref]: /uploads/r.png\n\n![r][ref]\n\n```md\n![a](/uploads/a.png)\n```\n\n" +
		"Inline `![a](/uploads/a.png)` too.\n\n    ![a](/uploads/a.png)\n"
	got, err := NewRenderer(client).Source(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"![a](https://cms.example/uploads/a.png) and [file](https://cms.example/uploads/f.pdf) and [x](https://x.example/uploads/y)",
		"[ref]: https://cms.example/uploads/r.png",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Source() lacks %q:\n%s", want, got)
		}
	}
	// Code shows Markdown rather than being it: a fenced block, an inline span and
	// an indented block keep the relative path.
	if n := strings.Count(got, "](/uploads/a.png)"); n != 3 {
		t.Errorf("Source() left %d relative copies in code, want 3:\n%s", n, got)
	}
}
