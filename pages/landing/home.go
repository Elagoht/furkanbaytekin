package pages

import (
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// Home is /, the sections home.json lists.
func Home(store *content.Store) (*collage.Page, error) {
	return sectionPage(store, "home", "/")
}
