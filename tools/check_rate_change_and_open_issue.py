"""
Check current CEMC rates against the snapshot; if different,
open a GitHub issue (for use in GitHub Actions).

Uses the GITHUB_TOKEN and GITHUB_REPOSITORY env vars
provided automatically by GitHub Actions.
"""

from __future__ import annotations

import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict

import requests

from cemc_rates.downloader import download_pdf
from cemc_rates.parser import parse_pdf
from cemc_rates.diffing import (
    DEFAULT_SNAPSHOT_PATH,
    load_snapshot,
    extract_residential_summary,
    diff_summaries,
    format_changes_markdown,
)


def load_current_summary(force: bool = False) -> Dict[str, Any]:
    pdf_path = download_pdf(force=force)
    parsed = parse_pdf(str(pdf_path))
    return extract_residential_summary(parsed)


def open_github_issue(title: str, body: str) -> None:
    repo = os.environ.get("GITHUB_REPOSITORY")
    token = os.environ.get("GITHUB_TOKEN")

    if not repo:
        raise RuntimeError("GITHUB_REPOSITORY not set")
    if not token:
        raise RuntimeError("GITHUB_TOKEN not set")

    url = f"https://api.github.com/repos/{repo}/issues"

    headers = {
        "Authorization": f"Bearer {token}",
        "Accept": "application/vnd.github+json",
    }

    payload = {
        "title": title,
        "body": body,
    }

    resp = requests.post(url, headers=headers, json=payload, timeout=30)
    if resp.status_code >= 300:
        raise RuntimeError(
            f"Failed to create issue. Status: {resp.status_code}, body: {resp.text}"
        )


def main() -> None:
    snapshot_path = DEFAULT_SNAPSHOT_PATH

    # Strategy:
    #  1. Load snapshot
    #  2. Parse current PDF
    #  3. Compute diff
    #  4. If changes, open issue and print to logs; else print "no change"

    print(f"Using snapshot: {snapshot_path}")

    baseline = load_snapshot(snapshot_path)

    baseline_summary = {
        "residential_standard": baseline["residential_standard"],
        "residential_supplemental": baseline["residential_supplemental"],
    }

    current_summary = load_current_summary(force=True)
    changes = diff_summaries(current_summary, baseline_summary)

    if not changes:
        print("No rate changes detected. No issue will be created.")
        return

    # Build issue title & body
    now = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    title = f"CEMC rate change detected – {now}"
    body_lines = []

    body_lines.append(
        "A difference between the current CEMC rates PDF and the stored snapshot "
        "has been detected.\n"
    )

    body_lines.append("## Summary of changes\n")
    body_lines.append(format_changes_markdown(changes))
    body_lines.append("\n---\n")
    body_lines.append(f"Snapshot file: `{snapshot_path}`")

    body = "\n".join(body_lines)

    print("Rate changes detected; creating GitHub issue...")
    open_github_issue(title, body)
    print("Issue created successfully.")


if __name__ == "__main__":
    main()
