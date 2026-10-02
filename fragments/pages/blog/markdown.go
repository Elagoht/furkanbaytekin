package fragments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"furkanbaytekin/data/blog"

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

	var out bytes.Buffer
	out.WriteString("---\n")
	field := func(name, value string) {
		if value != "" {
			fmt.Fprintf(&out, "%s: %s\n", name, value)
		}
	}
	field("title", quote(post.Title))
	if description != "" {
		field("description", quote(description))
	}
	field("slug", quote(post.Slug))
	field("url", quote(site.URL+"/blogs/"+url.PathEscape(post.Slug)))
	if post.Author.Name != "" {
		field("author", quote(post.Author.Name))
	}
	if post.Category.Name != "" {
		field("category", quote(post.Category.Name))
	}
	field("tags", quoteList(tags))
	field("published", timestamp(post.PublishedAt.Time))
	field("updated", timestamp(post.UpdatedAt.Time))
	if post.ReadTime > 0 {
		field("readTime", strconv.Itoa(post.ReadTime))
	}
	if post.CoverImage != "" {
		field("cover", quote(b.Client.Asset(post.CoverImage)))
	}
	out.WriteString("---\n\n")
	out.WriteString(body)
	if body != "" && body[len(body)-1] != '\n' {
		out.WriteByte('\n')
	}
	return out.Bytes(), []string{blog.TagPost(post.Slug), blog.TagPosts, blog.TagAuthors}, nil
}

// quote is s as a double-quoted YAML scalar. JSON's string escapes are a subset
// of YAML's, so a title holding a colon, a quote or a newline stays one value.
func quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // a string always encodes
	return string(bytes.TrimSuffix(b.Bytes(), []byte("\n")))
}

// quoteList is a YAML flow sequence of quoted strings: ["go", "web"].
func quoteList(items []string) string {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(quote(item))
	}
	b.WriteByte(']')
	return b.String()
}

// timestamp is t as a YAML timestamp in UTC, or nothing for the zero time.
func timestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
