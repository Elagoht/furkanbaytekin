package pages

import (
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// About is /about, the sections about.json lists.
func About(store *content.Store) (*collage.Page, error) {
	return sectionPage(store, "about", "/about")
}
