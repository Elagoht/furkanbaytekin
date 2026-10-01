package pages

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"furkanbaytekin/blog"
	"furkanbaytekin/content"
	"furkanbaytekin/fragments/layouts"
	"furkanbaytekin/fragments/seo"

	jsonld "github.com/Elagoht/collage-jsonld"
	"github.com/Elagoht/collage/pkg/collage"
	"golang.org/x/sync/errgroup"
)

type listView struct {
	Labels     content.BlogList
	Action     string
	Search     string
	Category   string
	Tag        string
	Filtered   bool
	Clear      string
	Categories []filterLink
	Tags       []filterLink
	Posts      []card
	Pagination pagination
}

type filterLink struct {
	Name   string
	Count  int
	URL    string
	Active bool
}

// BlogsPage is /blogs, cached for a few minutes per page and filter.
func (b *Blog) BlogsPage() *collage.Page {
	return collage.NewPage("blogs").
		WithLayouts(layouts.Layout(b.Store)).
		WithContent(b.list("blogs-content", "/blogs")).
		WithPath("en", "/blogs").
		WithCacheParams("page", "category", "tag").
		Incremental(5 * time.Minute).
		Build()
}

// SearchPage is /blogs/search. It is never cached: its key would be whatever
// anyone types.
func (b *Blog) SearchPage() *collage.Page {
	return collage.NewPage("blogs-search").
		WithLayouts(layouts.Layout(b.Store)).
		WithContent(b.list("blogs-search-content", "/blogs/search")).
		WithPath("en", "/blogs/search").
		Dynamic().
		Build()
}

func (b *Blog) list(name, path string) *collage.Fragment {
	return collage.NewFragment(
		name,
		"pages/blogs.html",
	).WithDataHandler(collage.DataHandler(
		func(ctx context.Context, rc *collage.RenderContext) (listView, []string, error) {
			return b.listData(ctx, rc, path)
		}),
	).Required().Build()
}

func (b *Blog) listData(ctx context.Context, rc *collage.RenderContext, path string) (listView, []string, error) {
	labels, err := b.Store.Blog()
	if err != nil {
		return listView{}, nil, err
	}
	query := rc.Request.URL.Query()
	view := listView{
		Labels:   labels.List,
		Action:   "/blogs/search",
		Category: query.Get("category"),
		Tag:      query.Get("tag"),
	}
	if path == "/blogs/search" {
		view.Search = strings.TrimSpace(query.Get("search"))
	}
	page := pageParam(query.Get("page"))

	var (
		posts      blog.PostList
		categories []blog.Term
		tags       []blog.Term
	)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		posts, err = b.Client.Posts(ctx, blog.PostQuery{
			Page:     page,
			Limit:    labels.List.PageSize,
			Category: view.Category,
			Tag:      view.Tag,
			Search:   view.Search,
		})
		return err
	})
	group.Go(func() (err error) {
		categories, err = b.Client.Categories(ctx)
		return err
	})
	group.Go(func() (err error) {
		tags, err = b.Client.Tags(ctx)
		return err
	})
	if err := group.Wait(); err != nil {
		return listView{}, nil, err
	}

	link := func(category, tag string, page int) string {
		v := url.Values{}
		if view.Search != "" {
			v.Set("search", view.Search)
		}
		if category != "" {
			v.Set("category", category)
		}
		if tag != "" {
			v.Set("tag", tag)
		}
		if page > 1 {
			v.Set("page", strconv.Itoa(page))
		}
		if len(v) == 0 {
			return path
		}
		return path + "?" + v.Encode()
	}
	toggle := func(current, value string) string {
		if current == value {
			return ""
		}
		return value
	}

	for _, term := range categories {
		if term.PostCount == 0 {
			continue
		}
		view.Categories = append(view.Categories, filterLink{
			Name: term.Name, Count: term.PostCount, Active: term.Slug == view.Category,
			URL: link(toggle(view.Category, term.Slug), view.Tag, 1),
		})
	}
	for _, term := range tags {
		if term.PostCount == 0 {
			continue
		}
		view.Tags = append(view.Tags, filterLink{
			Name: term.Name, Count: term.PostCount, Active: term.Slug == view.Tag,
			URL: link(view.Category, toggle(view.Tag, term.Slug), 1),
		})
	}

	view.Filtered = view.Category != "" || view.Tag != "" || view.Search != ""
	view.Clear = "/blogs"
	view.Posts = b.cards(posts.Posts)
	view.Pagination = paginate(page, posts.Pages(), func(n int) string {
		return link(view.Category, view.Tag, n)
	})

	seo.Apply(rc, labels.List.SEO)
	if err := emitBlogNode(rc, b.Store, labels.List); err != nil {
		return listView{}, nil, err
	}
	return view, []string{blog.TagPosts, blog.TagCategories, blog.TagTags}, nil
}

func emitBlogNode(rc *collage.RenderContext, store *content.Store, labels content.BlogList) error {
	site, err := store.Site()
	if err != nil {
		return err
	}
	jsonld.Emit(rc, jsonld.Blog{
		Name:        labels.SEO.Title,
		Description: labels.Description,
		URL:         labels.SEO.Canonical,
		AuthorName:  site.Person.Name,
		AuthorURL:   site.URL,
	})
	return nil
}
