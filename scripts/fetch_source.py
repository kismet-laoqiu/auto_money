#!/usr/bin/env python3.11
from __future__ import annotations

import argparse
import csv
import re
from datetime import datetime, timezone
from pathlib import Path
from typing import Iterable
from urllib.parse import urlparse

import requests
import trafilatura

ROOT = Path('/root/quant-lab')
RAW_DIR = ROOT / 'docs' / 'research' / 'raw'
TSV_PATH = ROOT / 'docs' / 'research' / 'sources.tsv'
TIMEOUT = 30


def slugify(value: str) -> str:
    value = value.strip().lower()
    value = re.sub(r'[^a-z0-9]+', '-', value)
    value = value.strip('-')
    return value or 'source'


def detect_extension(content_type: str, url: str) -> str:
    content_type = (content_type or '').lower()
    path = urlparse(url).path.lower()
    if 'pdf' in content_type or path.endswith('.pdf'):
        return '.pdf'
    if 'html' in content_type or path.endswith('.html') or path.endswith('.htm') or not Path(path).suffix:
        return '.md'
    suffix = Path(path).suffix
    return suffix if suffix else '.txt'


def html_to_markdown(url: str, html: str) -> str:
    extracted = trafilatura.extract(
        html,
        url=url,
        output_format='markdown',
        include_comments=False,
        include_tables=True,
        favor_precision=True,
        deduplicate=True,
    )
    if extracted:
        return extracted
    return html


def read_rows(path: Path) -> list[dict[str, str]]:
    if not path.exists():
        return []
    with path.open('r', encoding='utf-8', newline='') as handle:
        reader = csv.DictReader(handle, delimiter='\t')
        return list(reader)


def write_rows(path: Path, rows: Iterable[dict[str, str]]) -> None:
    fieldnames = [
        'id',
        'theory',
        'stance',
        'source_type',
        'title',
        'author_or_org',
        'year',
        'url',
        'local_raw_path',
        'status',
        'notes',
    ]
    with path.open('w', encoding='utf-8', newline='') as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter='\t')
        writer.writeheader()
        writer.writerows(rows)


def upsert_row(path: Path, row: dict[str, str]) -> None:
    rows = read_rows(path)
    replaced = False
    for index, existing in enumerate(rows):
        if existing['id'] == row['id']:
            rows[index] = row
            replaced = True
            break
    if not replaced:
        rows.append(row)
    rows.sort(key=lambda item: item['id'])
    write_rows(path, rows)


def main() -> int:
    parser = argparse.ArgumentParser(description='Capture a research source into docs/research/raw and sources.tsv')
    parser.add_argument('--id', required=True)
    parser.add_argument('--theory', required=True)
    parser.add_argument('--stance', required=True)
    parser.add_argument('--source-type', required=True)
    parser.add_argument('--title', required=True)
    parser.add_argument('--author-or-org', required=True)
    parser.add_argument('--year', required=True)
    parser.add_argument('--url', required=True)
    parser.add_argument('--notes', default='')
    parser.add_argument('--status', default='captured')
    args = parser.parse_args()

    RAW_DIR.mkdir(parents=True, exist_ok=True)
    TSV_PATH.parent.mkdir(parents=True, exist_ok=True)

    response = requests.get(args.url, timeout=TIMEOUT, headers={'User-Agent': 'quant-lab-research/0.1'})
    response.raise_for_status()

    extension = detect_extension(response.headers.get('Content-Type', ''), args.url)
    filename = f"{args.id}-{slugify(args.title)}{extension}"
    raw_path = RAW_DIR / filename

    fetched_at = datetime.now(timezone.utc).isoformat()
    if extension == '.md':
        body = html_to_markdown(args.url, response.text)
        content = (
            f"# {args.title}\n\n"
            f"- Source URL: {args.url}\n"
            f"- Author/Org: {args.author_or_org}\n"
            f"- Year: {args.year}\n"
            f"- Captured At UTC: {fetched_at}\n"
            f"- Theory: {args.theory}\n"
            f"- Stance: {args.stance}\n\n"
            f"---\n\n{body}\n"
        )
        raw_path.write_text(content, encoding='utf-8')
    else:
        raw_path.write_bytes(response.content)

    row = {
        'id': args.id,
        'theory': args.theory,
        'stance': args.stance,
        'source_type': args.source_type,
        'title': args.title,
        'author_or_org': args.author_or_org,
        'year': args.year,
        'url': args.url,
        'local_raw_path': str(raw_path.relative_to(ROOT)),
        'status': args.status,
        'notes': args.notes,
    }
    upsert_row(TSV_PATH, row)

    print(f'SAVED_PATH={raw_path}')
    print(f'TSV_ID={args.id}')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
