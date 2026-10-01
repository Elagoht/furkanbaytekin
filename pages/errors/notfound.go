package pages

import (
	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/layouts"
	fragments "furkanbaytekin/fragments/pages/errors"

	"github.com/Elagoht/collage/pkg/collage"
)

// NotFound is the page every unknown address answers with. It has no path: it
// is reached by failing to match, and "collage export" writes it as 404.html.
func NotFound(store *content.Store) *collage.Page {
	return collage.NewPage("not-found").
		WithLayouts(layouts.Master(store)).
		WithContent(fragments.NotFound(store)).
		Dynamic().
		Build()
}
