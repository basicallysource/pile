package catalog

import (
	"database/sql"
	"encoding/json"

	_ "modernc.org/sqlite"
)

// FromBrickLink is the Rebrickable part and color of a piece a sorter
// classified in BrickLink's ids, if the catalog has both (a piece of unknown
// color has none).
func (c *Catalog) FromBrickLink(part string, color int32, colorKnown bool) (PartColor, bool) {
	p := c.Parts[c.BrickLinkParts[part]]
	col, ok := c.BrickLinkColors[color]
	if p == nil || !colorKnown || !ok {
		return PartColor{}, false
	}
	return PartColor{Part: p.Num, Color: col}, true
}

// loadBrickLink reads which Rebrickable part and color each BrickLink id is,
// from a sorter Hive's parts database (its part_bricklink_ids table, and the
// external ids Rebrickable lists on each color). The sorter classifies in
// BrickLink's ids.
func (c *Catalog) loadBrickLink(path string) error {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	c.BrickLinkParts = map[string]string{}
	rows, err := db.Query(`select part_num, item_no, is_primary from part_bricklink_ids`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var part, item string
		var primary bool
		if err := rows.Scan(&part, &item, &primary); err != nil {
			rows.Close()
			return err
		}
		// One BrickLink id can stand for several Rebrickable parts: take the
		// one with the same number, else the one marked primary.
		cur, ok := c.BrickLinkParts[item]
		if !ok || part == item || (primary && cur != item) {
			c.BrickLinkParts[item] = part
		}
	}
	rows.Close()

	c.BrickLinkColors = map[int32]int32{}
	rows, err = db.Query(`select id, extra from colors`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int32
		var extra sql.NullString
		if err := rows.Scan(&id, &extra); err != nil {
			return err
		}
		var x struct {
			ExternalIDs struct {
				BrickLink struct {
					IDs []*int32 `json:"ext_ids"`
				} `json:"BrickLink"`
			} `json:"external_ids"`
		}
		if json.Unmarshal([]byte(extra.String), &x) != nil {
			continue
		}
		for _, bl := range x.ExternalIDs.BrickLink.IDs {
			if bl == nil {
				continue
			}
			if _, ok := c.BrickLinkColors[*bl]; !ok {
				c.BrickLinkColors[*bl] = id
			}
			if col := c.Colors[id]; col != nil && col.BrickLinkID == 0 {
				col.BrickLinkID = *bl
			}
		}
	}
	return rows.Err()
}
