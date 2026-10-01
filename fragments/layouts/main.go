package layouts

import (
	"context"
	"strconv"
	"strings"
	"time"

	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/seo"

	"github.com/Elagoht/collage/pkg/collage"
)

// Master is the page shell: the head, the header and the footer, all from
// site.json.
func Master(store *content.Store) *collage.Fragment {
	return collage.NewFragment("layout", "layouts/default.html").
		WithSlot("content", true, false).
		WithDataHandler(collage.DataHandler(func(_ context.Context, rc *collage.RenderContext) (content.Site, []string, error) {
			site, err := store.Site()
			if err != nil {
				return site, nil, err
			}
			// Defaults for every page; a page hoisting its own replaces them.
			seo.Apply(rc, content.SEO{Title: site.Title, Description: site.Description})
			site.Footer.Copyright = strings.ReplaceAll(site.Footer.Copyright, "{year}", strconv.Itoa(time.Now().Year()))
			return site, nil, nil
		})).
		Build()
}
