package documents

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// SitePage is a page with a fixed path, as llms.txt lists it.
type SitePage struct {
	Name        string
	Path        string
	Description string
}

// Discovery is what language models are pointed at. robots.txt, the sitemap and
// the feed are elagoht/robots', elagoht/sitemap's and elagoht/feed's.
type Discovery struct {
	Store  *content.Store
	Client *blog.Client
	Pages  []SitePage
}

// LLMs is /llms.txt.
func (d *Discovery) LLMs() *collage.Document {
	return collage.NewDocument("llms", "text/plain; charset=utf-8").
		WithPath("en", "/llms.txt").
		WithCacheParams().
		Incremental(time.Hour).
		WithHandler(d.llms).
		Build()
}

// llms is /llms.txt, after llmstxt.org: who the site is about, its pages, and
// every post, as Markdown links.
func (d *Discovery) llms(ctx context.Context, _ *collage.RenderContext) ([]byte, []string, error) {
	site, err := d.Store.Site()
	if err != nil {
		return nil, nil, err
	}
	posts, err := d.Client.AllPosts(ctx)
	if err != nil {
		return nil, nil, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n> %s\n\n", site.Name, oneLine(site.Description))
	if about, err := d.Store.Page("about"); err == nil && about.Person != nil && about.Person.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", oneLine(about.Person.Description))
	}

	b.WriteString("## Pages\n\n")
	for _, page := range d.Pages {
		link(&b, page.Name, site.URL+page.Path, page.Description)
	}

	b.WriteString("\n## Blog posts\n\n")
	for _, post := range posts {
		link(&b, post.Title, postURL(site, post), post.Description)
	}

	b.WriteString("\n## Optional\n\n")
	link(&b, "RSS feed", site.URL+"/rss", "The latest posts.")
	link(&b, "Sitemap", site.URL+"/sitemap.xml", "Every page and post.")
	return []byte(b.String()), []string{blog.TagPosts}, nil
}

func postURL(site content.Site, post blog.Post) string {
	return site.URL + "/blogs/" + url.PathEscape(post.Slug)
}

// link writes one Markdown list item, with the brackets that would end its
// text escaped.
func link(b *strings.Builder, text, href, description string) {
	text = strings.NewReplacer(`[`, `\[`, `]`, `\]`).Replace(oneLine(text))
	fmt.Fprintf(b, "- [%s](%s)", text, href)
	if description = oneLine(description); description != "" {
		fmt.Fprintf(b, ": %s", description)
	}
	b.WriteString("\n")
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
