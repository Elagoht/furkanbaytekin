// Package frontmatter writes the YAML front matter a Markdown document of this
// site opens with: a post's or a page's properties, between two "---" lines.
package frontmatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Writer collects the fields, in the order they are added, and leaves out the
// empty ones.
type Writer struct {
	b bytes.Buffer
}

// String adds name as a double-quoted scalar, unless value is empty.
func (w *Writer) String(name, value string) {
	if value != "" {
		w.field(name, Quote(value))
	}
}

// List adds name as a flow sequence of quoted strings, empty or not.
func (w *Writer) List(name string, items []string) {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(Quote(item))
	}
	b.WriteByte(']')
	w.field(name, b.String())
}

// Time adds name as a timestamp in UTC, unless t is the zero time.
func (w *Writer) Time(name string, t time.Time) {
	if !t.IsZero() {
		w.field(name, t.UTC().Format(time.RFC3339))
	}
}

// Int adds name as a number, unless n is zero or less.
func (w *Writer) Int(name string, n int) {
	if n > 0 {
		w.field(name, strconv.Itoa(n))
	}
}

func (w *Writer) field(name, value string) { fmt.Fprintf(&w.b, "%s: %s\n", name, value) }

// Bytes is the front matter, fences included, and the blank line after it.
func (w *Writer) Bytes() []byte {
	out := make([]byte, 0, w.b.Len()+9)
	out = append(out, "---\n"...)
	out = append(out, w.b.Bytes()...)
	return append(out, "---\n\n"...)
}

// Quote is s as a double-quoted YAML scalar. JSON's string escapes are a subset
// of YAML's, so a title holding a colon, a quote or a newline stays one value.
func Quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // a string always encodes
	return string(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
}
