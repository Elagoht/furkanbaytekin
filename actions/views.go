package actions

import (
	"context"
	"errors"
	"net/http"

	"furkanbaytekin/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// ViewAction counts a reader's view of a post in the CMS and answers with the
// post's new count, so a cached page can show a live one.
//
// It carries no forgery check: it only increments a public counter, which
// anyone can already do by opening the page, and a token would have to be
// planted in every cached post page.
func ViewAction(client *blog.Client) *collage.Action {
	return collage.NewAction("blog-view").
		WithPath("en", "/blogs/{slug}/view").
		WithMethods(http.MethodPost).
		WithMaxBodyBytes(1 << 10).
		WithoutCSRF().
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			return countView(ctx, rc, client)
		}).
		Build()
}

type viewResponse struct {
	Views int `json:"views"`
}

func countView(ctx context.Context, rc *collage.RenderContext, client *blog.Client) (*collage.ActionResult, error) {
	slug := rc.Param("slug")
	err := client.TrackView(ctx, slug, rc.Request.UserAgent())
	if errors.Is(err, blog.ErrNotFound) {
		return collage.JSON(http.StatusNotFound, []byte(`{"views":0}`)), nil
	}
	if err != nil {
		return nil, err
	}
	views, err := client.Views(ctx)
	if err != nil {
		return nil, err
	}
	return collage.JSONOf(http.StatusOK, viewResponse{Views: views[slug]})
}
