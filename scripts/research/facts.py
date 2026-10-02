#!/usr/bin/env python3
"""Facts about what sorters sorted, for telling: what a lot is made of, its
long tail, its weight and value, how old its colors are, and how fast and how
cleanly the machines sorted it.

It reads lots written by `bin/export-pieces` (CSV), and from pile's data
folder `hive.sqlite` (every piece's status and time), `parts.db` (BrickLink
weights, sold prices and part sizes, as a sorter Hive caches them) and the
Rebrickable CSVs. Standard library and numpy only.

    facts.py --data DIR --lot NAME=FILE.csv [--lot NAME=FILE.csv] --out facts.json

Prints a summary and writes every number to --out.
"""

import argparse
import collections
import csv
import datetime
import gzip
import json
import math
import sqlite3

import numpy as np

GRAMS_PER_POUND = 453.59237
OLD_NEW = {"Light Gray (old) to Light Bluish Gray": (7, 71), "Dark Gray (old) to Dark Bluish Gray": (8, 72), "Brown (old) to Reddish Brown": (6, 70)}
SURE = 0.8  # classifier confidence for the "rarest" and "biggest" picks
SURER = 0.9  # and for "most valuable", where one misread color makes a fortune


def read_lot(path):
    rows = []
    with open(path) as f:
        for r in csv.DictReader(f):
            r["count"] = int(r["count"])
            r["mean_confidence"] = float(r["mean_confidence"])
            r["sets_with_part"] = int(r["sets_with_part"]) if r["sets_with_part"] else None
            rows.append(r)
    return rows


def parts_db(path):
    db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    weight = {i: w for i, w in db.execute("select item_no, weight from bricklink_items where weight > 0")}
    price = {(i, c): p for i, c, p in db.execute("select item_no, bl_color_id, ord_used_avg from price_guides where ord_used_avg > 0")}
    size = {p: e for p, e in db.execute("select part_num, max_extent_mm from part_geometry where max_extent_mm > 0")}
    synced = dict(db.execute("select key, value from meta").fetchall())
    return weight, price, size, synced


def rarefy(counts, n):
    """Expected distinct kinds in a random n of the pieces (without replacement, approximated with)."""
    N = counts.sum()
    if n >= N:
        return float(len(counts))
    return float(np.sum(1 - np.exp(n * np.log1p(-counts / N))))


