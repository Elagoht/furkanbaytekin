package actions

import (
	"log/slog"
	"net/http"

	"furkanbaytekin/actions/funcs"
	"furkanbaytekin/data/blog"

	"github.com/Elagoht/collage/pkg/collage"
)

// Webhook is POST /api/webhook, where Bloggo reports every change.
//
// Bloggo sends the headers configured in its panel; this one expects the
// shared secret in X-Webhook-Secret. It has no forgery check: nothing about it
// is a browser's, and the secret is what a forgery token would have been. With
// no secret configured, every request is refused.
func Webhook(secret string, log *slog.Logger, client *blog.Client) *collage.Action {
	if secret == "" {
		log.Warn("webhook: WEBHOOK_SECRET is not set, so /api/webhook refuses everything")
	}
	return collage.NewAction("webhook").
		WithPath("en", "/api/webhook").
		WithMethods(http.MethodPost).
		WithMaxBodyBytes(4 << 20).
		WithoutCSRF().
		WithHandler(funcs.Webhook(secret, log, client)).
		Build()
}
