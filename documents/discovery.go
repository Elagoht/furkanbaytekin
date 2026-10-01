package documents

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"time"

	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// SitePage is a page with a fixed path, as robots.txt, the sitemap and
// llms.txt list it.
type SitePage struct {
	Name        string
	Path        string
	Description string
}

// Discovery is what crawlers and language models are pointed at.
type Discovery struct {
	Store  *content.Store
	Client *blog.Client
	Pages  []SitePage
}

func (d *Discovery) Documents() []*collage.Document {
	return []*collage.Document{
		collage.NewDocument("robots", "text/plain; charset=utf-8").
			WithPath("en", "/robots.txt").
			WithCacheParams().
			Incremental(time.Hour).
			WithHandler(d.robots).
			Build(),
		collage.NewDocument("sitemap", "application/xml; charset=utf-8").
			WithPath("en", "/sitemap.xml").
			WithCacheParams().
			Incremental(time.Hour).
			WithHandler(d.sitemap).
			Build(),
		collage.NewDocument("llms", "text/plain; charset=utf-8").
			WithPath("en", "/llms.txt").
			WithCacheParams().
			Incremental(time.Hour).
			WithHandler(d.llms).
			Build(),
	}
}

func (d *Discovery) robots(context.Context, *collage.RenderContext) ([]byte, []string, error) {
	site, err := d.Store.Site()
	if err != nil {
		return nil, nil, err
	}
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	b.WriteString("Allow: /\n")
	// Every search is a fresh render and says nothing the list does not.
	b.WriteString("Disallow: /blogs/search\n")
	b.WriteString("\nSitemap: " + site.URL + "/sitemap.xml\n")
	return []byte(b.String()), nil, nil
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	NS      string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (d *Discovery) sitemap(ctx context.Context, _ *collage.RenderContext) ([]byte, []string, error) {
	site, err := d.Store.Site()
	if err != nil {
		return nil, nil, err
	}
	posts, err := d.Client.AllPosts(ctx)
	if err != nil {
		return nil, nil, err
	}

	set := urlSet{NS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, page := range d.Pages {
		set.URLs = append(set.URLs, sitemapURL{Loc: site.URL + page.Path})
	}
	for _, post := range posts {
		modified := post.UpdatedAt.Time
		if modified.IsZero() {
			modified = post.PublishedAt.Time
		}
		entry := sitemapURL{Loc: postURL(site, post)}
		if !modified.IsZero() {
			entry.LastMod = modified.Format(time.DateOnly)
		}
		set.URLs = append(set.URLs, entry)
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("sitemap: %w", err)
	}
	return append([]byte(xml.Header), body...), []string{blog.TagPosts}, nil
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
