package pages

import (
	"time"

	"furkanbaytekin/fragments/layouts"
	fragments "furkanbaytekin/fragments/pages/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// List is /blogs, cached for a few minutes per page and filter.
func List(b *fragments.Blog) *collage.Page {
	return collage.NewPage("blogs").
		WithLayouts(layouts.Master(b.Store)).
		WithContent(b.List("blogs-content", "/blogs")).
		WithPath("en", "/blogs").
		WithCacheParams("page", "category", "tag").
		Incremental(5 * time.Minute).
		Build()
}

// Search is /blogs/search. It is never cached: its key would be whatever
// anyone types.
func Search(b *fragments.Blog) *collage.Page {
	return collage.NewPage("blogs-search").
		WithLayouts(layouts.Master(b.Store)).
		WithContent(b.List("blogs-search-content", "/blogs/search")).
		WithPath("en", "/blogs/search").
		Dynamic().
		Build()
}
