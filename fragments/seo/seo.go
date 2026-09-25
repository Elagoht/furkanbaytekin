// Package seo declares a page's title and metadata for the layout's head.
//
// The innermost declaration of each wins, so a page's replaces the site-wide
// default the layout made.
package seo

import (
	"furkanbaytekin/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// Apply hoists every non-empty field of meta.
func Apply(rc *collage.RenderContext, meta content.SEO) {
	if meta.Title != "" {
		rc.HoistTitle(meta.Title)
	}
	if meta.Description != "" {
		rc.HoistMeta("description", meta.Description)
	}
	if meta.Canonical != "" {
		rc.HoistLink("canonical", meta.Canonical)
	}
}
