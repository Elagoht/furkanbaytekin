package documents

import (
	"context"
	"time"

	"furkanbaytekin/data/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// PostModified is elagoht/sitemap's <lastmod>: when a post was last changed, or
// published when it never was. The list is the CMS's every post, which the
// client keeps for a minute, so a sitemap of every post asks for it once. Every
// other page has no date to give.
func PostModified(client *blog.Client) func(context.Context, string, collage.PageURL) time.Time {
	return func(ctx context.Context, page string, u collage.PageURL) time.Time {
		if page != "blog-post" {
			return time.Time{}
		}
		posts, err := client.AllPosts(ctx)
		if err != nil {
			return time.Time{}
		}
		for _, post := range posts {
			if post.Slug == u.Params["slug"] {
				if !post.UpdatedAt.IsZero() {
					return post.UpdatedAt.Time
				}
				return post.PublishedAt.Time
			}
		}
		return time.Time{}
	}
}
