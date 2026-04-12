#!/usr/bin/env python3
"""Search a ZIM file and return results as JSON.

Requires: pip install libzim

Usage:
    python3 script/zim-search.py --zim /opt/zim/wikipedia_en_all_nopic_2025-12.zim \
        --query "apophenia clinical schizophrenia" --limit 5

Output (JSON array):
    [{"title": "Apophenia", "path": "Apophenia", "snippet": "..."}]

The snippet is the first ~500 characters of the article's text content
(HTML tags stripped). If the ZIM file lacks a fulltext index, exits 1.
"""

import argparse
import json
import re
import sys

from libzim.reader import Archive
from libzim.search import Query, Searcher


def strip_html(html: str) -> str:
    """Remove HTML tags, collapse whitespace."""
    text = re.sub(r"<[^>]+>", " ", html)
    return re.sub(r"\s+", " ", text).strip()


def main():
    p = argparse.ArgumentParser(description="Search a ZIM file, return JSON.")
    p.add_argument("--zim", required=True, help="Path to .zim file")
    p.add_argument("--query", required=True, help="Fulltext search query")
    p.add_argument("--limit", type=int, default=5, help="Max results")
    p.add_argument("--snippet-len", type=int, default=500, help="Snippet length")
    args = p.parse_args()

    try:
        archive = Archive(args.zim)
    except Exception as e:
        print(f"error: cannot open ZIM file: {e}", file=sys.stderr)
        sys.exit(1)

    if not archive.has_fulltext_index:
        print("error: ZIM file has no fulltext index", file=sys.stderr)
        sys.exit(1)

    searcher = Searcher(archive)
    query = Query().set_query(args.query)
    results = searcher.search(query)

    out = []
    for path in results.getResults(0, args.limit):
        try:
            entry = archive.get_entry_by_path(path)
            item = entry.get_item()
            content = bytes(item.content).decode("utf-8", errors="replace")
            snippet = strip_html(content)[: args.snippet_len]
            out.append(
                {"title": entry.title, "path": path, "snippet": snippet}
            )
        except Exception:
            continue

    json.dump(out, sys.stdout, indent=2)
    print()


if __name__ == "__main__":
    main()
