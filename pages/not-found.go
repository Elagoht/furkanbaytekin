package pages

import (
	"context"

	"furkanbaytekin/content"
	"furkanbaytekin/fragments/layouts"
	"furkanbaytekin/fragments/seo"

	"github.com/Elagoht/collage/pkg/collage"
)

func NotFoundPage(store *content.Store) *collage.Page {
	fragment := collage.NewFragment(
		"not-found-content",
		"pages/404.html",
	).WithDataHandler(collage.DataHandler(
		func(_ context.Context, rc *collage.RenderContext) (content.NotFound, []string, error) {
			site, err := store.Site()
			if err != nil {
				return content.NotFound{}, nil, err
			}
			seo.Apply(rc, content.SEO{Title: site.NotFound.Title, Description: site.NotFound.Message})
			return site.NotFound, nil, nil
		}),
	).Build()

	return collage.NewPage("not-found").
		WithLayouts(layouts.Layout(store)).
		WithContent(fragment).
		Dynamic().
		Build()
}
