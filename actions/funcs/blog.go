package funcs

import (
	"context"
	"errors"
	"net/http"

	"furkanbaytekin/data/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// CountView counts a reader's view of a post in the CMS and answers with the
// post's new count, so a cached page can show a live one.
func CountView(client *blog.Client) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		return countView(ctx, rc, client)
	}
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
