package pages

import (
	"net/url"
	"strconv"

	"furkanbaytekin/blog"
	"furkanbaytekin/content"
)

// Blog is what the blog pages need: the site's words, and the CMS the posts
// come from.
type Blog struct {
	Store    *content.Store
	Client   *blog.Client
	Renderer *blog.Renderer
}

const dateLayout = "January 2, 2006"

// image is an <img> with its drawn size declared, which is what opti-image
// needs to serve a resized copy of a CMS image from this site.
type image struct {
	Src    string
	Alt    string
	Width  int
	Height int
}

type card struct {
	URL      string
	Title    string
	Excerpt  string
	Category string
	Date     string
	DateISO  string
	Cover    image
}

func (b *Blog) card(post blog.Post) card {
	return card{
		URL:      "/blogs/" + url.PathEscape(post.Slug),
		Title:    post.Title,
		Excerpt:  post.Description,
		Category: post.Category.Name,
		Date:     post.PublishedAt.Format(dateLayout),
		DateISO:  post.PublishedAt.Format("2006-01-02"),
		Cover:    b.cover(post, blog.ImageWidth/2, blog.ImageHeight/2),
	}
}

func (b *Blog) cover(post blog.Post, width, height int) image {
	if post.CoverImage == "" {
		return image{}
	}
	return image{Src: b.Client.Asset(post.CoverImage), Alt: post.Title, Width: width, Height: height}
}

func (b *Blog) cards(posts []blog.Post) []card {
	cards := make([]card, 0, len(posts))
	for _, post := range posts {
		cards = append(cards, b.card(post))
	}
	return cards
}

type pageLink struct {
	Number   int
	URL      string
	Current  bool
	Ellipsis bool
}

type pagination struct {
	Prev  string
	Next  string
	Pages []pageLink
}

// paginate links the pages of a list: the first, the last, and the ones around
// current, with an ellipsis for each gap.
func paginate(current, last int, link func(page int) string) pagination {
	p := pagination{}
	if last <= 1 {
		return p
	}
	if current > 1 {
		p.Prev = link(current - 1)
	}
	if current < last {
		p.Next = link(current + 1)
	}

	lo, hi := max(1, current-1), min(last, current+1)
	if current <= 2 {
		hi = min(last, 3)
	}
	if current >= last-1 {
		lo = max(1, last-2)
	}
	shown := []int{1}
	for n := lo; n <= hi; n++ {
		shown = append(shown, n)
	}
	shown = append(shown, last)

	previous := 0
	for _, n := range shown {
		if n <= previous {
			continue
		}
		if n > previous+1 {
			p.Pages = append(p.Pages, pageLink{Ellipsis: true})
		}
		p.Pages = append(p.Pages, pageLink{Number: n, URL: link(n), Current: n == current})
		previous = n
	}
	return p
}

func pageParam(raw string) int {
	page, err := strconv.Atoi(raw)
	if err != nil || page < 1 {
		return 1
	}
	return page
}
