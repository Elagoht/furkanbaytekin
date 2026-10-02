package fragments

import (
	"encoding/json"
	"testing"
)

// A quoted value is one YAML double-quoted scalar whatever it holds: JSON's
// escapes are a subset of YAML's, so reading it back as JSON is reading it back.
func TestQuoteKeepsAValueWhole(t *testing.T) {
	for _, value := range []string{`Hello: "Again"`, "two\nlines", `back\slash`, "<b>&</b>", "# not a comment", "çay"} {
		var back string
		if err := json.Unmarshal([]byte(quote(value)), &back); err != nil || back != value {
			t.Errorf("quote(%q) = %s, reads back as %q (%v)", value, quote(value), back, err)
		}
	}
	if got := quote("<b>&</b>"); got != `"<b>&</b>"` {
		t.Errorf("quote escapes HTML: %s", got)
	}
	if got := quoteList([]string{"go", "web"}); got != `["go", "web"]` {
		t.Errorf("quoteList = %s", got)
	}
}