def lot_facts(rows, weight, price, size):
    out = {}
    N = sum(r["count"] for r in rows)
    out["pieces"] = N
    by_pc = collections.Counter()
    by_mold = collections.Counter()
    by_color = collections.Counter()
    by_cat = collections.Counter()
    names = {}
    unknown_color = unmatched = 0
    grams = weighed = 0.0
    value = priced = 0.0
    big = []
    for r in rows:
        n = r["count"]
        mold = r["rb_canonical"] or ("bl:" + r["bl_part"])
        color = r["rb_color_name"] or ("unknown" if not r["bl_color"] else r["color_name"])
        if not r["rb_part"]:
            unmatched += n
        if not r["bl_color"]:
            unknown_color += n
        by_pc[(mold, color)] += n
        by_mold[mold] += n
        by_color[color] += n
        by_cat[r["category"] or "not in the catalog"] += n
        names[mold] = r["rb_part_name"] or r["part_name"]
        if r["bl_part"] in weight:
            grams += n * weight[r["bl_part"]]
            weighed += n
        if r["bl_color"] and (r["bl_part"], int(r["bl_color"])) in price:
            value += n * price[(r["bl_part"], int(r["bl_color"]))]
            priced += n
        if r["rb_part"] in size:
            big.append((size[r["rb_part"]], n, r))
    out["unknown_color_share"] = unknown_color / N
    out["not_in_catalog_share"] = unmatched / N
    out["part_colors"] = len(by_pc)
    out["molds"] = len(by_mold)
    out["colors"] = len([c for c in by_color if c != "unknown"])
    out["top_part_colors"] = [{"part": m, "name": names[m], "color": c, "pieces": n, "share": n / N} for (m, c), n in by_pc.most_common(15)]
    out["top_molds"] = [{"part": m, "name": names[m], "pieces": n, "share": n / N} for m, n in by_mold.most_common(15)]
    out["top_colors"] = [{"color": c, "pieces": n, "share": n / N} for c, n in by_color.most_common(15)]
    out["categories"] = [{"category": c, "pieces": n, "share": n / N} for c, n in by_cat.most_common(20)]
    counts = np.array(sorted(by_pc.values(), reverse=True), dtype=float)
    cum = np.cumsum(counts) / N
    out["part_colors_for_share"] = {f"{q:.0%}": int(np.searchsorted(cum, q) + 1) for q in (0.25, 0.5, 0.8, 0.9)}
    singles = int(np.sum(counts == 1))
    out["part_colors_seen_once"] = singles
    out["chance_next_piece_is_new"] = singles / N  # Good-Turing
    mold_counts = np.array(list(by_mold.values()), dtype=float)
    out["molds_seen_once"] = int(np.sum(mold_counts == 1))
    out["chance_next_piece_is_new_mold"] = int(np.sum(mold_counts == 1)) / N

    mean_g = grams / weighed if weighed else float("nan")
    out["weighed_share"] = weighed / N
    out["grams_per_piece"] = mean_g
    out["pieces_per_pound"] = GRAMS_PER_POUND / mean_g
    out["pounds"] = N * mean_g / GRAMS_PER_POUND
    out["priced_share"] = priced / N
    out["usd_per_piece"] = value / priced if priced else float("nan")
    out["usd_per_pound"] = (value / priced) * GRAMS_PER_POUND / mean_g if priced else float("nan")
    ppl = GRAMS_PER_POUND / mean_g
    out["distinct_by_pounds"] = [
        {"pounds": lb, "pieces": round(lb * ppl), "part_colors": rarefy(counts, lb * ppl), "molds": rarefy(mold_counts, lb * ppl)}
        for lb in (1, 5, 10, 25, 50, 100, 250, 500, 1000) if lb * ppl <= N
    ]

    old = {}
    for label, (o, n) in OLD_NEW.items():
        co = sum(r["count"] for r in rows if r["rb_color"] == str(o))
        cn = sum(r["count"] for r in rows if r["rb_color"] == str(n))
        old[label] = {"old": co, "new": cn, "old_share": co / (co + cn) if co + cn else None}
    out["old_colors"] = old

    # Counted parts only: printed and minifigure parts have no count of sets.
    sure = [r for r in rows if r["mean_confidence"] >= SURE and r["rb_part"] and r["sets_with_part"]]
    rare = sorted(sure, key=lambda r: (r["sets_with_part"], -r["count"]))[:15]
    out["rarest_molds_seen"] = [{"part": r["rb_part"], "name": r["rb_part_name"], "color": r["rb_color_name"], "sets_with_part": r["sets_with_part"], "pieces": r["count"], "machine": r["machine"], "confidence": r["mean_confidence"]} for r in rare]
    valued = []
    for r in sure:
        if r["mean_confidence"] >= SURER and r["bl_color"] and (r["bl_part"], int(r["bl_color"])) in price:
            valued.append((price[(r["bl_part"], int(r["bl_color"]))], r))
    valued.sort(key=lambda x: -x[0])
    out["most_valuable_seen"] = [{"part": r["rb_part"], "name": r["rb_part_name"], "color": r["rb_color_name"], "usd_used_sold_avg": p, "pieces": r["count"], "machine": r["machine"], "confidence": r["mean_confidence"]} for p, r in valued[:15]]
    big.sort(key=lambda x: -x[0])
    out["biggest_seen"] = [{"part": r["rb_part"], "name": r["rb_part_name"], "max_extent_mm": e, "pieces": n, "confidence": r["mean_confidence"]} for e, n, r in big if r["mean_confidence"] >= SURE][:10]
    sized = sum(n for _, n, _ in big)
    out["share_over_mm"] = {str(mm): sum(n for e, n, _ in big if e > mm) / sized for mm in (32, 48, 64, 80)} if sized else {}

    per_machine = collections.defaultdict(lambda: [0, set()])
    for r in rows:
        m = per_machine[r["machine"]]
        m[0] += r["count"]
        m[1].add((r["rb_canonical"] or r["bl_part"], r["rb_color"] or r["bl_color"]))
    out["machines"] = sorted(({"machine": k, "pieces": v[0], "part_colors": len(v[1])} for k, v in per_machine.items()), key=lambda x: -x["pieces"])
    return out


