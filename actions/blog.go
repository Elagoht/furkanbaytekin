package actions

import (
	"net/http"

	"furkanbaytekin/actions/funcs"
	"furkanbaytekin/data/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// View is POST /blogs/{slug}/view, where a post page's script counts a view.
//
// It carries no forgery check: it only increments a public counter, which
// anyone can already do by opening the page, and a token would have to be
// planted in every cached post page.
func View(client *blog.Client) *collage.Action {
	return collage.NewAction("blog-view").
		WithPath("en", "/blogs/{slug}/view").
		WithMethods(http.MethodPost).
		WithMaxBodyBytes(1 << 10).
		WithoutCSRF().
		WithHandler(funcs.CountView(client)).
		Build()
}
