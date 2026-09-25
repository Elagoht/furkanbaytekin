package actions

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"furkanbaytekin/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// WebhookAction is POST /api/webhook, where Bloggo reports every change, and
// invalidates the cached pages the change made wrong.
//
// Bloggo sends the headers configured in its panel; this one expects the
// shared secret in X-Webhook-Secret. It has no forgery check: nothing about it
// is a browser's, and the secret is what a forgery token would have been. With
// no secret configured, every request is refused.
func WebhookAction(secret string, log *slog.Logger) *collage.Action {
	if secret == "" {
		log.Warn("webhook: WEBHOOK_SECRET is not set, so /api/webhook refuses everything")
	}
	return collage.NewAction("webhook").
		WithPath("en", "/api/webhook").
		WithMethods(http.MethodPost).
		WithMaxBodyBytes(4 << 20).
		WithoutCSRF().
		WithHandler(func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			return receive(rc, secret, log)
		}).
		Build()
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

func receive(rc *collage.RenderContext, secret string, log *slog.Logger) (*collage.ActionResult, error) {
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
	all := []string{blog.TagPosts, blog.TagCategories, blog.TagTags, blog.TagAuthors}
	switch p.Entity {
	case "post":
		tags := []string{blog.TagPosts}
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
