package fragments

import (
	"context"

	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/seo"

	"github.com/Elagoht/collage/pkg/collage"
)

// NotFound is what an unknown address answers with, from site.json.
func NotFound(store *content.Store) *collage.Fragment {
	return collage.NewFragment(
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
}