def hive_facts(path, which_machines=None):
    """Statuses, sorting time and best days, per machine, from hive.sqlite."""
    db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
    q = """select m.id, m.name, m.mine, p.seen_at, p.classification_status, p.dead from pieces p join machines m on m.id = p.machine_id
           where p.seen_at is not null order by m.id, p.seen_at"""
    out = collections.OrderedDict()
    last = {}
    for mid, name, mine, seen, status, dead in db.execute(q):
        m = out.setdefault(mid, {"name": name, "mine": bool(mine), "statuses": collections.Counter(), "seconds": 0.0, "days": collections.Counter(), "hours": collections.Counter(), "first": seen, "last": seen})
        m["statuses"][status or "none"] += 1
        ok = status == "classified" and not dead
        if ok:
            t = datetime.datetime.fromtimestamp(seen)
            m["days"][t.date().isoformat()] += 1
            m["hours"][t.strftime("%Y-%m-%d %H")] += 1
        gap = seen - last.get(mid, seen)
        if 0 < gap <= 60:
            m["seconds"] += gap
        last[mid] = seen
        m["last"] = seen
    res = []
    for m in out.values():
        name = m["name"]
        classified = m["statuses"]["classified"]
        total = sum(m["statuses"].values())
        hours = m["seconds"] / 3600
        best_day = m["days"].most_common(1)
        best_hour = m["hours"].most_common(1)
        res.append({
            "machine": name, "mine": m["mine"], "pieces": total, "classified": classified,
            "statuses": dict(m["statuses"]), "sorting_hours": hours,
            "classified_per_hour": classified / hours if hours else None,
            "pieces_per_hour": total / hours if hours else None,
            "best_day": best_day[0] if best_day else None, "best_hour": best_hour[0] if best_hour else None,
            "first": datetime.datetime.fromtimestamp(m["first"]).date().isoformat(),
            "last": datetime.datetime.fromtimestamp(m["last"]).date().isoformat(),
        })
    return sorted(res, key=lambda x: -x["pieces"])


