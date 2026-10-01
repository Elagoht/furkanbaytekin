// Package seo declares a page's title and description for the layout's head.
//
// The innermost declaration of each wins, so a page's replaces the site-wide
// default the layout made. The rest of the head — the canonical URL, Open
// Graph, Twitter cards — is elagoht/meta's, from the page's own address.
package seo

import (
	"furkanbaytekin/data/content"

	meta "github.com/Elagoht/collage-meta"
	"github.com/Elagoht/collage/pkg/collage"
)

// Apply hoists meta's title as the page's <title>, and hands its title and
// description to elagoht/meta.
func Apply(rc *collage.RenderContext, page content.SEO) {
	if page.Title != "" {
		rc.HoistTitle(page.Title)
	}
	meta.Set(rc, meta.Page{Title: page.Title, Description: page.Description})
}
