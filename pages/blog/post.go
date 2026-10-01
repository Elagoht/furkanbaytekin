package pages

import (
	"context"
	"time"

	"furkanbaytekin/fragments/layouts"
	fragments "furkanbaytekin/fragments/pages/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// Post is /blogs/{slug}, cached for a few minutes. Its view count is refreshed
// by the page's script.
func Post(b *fragments.Blog) *collage.Page {
	return collage.NewPage("blog-post").
		WithLayouts(layouts.Master(b.Store)).
		WithContent(b.Post()).
		WithPath("en", "/blogs/{slug}").
		WithCacheParams().
		Incremental(10 * time.Minute).
		WithStaticParams(func(ctx context.Context, _ string) ([]map[string]string, error) {
			return postParams(ctx, b)
		}).
		Build()
}

// postParams lists the posts a static build writes a page for: every post the
// CMS has.
func postParams(ctx context.Context, b *fragments.Blog) ([]map[string]string, error) {
	posts, err := b.Client.AllPosts(ctx)
	if err != nil {
		return nil, err
	}
	params := make([]map[string]string, 0, len(posts))
	for _, post := range posts {
		params = append(params, map[string]string{"slug": post.Slug})
	}
	return params, nil
}
