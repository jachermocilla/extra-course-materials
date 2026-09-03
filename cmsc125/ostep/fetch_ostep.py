#!/usr/bin/env python3
"""
Build a single-PDF corpus for tropa from the *free* per-chapter PDFs of
"Operating Systems: Three Easy Pieces" (OSTEP), https://pages.cs.wisc.edu/~remzi/OSTEP/

OSTEP is published as one PDF per chapter, each paginated from 1. tropa's
`ingest` wants one PDF, so this script:

  1. scrapes the OSTEP index page for the chapter table,
  2. downloads every numbered chapter PDF (cached under book/.ostep-cache/),
  3. concatenates them in chapter order into  book/ostep.pdf,
  4. stamps a running "p. N" on every page (bottom-right, grey) and adds a
     clickable chapter bookmark tree, so the merged file can be navigated to a
     cited page,
  5. writes the sidecar  book/ostep.toc.json  that tropa's manual-TOC override
     (`manual_toc_headings` in textbook_tropa.py) reads -- each entry is the
     1-indexed start page of that chapter *within the combined PDF*.

Because the source chapters restart page numbering, citations after ingest
refer to the running page number inside book/ostep.pdf (chapter 1 starts at
p. 1) -- the number now stamped on each page -- not any printed OSTEP edition's
global page numbers. Ingest with --page-offset=0.

Usage:
    python scripts/fetch_ostep.py [--skip-dialogues] [--force]

Then:
    python textbook_tropa.py ingest ./book/ostep.pdf --page-offset=0 \
        --title="Operating Systems: Three Easy Pieces" \
        --author="Remzi H. Arpaci-Dusseau and Andrea C. Arpaci-Dusseau"
"""

import argparse
import html
import json
import re
import sys
import time
import urllib.request
from pathlib import Path

try:
    import pymupdf as fitz
except ImportError:  # older wheels
    import fitz

INDEX_URL = "https://pages.cs.wisc.edu/~remzi/OSTEP/"
USER_AGENT = "textbook-tropa fetch-ostep (+https://pages.cs.wisc.edu/~remzi/OSTEP/)"

ROOT = Path(__file__).resolve().parent.parent
BOOK_DIR = ROOT / "book"
CACHE_DIR = BOOK_DIR / ".ostep-cache"
OUT_PDF = BOOK_DIR / "ostep.pdf"
OUT_TOC = BOOK_DIR / "ostep.toc.json"

# <small>N</small> [<i>] <a href=foo.pdf ...>Title</a>  -- one chapter cell
CHAPTER_RE = re.compile(
    r"<small>\s*(\d+)\s*</small>\s*(?:<i>\s*)?"
    r"<a\s+href=([A-Za-z0-9-]+\.pdf)[^>]*>(.*?)</a>",
    re.S | re.I,
)

# The index's link text for dialogues is just "Dialogue"/"Summary"; give the
# prepended-as-embedding-context section titles something more descriptive.
NICER_TITLES = {
    "dialogue-threeeasy.pdf": "A Dialogue on the Book",
    "dialogue-virtualization.pdf": "A Dialogue on Virtualization",
    "dialogue-vm.pdf": "A Dialogue on Memory Virtualization",
    "dialogue-concurrency.pdf": "A Dialogue on Concurrency",
    "dialogue-persistence.pdf": "A Dialogue on Persistence",
    "dialogue-distribution.pdf": "A Dialogue on Distribution",
    "dialogue-security.pdf": "A Dialogue on Security",
    "cpu-dialogue.pdf": "Summary Dialogue on CPU Virtualization",
    "vm-dialogue.pdf": "Summary Dialogue on Memory Virtualization",
    "threads-dialogue.pdf": "Summary Dialogue on Concurrency",
    "file-dialogue.pdf": "Summary Dialogue on Persistence",
    "dist-dialogue.pdf": "Summary Dialogue on Distribution",
}


def fetch(url, binary=False):
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    with urllib.request.urlopen(req, timeout=60) as resp:
        data = resp.read()
    return data if binary else data.decode("utf-8", "replace")


