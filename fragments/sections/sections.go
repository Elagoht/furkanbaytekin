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

	"furkanbaytekin/content"

	"github.com/Elagoht/collage/pkg/collage"
)

type builder func(name string, data json.RawMessage) *collage.Fragment

var builders = map[string]builder{
	"hero":       typed[heroView]("hero"),
	"hobbies":    typed[hobbiesView]("hobbies"),
	"stats":      typed[statsView]("stats"),
	"profile":    typed[profileView]("profile"),
	"chips":      typed[chipsView]("chips"),
	"stack":      typed[stackView]("stack"),
	"expertise":  typed[expertiseView]("expertise"),
	"experience": typed[experienceView]("experience"),
	"education":  typed[educationView]("education"),
	"separator":  separator,
}

type view interface {
	heroView | hobbiesView | statsView |
		profileView | chipsView | stackView | expertiseView | experienceView | educationView
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
		newFragment, ok := builders[section.Type]
		if !ok {
			return nil, fmt.Errorf("%s: section %d has unknown type %q", page, i, section.Type)
		}
		fragments = append(fragments, newFragment(fmt.Sprintf("%s-%s-%d", page, section.Type, i), section.Data))
	}
	return fragments, nil
}

// typed is the builder of a section rendering fragments/sections/<kind>.html
// with its data decoded as T.
func typed[T view](kind string) builder {
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
