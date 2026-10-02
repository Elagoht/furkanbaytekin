// Package sections is every block a page can be made of: one fragment per
// section type, in the order the page's JSON lists them.
//
// Adding a section type is a view type, an entry in builders, a template under
// templates/fragments/sections and, if it needs one, a stylesheet under
// static/sections. Reordering, repeating or dropping sections is an edit to the
// page's JSON alone, picked up on the next render.
package sections

import (
	"context"
	"encoding/json"
	"fmt"

	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

type builder func(name string, data json.RawMessage) *collage.Fragment

// kind is one section type: the fragment it renders as, and how it is written as
// Markdown, for a page's .md.
type kind struct {
	fragment builder
	markdown func(data json.RawMessage, w *Markdown) error
}

var kinds = map[string]kind{
	"hero":       typed[heroView]("hero"),
	"hobbies":    typed[hobbiesView]("hobbies"),
	"stats":      typed[statsView]("stats"),
	"profile":    typed[profileView]("profile"),
	"chips":      typed[chipsView]("chips"),
	"stack":      typed[stackView]("stack"),
	"expertise":  typed[expertiseView]("expertise"),
	"experience": typed[experienceView]("experience"),
	"education":  typed[educationView]("education"),
	// A line between sections on the page; between Markdown sections the
	// headings are enough.
	"separator": {fragment: separator, markdown: func(json.RawMessage, *Markdown) error { return nil }},
}

// view is a section's data. Each writes itself as Markdown, so a section type
// cannot be added without its .md.
type view interface {
	heroView | hobbiesView | statsView |
		profileView | chipsView | stackView | expertiseView | experienceView | educationView
	markdown(w *Markdown)
}

// Resolver fills a slot with <page>.json's sections, read on every render.
func Resolver(store *content.Store, page string) collage.SlotResolverFunc {
	return func(*collage.RenderContext) ([]*collage.Fragment, error) {
		doc, err := store.Page(page)
		if err != nil {
			return nil, err
		}
		return build(page, doc.Sections)
	}
}

// Check builds <page>.json's sections once, so a section type no builder knows
// fails at startup rather than on the first request.
func Check(store *content.Store, page string) error {
	doc, err := store.Page(page)
	if err != nil {
		return err
	}
	_, err = build(page, doc.Sections)
	return err
}

func build(page string, sections []content.Section) ([]*collage.Fragment, error) {
	fragments := make([]*collage.Fragment, 0, len(sections))
	for i, section := range sections {
		k, ok := kinds[section.Type]
		if !ok {
			return nil, fmt.Errorf("%s: section %d has unknown type %q", page, i, section.Type)
		}
		fragments = append(fragments, k.fragment(fmt.Sprintf("%s-%s-%d", page, section.Type, i), section.Data))
	}
	return fragments, nil
}

// typed is the kind of a section rendering fragments/sections/<template>.html
// with its data decoded as T, and written as Markdown by T.
func typed[T view](template string) kind {
	return kind{fragment: fragmentOf[T](template), markdown: func(data json.RawMessage, w *Markdown) error {
		var view T
		if err := json.Unmarshal(data, &view); err != nil {
			return fmt.Errorf("%s: %w", template, err)
		}
		view.markdown(w)
		return nil
	}}
}

func fragmentOf[T view](kind string) builder {
	return func(name string, data json.RawMessage) *collage.Fragment {
		return collage.NewFragment(
			name,
			"fragments/sections/"+kind+".html",
		).WithDataHandler(collage.DataHandler(
			func(context.Context, *collage.RenderContext) (T, []string, error) {
				var view T
				if err := json.Unmarshal(data, &view); err != nil {
					return view, nil, fmt.Errorf("%s: %w", name, err)
				}
				return view, nil, nil
			}),
		).Build()
	}
}

func separator(name string, _ json.RawMessage) *collage.Fragment {
	return collage.NewFragment(name, "fragments/sections/separator.html").Build()
}
