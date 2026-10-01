package documents

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"time"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// Feed is /rss: the latest posts as RSS 2.0.
func Feed(store *content.Store, client *blog.Client) *collage.Document {
	return collage.NewDocument("feed", "application/rss+xml; charset=utf-8").
		WithPath("en", "/rss").
		WithCacheParams().
		Incremental(30 * time.Minute).
		WithHandler(func(ctx context.Context, _ *collage.RenderContext) ([]byte, []string, error) {
			body, err := feed(ctx, store, client)
			return body, []string{blog.TagPosts}, err
		}).
		Build()
}

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Self        atomLink  `xml:"atom:link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	Description string `xml:"description,omitempty"`
	Category    string `xml:"category,omitempty"`
	PubDate     string `xml:"pubDate"`
}

func feed(ctx context.Context, store *content.Store, client *blog.Client) ([]byte, error) {
	site, err := store.Site()
	if err != nil {
		return nil, err
	}
	labels, err := store.Blog()
	if err != nil {
		return nil, err
	}
	list, err := client.Posts(ctx, blog.PostQuery{Limit: labels.Feed.Size})
	if err != nil {
		return nil, err
	}

	channel := rssChannel{
		Title:       labels.Feed.Title,
		Link:        site.URL + "/blogs",
		Self:        atomLink{Href: site.URL + "/rss", Rel: "self", Type: "application/rss+xml"},
		Description: labels.Feed.Description,
		Language:    "en",
	}
	for _, post := range list.Posts {
		link := site.URL + "/blogs/" + url.PathEscape(post.Slug)
		item := rssItem{
			Title:       post.Title,
			Link:        link,
			GUID:        link,
			Description: post.Description,
			Category:    post.Category.Name,
			PubDate:     post.PublishedAt.Format(time.RFC1123Z),
		}
		channel.Items = append(channel.Items, item)
	}

	body, err := xml.MarshalIndent(rss{Version: "2.0", Atom: "http://www.w3.org/2005/Atom", Channel: channel}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("feed: %w", err)
	}
	return append([]byte(xml.Header), body...), nil
}
