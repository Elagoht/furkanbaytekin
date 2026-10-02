package pages

import (
	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/layouts"
	fragments "furkanbaytekin/fragments/pages/landing"

	"github.com/Elagoht/collage/pkg/collage"
)

// sectionPage is the page <name>.json describes, served at path, with
// permanent redirects from each pair's first path to its second.
func sectionPage(store *content.Store, name, path string, redirects ...[2]string) (*collage.Page, error) {
	content, err := fragments.Sections(store, name)
	if err != nil {
		return nil, err
	}
	b := collage.NewPage(name).
		WithLayouts(layouts.Master(store)).
		WithContent(content).
		WithPath("en", path).
		Static()
	for _, r := range redirects {
		b = b.WithPermanentRedirect(r[0], r[1])
	}
	return b.Build(), nil
}
