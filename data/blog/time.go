package blog

import (
	"encoding/json"
	"fmt"
	"time"
)

// Time reads the CMS's "2006-01-02 15:04:05" timestamps, which are UTC, as well
// as RFC 3339.
type Time struct {
	time.Time
}

var layouts = []string{time.DateTime, time.RFC3339, time.DateOnly}

func (t *Time) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	if text == "" {
		return nil
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("blog: unrecognised time %q", text)
}
