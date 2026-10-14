package catalog

import (
	"database/sql"
	"encoding/json"
	"strings"

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

// LDrawPart is the Rebrickable part an LDraw part file (named as
// ldraw.Name names it) stands for, if the catalog knows one.
func (c *Catalog) LDrawPart(name string) (string, bool) {
	if p, ok := c.LDrawParts[name]; ok {
		return p, true
	}
	// Rebrickable numbers most parts as LDraw does.
	if _, ok := c.Parts[name]; ok {
		return name, true
	}
	return "", false
}

// LikePart is the catalog part a part number from elsewhere (an old
// BrickLink number, an LDraw file) most likely is: the same number, the part
// LDraw's file of that name stands for, or the part numbered the same but
// for a letter (a mold variant).
func (c *Catalog) LikePart(num string) (string, bool) {
	if _, ok := c.Parts[num]; ok {
		return num, true
	}
	if p, ok := c.LDrawParts[strings.ToLower(num)]; ok {
		return p, true
	}
	if ps := c.stems[moldLetters.ReplaceAllString(num, "")]; len(ps) > 0 {
		return ps[0], true
	}
	return "", false
}

// LDrawColor is the Rebrickable color of an LDraw color code.
func (c *Catalog) LDrawColor(code int32) (int32, bool) {
	col, ok := c.LDrawColors[code]
	return col, ok
}

// loadIDs reads which Rebrickable part and color each BrickLink and LDraw id
// is, from a sorter Hive's parts database: its part_bricklink_ids table, the
// external ids Rebrickable lists on each part and color, and the LDraw file
// part_geometry measured each part from. The sorter classifies in
// BrickLink's ids; 3D models of sets are in LDraw's.
func (c *Catalog) loadIDs(path string) error {
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()

	c.BrickLinkParts = map[string]string{}
	err = rows(db, `select part_num, item_no, is_primary from part_bricklink_ids`, func(scan func(...any) error) error {
		var part, item string
		var primary bool
		if err := scan(&part, &item, &primary); err != nil {
			return err
		}
		// One BrickLink id can stand for several Rebrickable parts: take the
		// one with the same number, else the one marked primary.
		cur, ok := c.BrickLinkParts[item]
		if !ok || part == item || (primary && cur != item) {
			c.BrickLinkParts[item] = part
		}
		return nil
	})
	if err != nil {
		return err
	}

	// One LDraw file can stand for several Rebrickable parts (a part and its
	// mold variants): take the one with the same number, else the first.
	c.LDrawParts = map[string]string{}
	ldraw := func(file, part string) {
		file = strings.ToLower(file)
		if cur, ok := c.LDrawParts[file]; !ok || (part == file && cur != file) {
			c.LDrawParts[file] = part
		}
	}
	err = rows(db, `select part_num, external_ids from parts where external_ids like '%"LDraw"%'`, func(scan func(...any) error) error {
		var part string
		var ext sql.NullString
		if err := scan(&part, &ext); err != nil {
			return err
		}
		var x struct {
			LDraw []string `json:"LDraw"`
		}
		if json.Unmarshal([]byte(ext.String), &x) == nil {
			for _, f := range x.LDraw {
				ldraw(f, part)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = rows(db, `select part_num, ldraw_id from part_geometry where ldraw_id is not null and ldraw_id != ''`, func(scan func(...any) error) error {
		var part, file string
		if err := scan(&part, &file); err != nil {
			return err
		}
		if _, ok := c.LDrawParts[strings.ToLower(file)]; !ok {
			ldraw(file, part)
		}
		return nil
	})
	if err != nil {
		return err
	}

	c.BrickLinkColors = map[int32]int32{}
	c.LDrawColors = map[int32]int32{}
	return rows(db, `select id, extra from colors`, func(scan func(...any) error) error {
		var id int32
		var extra sql.NullString
		if err := scan(&id, &extra); err != nil {
			return err
		}
		var x struct {
			ExternalIDs map[string]struct {
				IDs []*int32 `json:"ext_ids"`
			} `json:"external_ids"`
		}
		if json.Unmarshal([]byte(extra.String), &x) != nil {
			return nil
		}
		for _, bl := range x.ExternalIDs["BrickLink"].IDs {
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
		// Rebrickable's [Unknown] color lists LDraw's codes that are not a
		// color (16 and 24, rubber and the like): none of them is one.
		if id < 0 {
			return nil
		}
		for _, ld := range x.ExternalIDs["LDraw"].IDs {
			if ld == nil {
				continue
			}
			if _, ok := c.LDrawColors[*ld]; !ok {
				c.LDrawColors[*ld] = id
			}
		}
		return nil
	})
}

// rows calls fn with a scanner for every row of q.
func rows(db *sql.DB, q string, fn func(scan func(...any) error) error) error {
	rs, err := db.Query(q)
	if err != nil {
		return err
	}
	defer rs.Close()
	for rs.Next() {
		if err := fn(rs.Scan); err != nil {
			return err
		}
	}
	return rs.Err()
}
