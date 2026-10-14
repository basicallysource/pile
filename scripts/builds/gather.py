"""Gathers the LDraw Official Model Repository's model of every set it has
into a folder: each set's main model file, and index.json listing them (the
set, the model's author, what it lacks, its page). pile's builds are made
from them: docs/builds.md.

    python3 scripts/builds/gather.py DATA/set-models

Run again, it fetches only what it lacks. The files are LDraw models shared
under CCAL 2.0 (attribution): keep the authors with them.
"""

import html
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

SITE = "https://library.ldraw.org"
# Seconds between requests, to go easy on the site.
PAUSE = 0.5


def get(url):
    req = urllib.request.Request(url, headers={"User-Agent": "pile-gather (github.com/basicallysource/pile)"})
    for attempt in range(3):
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                body = r.read()
            time.sleep(PAUSE)
            return body
        except urllib.error.HTTPError as e:
            if e.code == 404:
                raise
            print(f"  {url}: {e}; again", file=sys.stderr)
            time.sleep(5 * (attempt + 1))
        except Exception as e:
            print(f"  {url}: {e}; again", file=sys.stderr)
            time.sleep(5 * (attempt + 1))
    with urllib.request.urlopen(req, timeout=60) as r:
        return r.read()


def text(s):
    s = re.sub(r"<(script|style)\b.*?</\1>", "", s, flags=re.S)
    return html.unescape(re.sub(r"\s+", " ", re.sub(r"<[^>]+>", " ", s))).strip()


def sets():
    """Every set in the repository: its page id, number, name, theme, year."""
    out = {}
    page = 1
    while True:
        t = get(f"{SITE}/omr/sets?page={page}").decode()
        cells = re.findall(rf'<a\s[^>]*?href="{SITE}/omr/sets/(\d+)"[^>]*>(.*?)</a>', t, re.S)
        if not cells:
            return out
        for pid, cell in cells:
            c = text(cell)
            if c:
                out.setdefault(pid, []).append(c)
        page += 1


def models(pid):
    """The models on a set's page: each one's label ("Main Model" or an
    alternate's name), author, what it lacks, and its file."""
    t = get(f"{SITE}/omr/sets/{pid}").decode()
    out = []
    prev = 0
    for m in re.finditer(rf'href="({SITE}/library/omr/[^"]+)"', t):
        seg = text(t[prev : m.start()])
        prev = m.end()
        # The segment ends inside the link's own tag, which text() leaves.
        info = re.search(
            r"(.*?) Author: (.*?) Missing Parts (\w+) Missing Patterns (\w+) Missing Stickers (\w+)(?: Notes: (.*?))? 3D View\b",
            seg,
        )
        if not info:
            continue
        before, author, parts, patterns, stickers, notes = info.groups()
        # The label follows the page's "Models" heading (the first model) or
        # the model before's Download button.
        out.append(
            {
                "label": re.split(r"\bModels |\bDownload ", before)[-1].strip(),
                "author": author,
                "missingParts": parts == "Yes",
                "missingPatterns": patterns == "Yes",
                "missingStickers": stickers == "Yes",
                "notes": notes or "",
                "url": m.group(1),
            }
        )
    return out


def main(out):
    os.makedirs(out, exist_ok=True)
    path = os.path.join(out, "index.json")
    index = {}
    if os.path.exists(path):
        index = {e["set"]: e for e in json.load(open(path))}

    def save():
        with open(path + ".new", "w") as f:
            json.dump(sorted(index.values(), key=lambda e: e["set"]), f, indent=1)
        os.replace(path + ".new", path)

    listed = sets()
    print(f"{len(listed)} sets listed", file=sys.stderr)
    for n, (pid, cells) in enumerate(sorted(listed.items(), key=lambda kv: kv[1][0])):
        num, name = cells[0], cells[1] if len(cells) > 1 else ""
        have = index.get(num)
        if have and os.path.exists(os.path.join(out, have["file"])):
            continue
        ms = models(pid)
        main_model = next((m for m in ms if m["label"] == "Main Model"), ms[0] if ms else None)
        if not main_model:
            print(f"{num}: no model on its page", file=sys.stderr)
            continue
        file = main_model["url"].rsplit("/", 1)[1]
        try:
            body = get(main_model["url"])
        except urllib.error.HTTPError as e:
            print(f"{num}: {main_model['url']}: {e}; skipped", file=sys.stderr)
            continue
        with open(os.path.join(out, file), "wb") as f:
            f.write(body)
        index[num] = {
            "set": num,
            "name": name,
            "file": file,
            "author": main_model["author"],
            "missingParts": main_model["missingParts"],
            "missingPatterns": main_model["missingPatterns"],
            "missingStickers": main_model["missingStickers"],
            "notes": main_model["notes"],
            "page": f"{SITE}/omr/sets/{pid}",
        }
        if n % 25 == 0:
            save()
            print(f"{n}/{len(listed)} {num}", file=sys.stderr)
    save()
    print(f"{len(index)} models in {out}", file=sys.stderr)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    main(sys.argv[1])
