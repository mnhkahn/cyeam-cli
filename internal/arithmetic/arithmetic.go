// Package arithmetic defines the server-provided oral arithmetic API models.
package arithmetic

type Type struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Grade int    `json:"grade"`
}

type Options struct {
	Grade, Count int
	Types        []string
	Seed         int64
	Answers      bool
}

type Question struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	Answer string `json:"answer,omitempty"`
}

type Exercise struct {
	Grade     int        `json:"grade"`
	Seed      int64      `json:"seed"`
	Questions []Question `json:"questions"`
	Types     []Type     `json:"-"`
}
