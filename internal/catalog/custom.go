package catalog

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// A custom model as kept in the custom models file: its Rebrickable listing
// and its parts, in Rebrickable's part numbers and colors.
type customModel struct {
	Num      string `json:"num"`
	Name     string `json:"name"`
	Designer string `json:"designer"`
	URL      string `json:"url"`
	Image    string `json:"image"`
	Year     int32  `json:"year"`
	Lines    []struct {
		Part     string `json:"part"`
		Color    int32  `json:"color"`
		Quantity int32  `json:"quantity"`
	} `json:"lines"`
}

// loadCustom adds the free custom models (MOCs) in path to the sets, marked
// Custom. The file is a JSON list of customModel; a missing file adds none.
func (c *Catalog) loadCustom(path string) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var models []customModel
	if err := json.Unmarshal(b, &models); err != nil {
		return err
	}
	for _, m := range models {
		s := &Set{Num: m.Num, Name: m.Name, Year: m.Year, Theme: "Custom", ThemeGroup: "Custom models", ImageURL: m.Image, Custom: true, Designer: m.Designer, URL: m.URL}
		for _, l := range m.Lines {
			s.Lines = append(s.Lines, Line{Part: l.Part, Color: l.Color, Quantity: l.Quantity, ImageURL: c.Images[PartColor{l.Part, l.Color}]})
		}
		c.Sets[s.Num] = s
	}
	return nil
}
