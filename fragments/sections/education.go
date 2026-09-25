package sections

type educationView struct {
	Label string   `json:"label"`
	Items []school `json:"items"`
}

type school struct {
	School    string `json:"school"`
	Programme string `json:"programme"`
	Date      string `json:"date"`
}
