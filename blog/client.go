// Package blog reads the site's posts from the Bloggo CMS.
//
// The trusted-frontend key only ever travels from this server to the CMS; a
// browser never sees it.
package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrNotFound is what the CMS answers for a slug it does not have.
var ErrNotFound = errors.New("blog: not found")

type Client struct {
	base   *url.URL
	key    string
	client *http.Client
}

// New returns a client for the API at baseURL, e.g.
// "https://myblogcms.furkanbaytekin.dev/api". The CMS refuses every request
// without key, so an empty one is refused here, where it can be named, rather
// than as a 401 on every page that lists posts.
func New(baseURL, key string) (*Client, error) {
	if key == "" {
		return nil, errors.New("blog: BLOG_TRUSTED_FRONTEND_KEY is not set; the CMS answers every request without it with 401")
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("blog: API URL %q is not absolute", baseURL)
	}
	return &Client{
		base: base,
		key:  key,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// A redirect is answered, not followed: following one would send
			// the key wherever the CMS pointed.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// Every image the CMS stores, covers and images inside posts alike, is this
// size.
const (
	ImageWidth  = 1280
	ImageHeight = 720
)

// Origin is the scheme and host the CMS serves its files from.
func (c *Client) Origin() string {
	return c.base.Scheme + "://" + c.base.Host
}

// IsAsset reports whether raw is a file the CMS serves.
func (c *Client) IsAsset(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == c.base.Scheme && u.Host == c.base.Host
}

// Asset resolves a CMS upload path, such as a cover image, to an absolute URL.
func (c *Client) Asset(path string) string {
	if path == "" || strings.Contains(path, "://") {
		return path
	}
	return (&url.URL{Scheme: c.base.Scheme, Host: c.base.Host, Path: path}).String()
}

type Author struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type Term struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	PostCount int    `json:"postCount"`
}

type Post struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Spot        string `json:"spot"`
	CoverImage  string `json:"coverImage"`
	AudioFile   string `json:"audioFile"`
	ReadCount   int    `json:"readCount"`
	ReadTime    int    `json:"readTime"`
	PublishedAt Time   `json:"publishedAt"`
	UpdatedAt   Time   `json:"updatedAt"`
	Author      Author `json:"author"`
	Category    Term   `json:"category"`
	Tags        []Term `json:"tags"`
	// Content is Markdown. Only a single post carries it.
	Content string `json:"content"`
}

type PostList struct {
	Posts []Post `json:"data"`
	Page  int    `json:"page"`
	Take  int    `json:"take"`
	Total int    `json:"total"`
}

// Pages is the number of pages the list's total spans.
func (l PostList) Pages() int {
	if l.Take <= 0 {
		return 1
	}
	return max(1, (l.Total+l.Take-1)/l.Take)
}

type PostQuery struct {
	Page     int
	Limit    int
	Category string
	Tag      string
	Search   string
}

func (q PostQuery) values() url.Values {
	v := url.Values{}
	if q.Page > 0 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	for key, value := range map[string]string{"category": q.Category, "tag": q.Tag, "search": q.Search} {
		if value != "" {
			v.Set(key, value)
		}
	}
	return v
}

type categoryList struct {
	Categories []Term `json:"categories"`
}

type tagList struct {
	Tags []Term `json:"tags"`
}

type viewCounts map[string]int

// response is every body this client decodes.
type response interface {
	PostList | Post | categoryList | tagList | viewCounts
}

func (c *Client) Posts(ctx context.Context, q PostQuery) (PostList, error) {
	return get[PostList](ctx, c, "/posts", q.values())
}

// slugPattern is what a post's slug may be. Anything else — "..", above all —
// never reaches the CMS, which might resolve it to an endpoint other than a
// post's and answer it with the key's authority.
var slugPattern = regexp.MustCompile(`^[A-Za-z0-9]+(?:[-_+][A-Za-z0-9]+)*$`)

// AllPosts is every published post, newest first, read a page at a time.
func (c *Client) AllPosts(ctx context.Context) ([]Post, error) {
	const limit, maxPages = 100, 100
	var posts []Post
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		list, err := c.Posts(ctx, PostQuery{Page: page, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, post := range list.Posts {
			if !seen[post.Slug] {
				seen[post.Slug] = true
				posts = append(posts, post)
			}
		}
		if len(list.Posts) == 0 || page >= list.Pages() {
			break
		}
	}
	return posts, nil
}

func (c *Client) Post(ctx context.Context, slug string) (Post, error) {
	if !slugPattern.MatchString(slug) {
		return Post{}, fmt.Errorf("blog: slug %q: %w", slug, ErrNotFound)
	}
	return get[Post](ctx, c, "/posts/"+url.PathEscape(slug), nil)
}

func (c *Client) Categories(ctx context.Context) ([]Term, error) {
	body, err := get[categoryList](ctx, c, "/categories", nil)
	return body.Categories, err
}

func (c *Client) Tags(ctx context.Context) ([]Term, error) {
	body, err := get[tagList](ctx, c, "/tags", nil)
	return body.Tags, err
}

// Views is every published post's view count, by slug.
func (c *Client) Views(ctx context.Context) (map[string]int, error) {
	return get[viewCounts](ctx, c, "/posts/views", nil)
}

// TrackView counts one view of slug by a reader with userAgent.
func (c *Client) TrackView(ctx context.Context, slug, userAgent string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("blog: slug %q: %w", slug, ErrNotFound)
	}
	body, err := json.Marshal(struct {
		UserAgent string `json:"userAgent"`
	}{UserAgent: truncate(userAgent, 500)})
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, "/posts/"+url.PathEscape(slug)+"/view", nil, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return nil
}

func get[T response](ctx context.Context, c *Client, path string, query url.Values) (T, error) {
	var body T
	resp, err := c.do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<20)).Decode(&body); err != nil {
		return body, fmt.Errorf("blog: GET %s: decode: %w", path, err)
	}
	return body, nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	target := *c.base
	target.Path += path
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-trusted-frontend", c.key)
	req.Header.Set("User-Agent", "furkanbaytekin.dev")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	// The CMS allows 100 requests a minute from one address, and one more every
	// 600ms after that: a static export, which renders every post, runs past
	// it. It sends no Retry-After, so a GET that was refused is asked again a
	// moment later, for as long as its context allows. A POST is not: its body
	// has been read, and a view counted twice is worse than one not counted.
	for attempt := 1; err == nil && resp.StatusCode == http.StatusTooManyRequests && body == nil && attempt <= maxRetries; attempt++ {
		resp.Body.Close()
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("blog: %s %s: CMS answered %s, and there was no time to ask again: %w", method, path, resp.Status, ctx.Err())
		case <-time.After(retryWait):
		}
		resp, err = c.client.Do(req)
	}
	if err != nil {
		return nil, fmt.Errorf("blog: %s %s: %w", method, path, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fmt.Errorf("blog: %s %s: %w", method, path, ErrNotFound)
	}
	if resp.StatusCode >= 300 {
		// Including a redirect, which is never followed.
		resp.Body.Close()
		return nil, fmt.Errorf("blog: %s %s: CMS answered %s", method, path, resp.Status)
	}
	return resp, nil
}

// How often, and how far apart, a GET the CMS refused for its rate limit is asked
// again. Its limiter frees a request every 600ms, so one wait is usually enough.
const (
	maxRetries = 5
	retryWait  = 700 * time.Millisecond
)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}