def parse_chapters(index_html):
    """[(chapter_number, href, title), ...] in chapter order, deduped."""
    start = index_html.find('name="book-chapters"')
    region = index_html[start:] if start >= 0 else index_html
    for marker in ('name="news"', 'name="homework"', "Homework"):
        cut = region.find(marker)
        if cut > 0:
            region = region[:cut]
            break

    seen, rows = set(), []
    for m in CHAPTER_RE.finditer(region):
        num = int(m.group(1))
        href = m.group(2)
        if href in seen:
            continue
        seen.add(href)
        title = html.unescape(re.sub(r"\s+", " ", re.sub(r"<[^>]+>", "", m.group(3))).strip())
        rows.append((num, href, NICER_TITLES.get(href, title)))
    rows.sort()
    return rows


def download_all(chapters, force=False):
    CACHE_DIR.mkdir(parents=True, exist_ok=True)
    paths = []
    for num, href, title in chapters:
        dest = CACHE_DIR / href
        if dest.exists() and dest.stat().st_size > 0 and not force:
            print(f"  cached  ch {num:>2}  {href}")
        else:
            print(f"  get     ch {num:>2}  {href}")
            dest.write_bytes(fetch(INDEX_URL + href, binary=True))
            time.sleep(0.4)  # be polite to a university server
        paths.append((num, href, title, dest))
    return paths


def stamp_page_numbers(doc):
    """Draw the running 1..N page number on every page, bottom-right in small
    grey type. The merged file otherwise carries no continuous numbering (each
    source chapter restarts at 1), so this is what makes tropa's `(p. N)`
    citations -- which equal this sheet order -- something a reader can turn to.
    Bottom-right avoids OSTEP's own chapter page number (bottom-centre on odd
    pages, top-outer on even)."""
    for i, page in enumerate(doc, start=1):
        r = page.rect
        box = fitz.Rect(r.width - 90, r.height - 26, r.width - 14, r.height - 8)
        page.insert_textbox(box, f"p. {i}", fontsize=8, fontname="helv",
                            color=(0.5, 0.5, 0.5), align=fitz.TEXT_ALIGN_RIGHT)


def build(paths):
    combined = fitz.open()
    toc = []
    for num, href, title, path in paths:
        start_page = combined.page_count + 1  # 1-indexed, within the combined PDF
        with fitz.open(path) as part:
            combined.insert_pdf(part)
        toc.append({"start": start_page, "title": f"Chapter {num}: {title}"})
    stamp_page_numbers(combined)
    # clickable chapter bookmarks in the merged file
    combined.set_toc([[1, e["title"], e["start"]] for e in toc])
    BOOK_DIR.mkdir(parents=True, exist_ok=True)
    combined.save(OUT_PDF, garbage=4, deflate=True)
    total = combined.page_count
    combined.close()
    OUT_TOC.write_text(json.dumps(toc, indent=2) + "\n")
    return total, toc


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--skip-dialogues", action="store_true",
                    help="omit the 'Dialogue'/'Summary' interlude chapters")
    ap.add_argument("--force", action="store_true", help="re-download cached PDFs")
    args = ap.parse_args()

    print(f"Fetching index: {INDEX_URL}")
    chapters = parse_chapters(fetch(INDEX_URL))
    if not chapters:
        sys.exit("error: no chapters parsed from the index page (layout changed?)")

    if args.skip_dialogues:
        chapters = [c for c in chapters
                    if not (c[1].startswith("dialogue-") or c[1].endswith("-dialogue.pdf"))]

    nums = [c[0] for c in chapters]
    print(f"  {len(chapters)} chapters (ch {min(nums)}-{max(nums)})")

    paths = download_all(chapters, force=args.force)

    print("Concatenating ...")
    total, toc = build(paths)

    print()
    print(f"  wrote {OUT_PDF.relative_to(ROOT)}  ({total} pages)")
    print(f"  wrote {OUT_TOC.relative_to(ROOT)}  ({len(toc)} entries, first start={toc[0]['start']})")
    print()
    print("Next: python textbook_tropa.py ingest ./book/ostep.pdf --page-offset=0 \\")
    print('        --title="Operating Systems: Three Easy Pieces" \\')
    print('        --author="Remzi H. Arpaci-Dusseau and Andrea C. Arpaci-Dusseau"')


if __name__ == "__main__":
    main()
