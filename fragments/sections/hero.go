package sections

import "furkanbaytekin/content"

type heroView struct {
	Avatar  image          `json:"avatar"`
	Name    string         `json:"name"`
	Titles  []content.Link `json:"titles"`
	Bio     string         `json:"bio"`
	Actions []action       `json:"actions"`
	Socials []content.Link `json:"socials"`
}

// image is an <img> with its drawn size declared, which is what opti-image
// needs to serve a resized copy in place of the source.
type image struct {
	Src    string `json:"src"`
	Alt    string `json:"alt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
	// Variant is "primary" or "ghost".
	Variant string `json:"variant"`
}
