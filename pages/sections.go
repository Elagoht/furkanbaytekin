package pages

import (
	"context"

	"furkanbaytekin/content"
	"furkanbaytekin/fragments/layouts"
	"furkanbaytekin/fragments/sections"
	"furkanbaytekin/fragments/seo"

	jsonld "github.com/Elagoht/collage-jsonld"
	"github.com/Elagoht/collage/pkg/collage"
)

// SectionPage is the page <name>.json describes, served at path.
func SectionPage(store *content.Store, name, path string) (*collage.Page, error) {
	if err := sections.Check(store, name); err != nil {
		return nil, err
	}

	fragment := collage.NewFragment(
		name+"-content",
		"pages/sections.html",
	).WithDataHandler(collage.Effect(
		func(_ context.Context, rc *collage.RenderContext) error {
			return head(rc, store, name)
		}),
	).WithSlot("sections", false, true).
		WithSlotResolver("sections", sections.Resolver(store, name)).
		Build()

	return collage.NewPage(name).
		WithLayouts(layouts.Layout(store)).
		WithContent(fragment).
		WithPath("en", path).
		Static().
		Build(), nil
}

func head(rc *collage.RenderContext, store *content.Store, name string) error {
	page, err := store.Page(name)
	if err != nil {
		return err
	}
	seo.Apply(rc, page.SEO)
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
