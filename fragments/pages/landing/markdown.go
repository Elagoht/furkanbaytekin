package fragments

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/sections"
	"furkanbaytekin/frontmatter"

	"github.com/Elagoht/collage/pkg/collage"
)

// Markdown is the page <name>.json describes, at path, as Markdown: its title,
// description and address as front matter, then its sections in order, each
// written by its type.
func Markdown(store *content.Store, name, path string) collage.DocumentHandlerFunc {
	return func(context.Context, *collage.RenderContext) ([]byte, []string, error) {
		page, err := store.Page(name)
		if err != nil {
			return nil, nil, err
		}
		site, err := store.Site()
		if err != nil {
			return nil, nil, err
		}
		body, err := sections.WriteSections(page.Sections, site.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", name, err)
		}

		var fm frontmatter.Writer
		fm.String("title", page.SEO.Title)
		fm.String("description", page.SEO.Description)
		fm.String("url", site.URL+path)
		fm.String("author", site.Person.Name)
		out := bytes.NewBuffer(fm.Bytes())
		// A hero is the page's heading; without one, the person the site is
		// about is: the sections' own headings — "About" — sit under it.
		if len(page.Sections) == 0 || page.Sections[0].Type != "hero" {
			fmt.Fprintf(out, "# %s\n\n", site.Person.Name)
		}
		out.WriteString(body)
		return out.Bytes(), nil, nil
	}
}

// MarkdownRoute is the name of the Markdown document of page name: "about-md".
func MarkdownRoute(name string) string { return name + "-md" }

// hoistMarkdown links the page's Markdown from its head, when it has one.
func hoistMarkdown(rc *collage.RenderContext, name string) {
	href, err := rc.URL(MarkdownRoute(name), nil)
	if err != nil {
		return // no document by that name: this page has no Markdown
	}
	rc.Hoist("head", "link:alternate:markdown", template.HTML(
		`<link rel="alternate" type="text/markdown" href="`+template.HTMLEscapeString(href)+`">`))
}
