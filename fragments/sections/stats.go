package sections

type statsView struct {
	Items []stat `json:"items"`
}

type stat struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Href  string `json:"href"`
	// External opens the link in a new tab.
	External bool `json:"external"`
}
