package content

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

//go:embed *.json
var embedded embed.FS

func Embedded() fs.FS { return embedded }

type Store struct {
	files fs.FS
}

func NewStore(files fs.FS) *Store {
	return &Store{files: files}
}

type Site struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Person      Person   `json:"person"`
	Nav         []Link   `json:"nav"`
	Footer      Footer   `json:"footer"`
	NotFound    NotFound `json:"notFound"`
}

// NotFound is the page an unknown address answers with.
type NotFound struct {
	Title   string   `json:"title"`
	Code    string   `json:"code"`
	Heading string   `json:"heading"`
	Message string   `json:"message"`
	Actions []Action `json:"actions"`
}

type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
	// Variant is "primary" or "ghost".
	Variant string `json:"variant"`
}

type Person struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	JobTitle string   `json:"jobTitle"`
	Image    string   `json:"image"`
	SameAs   []string `json:"sameAs"`
}

type Link struct {
	Label string `json:"label"`
	Href  string `json:"href"`
	Icon  string `json:"icon,omitempty"`
}

type Footer struct {
	Groups    []FooterGroup `json:"groups"`
	Copyright string        `json:"copyright"`
}

type FooterGroup struct {
	Title string `json:"title"`
	Links []Link `json:"links"`
}

type Page struct {
	SEO      SEO            `json:"seo"`
	Person   *PersonDetails `json:"person,omitempty"`
	Sections []Section      `json:"sections"`
}

type PersonDetails struct {
	Description string   `json:"description,omitempty"`
	KnowsAbout  []string `json:"knowsAbout,omitempty"`
}

type SEO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Section struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Blog struct {
	List BlogList `json:"list"`
	Post BlogPost `json:"post"`
	Feed BlogFeed `json:"feed"`
}

type BlogList struct {
	SEO               SEO    `json:"seo"`
	Heading           string `json:"heading"`
	Description       string `json:"description"`
	SearchPlaceholder string `json:"searchPlaceholder"`
	FilterLabel       string `json:"filterLabel"`
	CategoriesLabel   string `json:"categoriesLabel"`
	TagsLabel         string `json:"tagsLabel"`
	ClearLabel        string `json:"clearLabel"`
	EmptyMessage      string `json:"emptyMessage"`
	PageSize          int    `json:"pageSize"`
}

type BlogPost struct {
	TitleSuffix  string `json:"titleSuffix"`
	TOCLabel     string `json:"tocLabel"`
	RelatedLabel string `json:"relatedLabel"`
	AudioLabel   string `json:"audioLabel"`
	RelatedCount int    `json:"relatedCount"`
}

type BlogFeed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Size        int    `json:"size"`
}

type Document interface {
	Site | Page | Blog
}

func (s *Store) Site() (Site, error) { return load[Site](s, "site.json") }
func (s *Store) Blog() (Blog, error) { return load[Blog](s, "blog.json") }

func (s *Store) Page(name string) (Page, error) { return load[Page](s, name+".json") }

func load[T Document](s *Store, name string) (T, error) {
	var doc T
	raw, err := fs.ReadFile(s.files, name)
	if err != nil {
		return doc, fmt.Errorf("content: read %s: %w", name, err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return doc, fmt.Errorf("content: decode %s: %w", name, err)
	}
	return doc, nil
}
