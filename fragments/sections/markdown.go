package sections

import (
	"fmt"
	"strings"

	"furkanbaytekin/data/content"
)

// Markdown is a section page being written as Markdown: each section adds its
// block, separated from the last by a blank line.
type Markdown struct {
	b strings.Builder
	// base is the site's origin, which an image or link path is made absolute
	// against: the Markdown is read away from the site.
	base string
	// bullet is the marker of the last list written, or "" when the last block
	// was not a list.
	bullet string
}

// WriteSections writes sections as Markdown, in order, with image and link
// paths absolute against base.
func WriteSections(sections []content.Section, base string) (string, error) {
	w := &Markdown{base: base}
	for i, section := range sections {
		k, ok := kinds[section.Type]
		if !ok {
			return "", fmt.Errorf("section %d has unknown type %q", i, section.Type)
		}
		if err := k.markdown(section.Data, w); err != nil {
			return "", fmt.Errorf("section %d: %w", i, err)
		}
	}
	return w.b.String(), nil
}

// block starts a block: a blank line after the one before it.
func (w *Markdown) block() {
	if w.b.Len() > 0 {
		w.b.WriteString("\n")
	}
	w.bullet = ""
}

func (w *Markdown) heading(level int, text string) {
	if text == "" {
		return
	}
	w.block()
	fmt.Fprintf(&w.b, "%s %s\n", strings.Repeat("#", level), oneLine(text))
}

func (w *Markdown) paragraph(text string) {
	if text = strings.TrimSpace(text); text == "" {
		return
	}
	w.block()
	w.b.WriteString(text + "\n")
}

func (w *Markdown) list(items []string) {
	if len(items) == 0 {
		return
	}
	// Two lists one after the other, with the same marker, are one list in
	// Markdown — a blank line between them does not part them — so a list right
	// after a list takes the other marker: the stats after the hobbies.
	bullet := "-"
	if w.bullet == "-" {
		bullet = "*"
	}
	w.block()
	for _, item := range items {
		w.b.WriteString(bullet + " " + oneLine(item) + "\n")
	}
	w.bullet = bullet
}

func (w *Markdown) image(img image) {
	if img.Src == "" {
		return
	}
	w.block()
	fmt.Fprintf(&w.b, "![%s](%s)\n", label(img.Alt), w.url(img.Src))
}

// link is a Markdown link to href, absolute against the site when it is a path.
func (w *Markdown) link(text, href string) string {
	if href == "" {
		return oneLine(text)
	}
	return "[" + label(text) + "](" + w.url(href) + ")"
}

func (w *Markdown) url(href string) string {
	if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
		return w.base + href
	}
	return href
}

// label is text fit for a link's or an image's brackets.
func label(text string) string {
	return strings.NewReplacer(`[`, `\[`, `]`, `\]`).Replace(oneLine(text))
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func (v heroView) markdown(w *Markdown) {
	w.heading(1, v.Name)
	titles := make([]string, 0, len(v.Titles))
	for _, t := range v.Titles {
		titles = append(titles, w.link(t.Label, t.Href))
	}
	w.paragraph(strings.Join(titles, " · "))
	w.paragraph(v.Bio)
	links := make([]string, 0, len(v.Actions)+len(v.Socials))
	for _, a := range v.Actions {
		links = append(links, w.link(a.Label, a.Href))
	}
	for _, s := range v.Socials {
		links = append(links, w.link(s.Label, s.Href))
	}
	w.list(links)
}

func (v hobbiesView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	items := make([]string, 0, len(v.Items))
	for _, h := range v.Items {
		items = append(items, joinNonEmpty(": ", bold(h.Title), h.Description))
	}
	w.list(items)
}

func (v statsView) markdown(w *Markdown) {
	items := make([]string, 0, len(v.Items))
	for _, s := range v.Items {
		items = append(items, w.link(joinNonEmpty(" ", s.Value, s.Label), s.Href))
	}
	w.list(items)
}

func (v profileView) markdown(w *Markdown) {
	w.heading(2, v.Heading)
	w.image(v.Image)
	w.paragraph(v.Text)
}

func (v chipsView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	w.paragraph(strings.Join(v.Items, ", "))
}

func (v stackView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	items := make([]string, 0, len(v.Groups))
	for _, g := range v.Groups {
		items = append(items, joinNonEmpty(": ", bold(g.Title), strings.Join(g.Items, ", ")))
	}
	w.list(items)
}

func (v expertiseView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	w.list(v.Items)
}

func (v experienceView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	for _, j := range v.Items {
		w.heading(3, joinNonEmpty(", ", j.Title, j.Company))
		if j.Date != "" {
			w.paragraph("*" + oneLine(j.Date) + "*")
		}
		// The points mark phrases **like this** already, which is Markdown.
		w.list(j.Points)
	}
}

func (v educationView) markdown(w *Markdown) {
	w.heading(2, v.Label)
	items := make([]string, 0, len(v.Items))
	for _, s := range v.Items {
		items = append(items, joinNonEmpty(", ", bold(s.School), s.Programme, s.Date))
	}
	w.list(items)
}

// bold is text in **bold**, or nothing for no text.
func bold(text string) string {
	if text = oneLine(text); text == "" {
		return ""
	}
	return "**" + text + "**"
}

// joinNonEmpty joins the parts that are not empty with sep.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = oneLine(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
