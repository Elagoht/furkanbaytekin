package fragments

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/frontmatter"

	"github.com/Elagoht/collage/pkg/collage"
)

// Markdown is a post as Markdown, for /blogs/{slug}.md: its properties as YAML
// front matter, then the body the CMS holds, with its images and uploads made
// absolute so it reads the same anywhere.
func (b *Blog) Markdown(ctx context.Context, rc *collage.RenderContext) ([]byte, []string, error) {
	slug := rc.Param("slug")
	post, err := b.Client.Post(ctx, slug)
	if errors.Is(err, blog.ErrNotFound) {
		return nil, nil, fmt.Errorf("blog post %q: %w", slug, collage.ErrNotFound)
	}
	if err != nil {
		return nil, nil, err
	}
	site, err := b.Store.Site()
	if err != nil {
		return nil, nil, err
	}
	body, err := b.Renderer.Source(post.Content)
	if err != nil {
		return nil, nil, fmt.Errorf("blog post %q: markdown: %w", slug, err)
	}

	description := post.Description
	if description == "" {
		description = post.Spot
	}
	tags := make([]string, 0, len(post.Tags))
	for _, tag := range post.Tags {
		tags = append(tags, tag.Name)
	}

	var fm frontmatter.Writer
	fm.String("title", post.Title)
	fm.String("description", description)
	fm.String("slug", post.Slug)
	fm.String("url", site.URL+"/blogs/"+url.PathEscape(post.Slug))
	fm.String("author", post.Author.Name)
	fm.String("category", post.Category.Name)
	fm.List("tags", tags)
	fm.Time("published", post.PublishedAt.Time)
	fm.Time("updated", post.UpdatedAt.Time)
	fm.Int("readTime", post.ReadTime)
	if post.CoverImage != "" {
		fm.String("cover", b.Client.Asset(post.CoverImage))
	}
	out := bytes.NewBuffer(fm.Bytes())
	out.WriteString(body)
	if body != "" && body[len(body)-1] != '\n' {
		out.WriteByte('\n')
	}
	return out.Bytes(), []string{blog.TagPost(post.Slug), blog.TagPosts, blog.TagAuthors}, nil
}
