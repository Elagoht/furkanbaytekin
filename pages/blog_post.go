package pages

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"time"

	"furkanbaytekin/blog"
	"furkanbaytekin/content"
	"furkanbaytekin/fragments/layouts"
	"furkanbaytekin/fragments/seo"

	jsonld "github.com/Elagoht/collage-jsonld"
	"github.com/Elagoht/collage/pkg/collage"
)

type postView struct {
	Labels   content.BlogPost
	Slug     string
	Title    string
	Excerpt  string
	Category string
	Date     string
	DateISO  string
	ReadTime int
	Views    int
	Cover    image
	Audio    string
	Content  template.HTML
	Headings []blog.Heading
	Related  []card
}

// PostPage is /blogs/{slug}, cached for a few minutes. Its view count is
// refreshed by the page's script.
func (b *Blog) PostPage() *collage.Page {
	content := collage.NewFragment(
		"blog-post-content",
		"pages/blog-post.html",
	).WithDataHandler(collage.DataHandler(b.postData)).Required().Build()

	return collage.NewPage("blog-post").
		WithLayout(layouts.Layout(b.Store)).
		WithContent(content).
		WithPath("en", "/blogs/{slug}").
		WithCacheParams().
		Incremental(10 * time.Minute).
		WithStaticParams(b.postParams).
		Build()
}

// postParams lists the posts a static build writes a page for: every post the
// CMS has.
func (b *Blog) postParams(ctx context.Context, _ string) ([]map[string]string, error) {
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

func (b *Blog) postData(ctx context.Context, rc *collage.RenderContext) (postView, []string, error) {
	slug := rc.Param("slug")
	post, err := b.Client.Post(ctx, slug)
	if errors.Is(err, blog.ErrNotFound) {
		return postView{}, nil, fmt.Errorf("blog post %q: %w", slug, collage.ErrNotFound)
	}
	if err != nil {
		return postView{}, nil, err
	}
	labels, err := b.Store.Blog()
	if err != nil {
		return postView{}, nil, err
	}
	site, err := b.Store.Site()
	if err != nil {
		return postView{}, nil, err
	}
	rendered, err := b.Renderer.Render(post.Content)
	if err != nil {
		return postView{}, nil, fmt.Errorf("blog post %q: markdown: %w", slug, err)
	}

	related, err := b.related(rc, post, labels.Post.RelatedCount)
	if err != nil {
		return postView{}, nil, err
	}

	canonical := site.URL + "/blogs/" + url.PathEscape(post.Slug)
	excerpt := post.Description
	if excerpt == "" {
		excerpt = post.Spot
	}
	seo.Apply(rc, content.SEO{
		Title:       post.Title + labels.Post.TitleSuffix,
		Description: excerpt,
		Canonical:   canonical,
	})
	jsonld.Emit(rc, jsonld.BlogPosting{
		Headline:      post.Title,
		Description:   excerpt,
		URL:           canonical,
		Section:       post.Category.Name,
		Keywords:      termNames(post.Tags),
		DatePublished: post.PublishedAt.Time,
		DateModified:  post.UpdatedAt.Time,
		AuthorName:    post.Author.Name,
		AuthorURL:     site.URL,
		ImageURL:      b.Client.Asset(post.CoverImage),
	})

	return postView{
		Labels:   labels.Post,
		Slug:     post.Slug,
		Title:    post.Title,
		Excerpt:  excerpt,
		Category: post.Category.Name,
		Date:     post.PublishedAt.Format(dateLayout),
		DateISO:  post.PublishedAt.Format("2006-01-02"),
		ReadTime: post.ReadTime,
		Views:    post.ReadCount,
		Cover:    b.cover(post, blog.ImageWidth, blog.ImageHeight),
		Audio:    b.Client.Asset(post.AudioFile),
		Content:  rendered.HTML,
		Headings: rendered.Headings,
		Related:  related,
	}, []string{blog.TagPost(post.Slug), blog.TagPosts, blog.TagAuthors}, nil
}

// related is the latest posts in post's category, without post itself.
//
// The list is fetched once per category and shared between the posts in it, for
// as long as a post page is cached, and dropped with the posts when the webhook
// invalidates them. An export renders every post, and without this it asked the
// CMS for the same few lists once per post — past the CMS's 100 requests a
// minute.
func (b *Blog) related(rc *collage.RenderContext, post blog.Post, count int) ([]card, error) {
	if count <= 0 {
		return nil, nil
	}
	category := post.Category.Slug
	list, err := collage.Cached(rc, fmt.Sprintf("related:%s:%d", category, count+1), 10*time.Minute, []string{blog.TagPosts},
		func(ctx context.Context) (blog.PostList, error) {
			return b.Client.Posts(ctx, blog.PostQuery{Limit: count + 1, Category: category})
		})
	if err != nil {
		return nil, err
	}
	var others []blog.Post
	for _, candidate := range list.Posts {
		if candidate.Slug != post.Slug && len(others) < count {
			others = append(others, candidate)
		}
	}
	return b.cards(others), nil
}

func termNames(terms []blog.Term) []string {
	names := make([]string, 0, len(terms))
	for _, term := range terms {
		names = append(names, term.Name)
	}
	return names
}
