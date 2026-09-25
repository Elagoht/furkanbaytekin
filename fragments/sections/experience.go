package sections

type experienceView struct {
	Label string `json:"label"`
	Items []job  `json:"items"`
}

type job struct {
	Title   string `json:"title"`
	Company string `json:"company"`
	Date    string `json:"date"`
	// Points may mark phrases **like this**; see the "emphasis" template func.
	Points []string `json:"points"`
}