def catalog_facts(rb):
    """How widely LEGO uses its molds: the parts in the most sets."""
    first = {}
    with gzip.open(f"{rb}/inventories.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            v = int(r["version"])
            if r["set_num"].startswith("fig-"):
                continue
            if r["set_num"] not in first or v < first[r["set_num"]][1]:
                first[r["set_num"]] = (r["id"], v)
    inv = {i: s for s, (i, _) in first.items()}
    sets_with = collections.defaultdict(set)
    qty = collections.Counter()
    with gzip.open(f"{rb}/inventory_parts.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            s = inv.get(r["inventory_id"])
            if s and r["is_spare"] != "True":
                sets_with[r["part_num"]].add(s)
                qty[(r["part_num"], r["color_id"])] += int(r["quantity"])
    names = {}
    with gzip.open(f"{rb}/parts.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            names[r["part_num"]] = r["name"]
    colors = {}
    with gzip.open(f"{rb}/colors.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            colors[r["id"]] = r["name"]
    widest = sorted(sets_with.items(), key=lambda x: -len(x[1]))[:10]
    total = sum(qty.values())
    return {
        "sets": len(inv),
        "pieces_in_all_sets": total,
        "parts_in_most_sets": [{"part": p, "name": names.get(p), "sets": len(s)} for p, s in widest],
        "most_used_part_colors": [{"part": p, "name": names.get(p), "color": colors.get(c), "pieces": n, "share": n / total} for (p, c), n in qty.most_common(10)],
    }


def set_facts(rb, sets, stream_rows):
    """For a few sets: what they are made of, how much of each is the
    commonest pieces in the stream, and how much of one is in another."""
    first = {}
    with gzip.open(f"{rb}/inventories.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            v = int(r["version"])
            if r["set_num"] in sets and (r["set_num"] not in first or v < first[r["set_num"]][1]):
                first[r["set_num"]] = (r["id"], v)
    inv = {i: s for s, (i, _) in first.items()}
    cats = {}
    with gzip.open(f"{rb}/part_categories.csv.gz", "rt") as f:
        cats = {r["id"]: r["name"] for r in csv.DictReader(f)}
    part_cat, names = {}, {}
    with gzip.open(f"{rb}/parts.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            part_cat[r["part_num"]] = cats.get(r["part_cat_id"], "")
            names[r["part_num"]] = r["name"]
    printed = set()
    with gzip.open(f"{rb}/part_relationships.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            if r["rel_type"] == "P":
                printed.add(r["child_part_num"])
    colors = {}
    with gzip.open(f"{rb}/colors.csv.gz", "rt") as f:
        colors = {r["id"]: r["name"] for r in csv.DictReader(f)}
    lines = collections.defaultdict(collections.Counter)
    with gzip.open(f"{rb}/inventory_parts.csv.gz", "rt") as f:
        for r in csv.DictReader(f):
            s = inv.get(r["inventory_id"])
            if not s or r["is_spare"] == "True":
                continue
            c = part_cat.get(r["part_num"], "")
            if c.startswith("Minifig") or c == "Stickers" or r["part_num"] in printed:
                continue
            lines[s][(r["part_num"], r["color_id"])] += int(r["quantity"])
    # The stream's commonest part-colors, in Rebrickable's ids.
    stream = collections.Counter()
    for r in stream_rows:
        if r["rb_part"] and r["rb_color"]:
            stream[(r["rb_part"], r["rb_color"])] += r["count"]
    top100 = {k for k, _ in stream.most_common(100)}
    top1000 = {k for k, _ in stream.most_common(1000)}
    out = {}
    for s, ls in lines.items():
        total = sum(ls.values())
        top = ls.most_common(5)
        out[s] = {
            "counted_pieces": total,
            "part_colors": len(ls),
            "top_lines": [{"part": p, "name": names.get(p), "color": colors.get(c), "quantity": q} for (p, c), q in top],
            "share_in_stream_top100": sum(q for k, q in ls.items() if k in top100) / total,
            "share_in_stream_top1000": sum(q for k, q in ls.items() if k in top1000) / total,
            "share_seen_in_stream": sum(q for k, q in ls.items() if k in stream) / total,
        }
    overlap = {}
    for a in lines:
        for b in lines:
            if a != b:
                shared = sum(min(q, lines[b].get(k, 0)) for k, q in lines[a].items())
                overlap[f"{a} in {b}"] = shared / sum(lines[a].values())
    return {"sets": out, "overlap": overlap}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", required=True)
    ap.add_argument("--lot", action="append", default=[], help="NAME=FILE.csv from export-pieces")
    ap.add_argument("--sets", default="75192-1,6212-1,10179-1,75301-1", help="sets to describe, comma separated")
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    weight, price, size, synced = parts_db(f"{a.data}/parts.db")
    facts = {"prices_and_weights_as_of": synced.get("profile_catalog.last_sync.prices"), "lots": {}}
    for spec in a.lot:
        name, path = spec.split("=", 1)
        facts["lots"][name] = lot_facts(read_lot(path), weight, price, size)
    facts["machines"] = hive_facts(f"{a.data}/hive.sqlite")
    facts["catalog"] = catalog_facts(f"{a.data}/rebrickable")
    if a.lot:
        name, path = a.lot[-1].split("=", 1)
        facts["set_facts"] = set_facts(f"{a.data}/rebrickable", set(a.sets.split(",")), read_lot(path))
        facts["set_facts"]["stream"] = name
    with open(a.out, "w") as f:
        json.dump(facts, f, indent=1, default=str)
    for name, l in facts["lots"].items():
        print(f"\n== {name}: {l['pieces']:,} pieces, {l['part_colors']:,} part-colors, {l['molds']:,} molds, {l['pounds']:.0f} lb")
        print(f"  {l['pieces_per_pound']:.0f} pieces a pound, ${l['usd_per_pound']:.2f} a pound used sold; next piece new part-color {l['chance_next_piece_is_new']:.2%}")
        print("  for share:", l["part_colors_for_share"])
        print("  top:", [(x["name"], x["color"], f"{x['share']:.2%}") for x in l["top_part_colors"][:5]])
        print("  old colors:", {k: f"{v['old_share']:.1%}" for k, v in l["old_colors"].items() if v["old_share"] is not None})
        print("  distinct by pounds:", [(d["pounds"], round(d["part_colors"]), round(d["molds"])) for d in l["distinct_by_pounds"]])
    print()
    for m in facts["machines"][:12]:
        print(m["machine"], m["pieces"], m["classified"], f"{m['sorting_hours']:.0f} h", m["classified_per_hour"] and f"{m['classified_per_hour']:.0f}/h", m["best_day"])


if __name__ == "__main__":
    main()
