package pages

import (
	"furkanbaytekin/data/content"
	fragments "furkanbaytekin/fragments/pages/landing"

	"github.com/Elagoht/collage/pkg/collage"
)

// AboutMarkdown is /about.md: /about as Markdown, its sections written by type,
// with its title, description and address as front matter. Static, as the page
// is: both are about.json.
func AboutMarkdown(store *content.Store) *collage.Document {
	return sectionMarkdown(store, "about", "/about")
}

// sectionMarkdown is the Markdown of the section page <name>.json describes at
// path, served at path + ".md". The page links it from its head.
func sectionMarkdown(store *content.Store, name, path string) *collage.Document {
	return collage.NewDocument(fragments.MarkdownRoute(name), "text/markdown; charset=utf-8").
		WithPath("en", path+".md").
		WithHandler(fragments.Markdown(store, name, path)).
		Static().
		Build()
}
