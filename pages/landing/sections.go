package pages

import (
	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/layouts"
	fragments "furkanbaytekin/fragments/pages/landing"

	"github.com/Elagoht/collage/pkg/collage"
)

// sectionPage is the page <name>.json describes, served at path.
func sectionPage(store *content.Store, name, path string) (*collage.Page, error) {
	content, err := fragments.Sections(store, name)
	if err != nil {
		return nil, err
	}
	return collage.NewPage(name).
		WithLayouts(layouts.Master(store)).
		WithContent(content).
		WithPath("en", path).
		Static().
		Build(), nil
}
