package funcs

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"furkanbaytekin/data/blog"

	sitemap "github.com/Elagoht/collage-sitemap"
	"github.com/Elagoht/collage/pkg/collage"
)

// Webhook receives what Bloggo reports, and invalidates the cached pages the
// change made wrong. With no secret, every request is refused.
func Webhook(secret string, log *slog.Logger, client *blog.Client) collage.ActionHandlerFunc {
	return func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		return receive(rc, secret, log, client)
	}
}

// webhookPayload is what Bloggo sends. Data is the changed entity, which this
// site does not need: it reads the CMS again when a page renders.
type webhookPayload struct {
	Event   string          `json:"event"`
	Entity  string          `json:"entity"`
	Slug    *string         `json:"slug"`
	OldSlug *string         `json:"oldSlug"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

type webhookResponse struct {
	Success     bool     `json:"success"`
	Invalidated []string `json:"invalidated"`
	Message     string   `json:"message,omitempty"`
}

func receive(rc *collage.RenderContext, secret string, log *slog.Logger, client *blog.Client) (*collage.ActionResult, error) {
	given := rc.Request.Header.Get("X-Webhook-Secret")
	if secret == "" || subtle.ConstantTimeCompare([]byte(given), []byte(secret)) != 1 {
		log.Warn("webhook: refused", "remote", rc.Request.RemoteAddr)
		return collage.JSONOf(http.StatusUnauthorized, webhookResponse{Message: "invalid X-Webhook-Secret"})
	}

	var payload webhookPayload
	if err := json.NewDecoder(rc.Request.Body).Decode(&payload); err != nil {
		return collage.JSONOf(http.StatusBadRequest, webhookResponse{Message: "invalid payload"})
	}

	tags := tagsFor(payload)
	if len(tags) > 0 {
		// The list of every post is kept for a minute; what is made again
		// now must be made from the CMS as it is after the change.
		client.ForgetPosts()
	}
	log.Info("webhook", "event", payload.Event, "invalidated", tags)

	result, err := collage.JSONOf(http.StatusOK, webhookResponse{Success: true, Invalidated: tags})
	if err != nil {
		return nil, err
	}
	result.InvalidateTags = tags
	return result, nil
}

// tagsFor is what a change makes wrong. An entity this site does not show is
// acknowledged and invalidates nothing: refusing it would only make Bloggo retry.
func tagsFor(p webhookPayload) []string {
	all := []string{blog.TagPosts, blog.TagCategories, blog.TagTags, blog.TagAuthors, sitemap.Tag}
	switch p.Entity {
	case "post":
		// The sitemap lists every post and when it changed.
		tags := []string{blog.TagPosts, sitemap.Tag}
		for _, slug := range []*string{p.Slug, p.OldSlug} {
			if slug != nil && *slug != "" {
				tags = append(tags, blog.TagPost(*slug))
			}
		}
		return tags
	case "category":
		// Category names are on every card and post page.
		return []string{blog.TagCategories, blog.TagPosts}
	case "tag":
		return []string{blog.TagTags, blog.TagPosts}
	case "author":
		return []string{blog.TagAuthors}
	case "cms":
		return all
	default:
		return []string{}
	}
}
