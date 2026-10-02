package frontmatter

import (
	"encoding/json"
	"testing"
	"time"
)

// A quoted value is one YAML double-quoted scalar whatever it holds: JSON's
// escapes are a subset of YAML's, so reading it back as JSON is reading it back.
func TestQuoteKeepsAValueWhole(t *testing.T) {
	for _, value := range []string{`Hello: "Again"`, "two\nlines", `back\slash`, "<b>&</b>", "# not a comment", "çay"} {
		var back string
		if err := json.Unmarshal([]byte(Quote(value)), &back); err != nil || back != value {
			t.Errorf("Quote(%q) = %s, reads back as %q (%v)", value, Quote(value), back, err)
		}
	}
	if got := Quote("<b>&</b>"); got != `"<b>&</b>"` {
		t.Errorf("Quote escapes HTML: %s", got)
	}
}

func TestWriterLeavesOutEmptyFields(t *testing.T) {
	var w Writer
	w.String("title", "A")
	w.String("description", "")
	w.List("tags", []string{"go", "web"})
	w.List("none", nil)
	w.Time("published", time.Date(2026, 8, 29, 8, 52, 1, 0, time.UTC))
	w.Time("updated", time.Time{})
	w.Int("readTime", 3)
	w.Int("zero", 0)
	want := "---\ntitle: \"A\"\ntags: [\"go\", \"web\"]\nnone: []\npublished: 2026-08-29T08:52:01Z\nreadTime: 3\n---\n\n"
	if got := string(w.Bytes()); got != want {
		t.Errorf("Bytes() =\n%s\nwant\n%s", got, want)
	}
}
