package main

import (
	"fmt"
	"log/slog"
	"strings"

	"furkanbaytekin/actions"
	"furkanbaytekin/data/blog"
	"furkanbaytekin/data/content"
	"furkanbaytekin/documents"
	blogfragments "furkanbaytekin/fragments/pages/blog"
	blogpages "furkanbaytekin/pages/blog"
	errorpages "furkanbaytekin/pages/errors"
	landingpages "furkanbaytekin/pages/landing"

	"github.com/Elagoht/collage/pkg/collage"
)

// routeSources is what the routes read from: the site's words, the CMS, and the
// webhook's secret.
type routeSources struct {
	store         *content.Store
	client        *blog.Client
	webhookSecret string
	log           *slog.Logger
}

// register adds every page, document and action to app. A new route goes here.
func register(app *collage.App, src routeSources) error {
	home, err := landingpages.Home(src.store)
	if err != nil {
		return fmt.Errorf("page %q: %w", "home", err)
	}
	about, err := landingpages.About(src.store)
	if err != nil {
		return fmt.Errorf("page %q: %w", "about", err)
	}
	posts := &blogfragments.Blog{Store: src.store, Client: src.client, Renderer: blog.NewRenderer(src.client)}
	discovery, err := discoveryDocuments(src.store, src.client)
	if err != nil {
		return err
	}

	items := []collage.Registrable{
		home,
		about,
		blogpages.List(posts),
		blogpages.Search(posts),
		blogpages.Post(posts),
		actions.View(src.client),
		actions.Webhook(src.webhookSecret, src.log),
	}
	for _, document := range discovery {
		items = append(items, document)
	}
	items = append(items, documents.Health(), documents.Feed(src.store, src.client))
	if err := app.Register(items...); err != nil {
		return err
	}

	// Registered rather than given a path: it is reached by failing to match.
	// "collage export" writes it as 404.html.
	if err := app.RegisterNotFoundPage(errorpages.NotFound(src.store)); err != nil {
		return fmt.Errorf("register not-found page: %w", err)
	}
	return nil
}

// discoveryDocuments is robots.txt, the sitemap and llms.txt, listing the pages
// with a fixed path and what each says it is about.
func discoveryDocuments(store *content.Store, client *blog.Client) ([]*collage.Document, error) {
	listed := []documents.SitePage{{Name: "Home", Path: "/"}, {Name: "About", Path: "/about"}}
	for i, page := range listed {
		doc, err := store.Page(strings.ToLower(page.Name))
		if err != nil {
			return nil, err
		}
		listed[i].Description = doc.SEO.Description
	}
	words, err := store.Blog()
	if err != nil {
		return nil, err
	}
	listed = append(listed, documents.SitePage{Name: "Blog", Path: "/blogs", Description: words.List.Description})
	discovery := &documents.Discovery{Store: store, Client: client, Pages: listed}
	return discovery.Documents(), nil
}
