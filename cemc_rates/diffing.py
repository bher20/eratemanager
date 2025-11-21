from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Tuple


DEFAULT_SNAPSHOT_PATH = Path("snapshots/residential_v1.json")


@dataclass
class FieldChange:
    section: str
    field: str
    old: Any
    new: Any


def load_snapshot(path: Path = DEFAULT_SNAPSHOT_PATH) -> Dict[str, Any]:
    import json

    if not path.exists():
        raise FileNotFoundError(f"Snapshot file not found: {path}")
    with path.open("r") as f:
        return json.load(f)


def extract_residential_summary(parsed: Dict[str, Any]) -> Dict[str, Any]:
    """Return only the residential fields we care about for versioning."""
    rs = parsed["residential_standard"]
    srs = parsed["residential_supplemental"]

    return {
        "residential_standard": {
            "customer_charge": rs.get("customer_charge_usd_per_month"),
            "energy_rate": rs.get("energy_rate_usd_per_kwh"),
            "tva_fuel_rate": rs.get("tva_fuel_rate_usd_per_kwh"),
        },
        "residential_supplemental": {
            "part_a": srs.get("customer_charge_part_a_usd_per_month"),
            "part_b": srs.get("customer_charge_part_b_usd_per_month"),
            "energy_rate": srs.get("energy_rate_usd_per_kwh"),
            "tva_fuel_rate": srs.get("tva_fuel_rate_usd_per_kwh"),
        },
    }


def diff_summaries(
    current: Dict[str, Any],
    baseline: Dict[str, Any],
) -> List[FieldChange]:
    """Compute a flat list of field changes between current and baseline summaries."""
    changes: List[FieldChange] = []

    for section in ("residential_standard", "residential_supplemental"):
        curr_section = current.get(section, {})
        base_section = baseline.get(section, {})

        all_fields = sorted(set(curr_section.keys()) | set(base_section.keys()))
        for field in all_fields:
            old = base_section.get(field)
            new = curr_section.get(field)
            if old != new:
                changes.append(FieldChange(section=section, field=field, old=old, new=new))

    return changes


def format_changes_markdown(changes: List[FieldChange]) -> str:
    """Format changes as a Markdown list suitable for a GitHub issue."""
    if not changes:
        return "No changes detected."

    lines: List[str] = []
    lines.append("The following rate changes were detected:\n")

    by_section: Dict[str, List[FieldChange]] = {}
    for ch in changes:
        by_section.setdefault(ch.section, []).append(ch)

    for section, section_changes in by_section.items():
        lines.append(f"### {section}")
        lines.append("")
        for ch in section_changes:
            lines.append(
                f"- **{ch.field}**: `{ch.old}` → `{ch.new}`"
            )
        lines.append("")

    return "\n".join(lines)


def format_changes_console(changes: List[FieldChange]) -> str:
    """Human-readable text for CLI output."""
    if not changes:
        return "No changes detected."

    lines: List[str] = []
    lines.append("Changes detected:\n")

    for ch in changes:
        lines.append(
            f"[{ch.section}] {ch.field}: {ch.old} -> {ch.new}"
        )

    return "\n".join(lines)
