package sections

type stackView struct {
	Label  string       `json:"label"`
	Groups []stackGroup `json:"groups"`
}

type stackGroup struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}
