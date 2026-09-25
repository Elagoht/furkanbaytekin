package sections

type hobbiesView struct {
	Label string  `json:"label"`
	Items []hobby `json:"items"`
}

type hobby struct {
	Icon        string `json:"icon"`
	Title       string `json:"title"`
	Description string `json:"description"`
}
