package pages

import (
	"furkanbaytekin/data/content"

	"github.com/Elagoht/collage/pkg/collage"
)

// Home is /, the sections home.json lists. It also owns the addresses the
// previous site served its own files at, sent on for good to where this one does:
// browsers ask for /favicon.ico whatever a page links, and a feed reader
// subscribed to /rss.xml keeps asking for it.
func Home(store *content.Store) (*collage.Page, error) {
	return sectionPage(store, "home", "/", legacyAddresses...)
}

// legacyAddresses are the previous site's paths for what this one serves
// elsewhere: from, then to.
var legacyAddresses = [][2]string{
	{"/favicon.ico", "/static/icons/favicon.ico"},
	{"/favicon-32x32.png", "/static/icons/favicon-32x32.png"},
	{"/apple-touch-icon.png", "/static/icons/apple-touch-icon.png"},
	{"/android-chrome-192x192.png", "/static/icons/android-chrome-192x192.png"},
	{"/android-chrome-512x512.png", "/static/icons/android-chrome-512x512.png"},
	{"/manifest.json", "/static/manifest.webmanifest"},
	{"/rss.xml", "/rss"},
}
