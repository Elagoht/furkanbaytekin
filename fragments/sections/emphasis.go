package sections

import (
	"html/template"
	"strings"
)

// Emphasis escapes text and turns each **pair** into <b>…</b>. An unpaired
// marker is left as written.
func Emphasis(text string) template.HTML {
	parts := strings.Split(text, "**")
	if len(parts)%2 == 0 {
		return template.HTML(template.HTMLEscapeString(text))
	}
	var out strings.Builder
	for i, part := range parts {
		escaped := template.HTMLEscapeString(part)
		if i%2 == 1 {
			out.WriteString("<b>" + escaped + "</b>")
		} else {
			out.WriteString(escaped)
		}
	}
	return template.HTML(out.String())
}
