package fragments

import (
	"context"

	"furkanbaytekin/data/content"
	"furkanbaytekin/fragments/sections"
	"furkanbaytekin/fragments/seo"

	jsonld "github.com/Elagoht/collage-jsonld"
	"github.com/Elagoht/collage/pkg/collage"
)

// Sections is the content of the page <name>.json describes: its sections, in
// the order the file lists them, read on every render. A section type no builder
// knows fails here, at startup, rather than on the first request.
func Sections(store *content.Store, name string) (*collage.Fragment, error) {
	if err := sections.Check(store, name); err != nil {
		return nil, err
	}
	return collage.NewFragment(
		name+"-content",
		"pages/sections.html",
	).WithDataHandler(collage.Effect(
		func(_ context.Context, rc *collage.RenderContext) error {
			return head(rc, store, name)
		}),
	).WithSlot("sections", false, true).
		WithSlotResolver("sections", sections.Resolver(store, name)).
		Build(), nil
}

func head(rc *collage.RenderContext, store *content.Store, name string) error {
	page, err := store.Page(name)
	if err != nil {
		return err
	}
	seo.Apply(rc, page.SEO)
	hoistMarkdown(rc, name)
	if page.Person == nil {
		return nil
	}

	site, err := store.Site()
	if err != nil {
		return err
	}
	description := page.Person.Description
	if description == "" {
		description = site.Description
	}
	jsonld.Emit(rc, jsonld.Person{
		Name:        site.Person.Name,
		Description: description,
		URL:         site.URL,
		ImageURL:    site.Person.Image,
		JobTitle:    site.Person.JobTitle,
		SameAs:      site.Person.SameAs,
		Email:       site.Person.Email,
		KnowsAbout:  page.Person.KnowsAbout,
	})
	return nil
}
