package documents

import (
	"context"
	"net/url"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"

	feed "github.com/Elagoht/collage-feed"
)

// Feed is /rss, served by elagoht/feed: the latest posts, with the title and
// description blog.json's feed gives them, read once when the site starts. It
// stays at /rss, where readers subscribed to it, and is made again when the
// posts change.
func Feed(store *content.Store, client *blog.Client) (feed.Feed, error) {
	words, err := store.Blog()
	if err != nil {
		return feed.Feed{}, err
	}
	return feed.Feed{
		Title:       words.Feed.Title,
		Description: words.Feed.Description,
		Language:    "en",
		Link:        "/blogs",
		RSS:         "/rss",
		Atom:        "-",
		Limit:       words.Feed.Size,
		Tags:        []string{blog.TagPosts},
		Items: func(ctx context.Context) ([]feed.Item, error) {
			list, err := client.Posts(ctx, blog.PostQuery{Limit: words.Feed.Size})
			if err != nil {
				return nil, err
			}
			items := make([]feed.Item, 0, len(list.Posts))
			for _, post := range list.Posts {
				items = append(items, feed.Item{
					Title:      post.Title,
					Link:       "/blogs/" + url.PathEscape(post.Slug),
					Summary:    post.Description,
					Author:     post.Author.Name,
					Published:  post.PublishedAt.Time,
					Updated:    post.UpdatedAt.Time,
					Categories: []string{post.Category.Name},
				})
			}
			return items, nil
		},
	}, nil
}
