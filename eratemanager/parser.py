import re
from typing import Any, Dict, Optional

import pdfplumber


def pdf_to_text(pdf_path: str) -> str:
    """
    Extract all text from a PDF into a single string.
    """
    all_text = []
    with pdfplumber.open(pdf_path) as pdf:
        for page in pdf.pages:
            txt = page.extract_text() or ""
            all_text.append(txt)
    return "\n".join(all_text)


def _clean_float(s: str) -> Optional[float]:
    if s is None:
        return None
    s = s.replace(",", "").strip()
    if s.startswith("."):
        s = "0" + s
    try:
        return float(s)
    except ValueError:
        return None


def _find_section(
    text: str,
    header_pattern: str,
    next_header_patterns: Optional[list] = None,
) -> Optional[str]:
    """
    Extract the text block starting at header_pattern
    and ending before the first of next_header_patterns.
    """
    header_regex = re.compile(header_pattern, re.IGNORECASE | re.DOTALL)
    m = header_regex.search(text)
    if not m:
        return None

    start = m.start()

    if not next_header_patterns:
        return text[start:]

    end_candidates = []
    for pat in next_header_patterns:
        r = re.compile(pat, re.IGNORECASE | re.DOTALL)
        m2 = r.search(text, pos=start + 1)
        if m2:
            end_candidates.append(m2.start())

    if not end_candidates:
        return text[start:]

    end = min(end_candidates)
    return text[start:end]


def parse_residential_standard(section: str) -> Dict[str, Any]:
    """
    Parse "RESIDENTIAL RATE – SCHEDULE RS" block.
    """
    if not section:
        return {"is_present": False, "raw_section": None}

    cust_match = re.search(
        r"Customer\s+Charge[:\s]*\$?([\d,]*\.?\d+)",
        section,
        re.IGNORECASE,
    )
    customer_charge = _clean_float(cust_match.group(1)) if cust_match else None

    energy_match = re.search(
        r"Energy\s+Charge[:\s]*\$?([\d.,]*\.?\d+)\$?",
        section,
        re.IGNORECASE,
    )
    energy_rate = _clean_float(energy_match.group(1)) if energy_match else None

    tva_match = re.search(
        r"TVA\s+Fuel\s+Charge[:\s]*\$?([\d.,]*\.?\d+)\$?",
        section,
        re.IGNORECASE,
    )
    tva_rate = _clean_float(tva_match.group(1)) if tva_match else None

    return {
        "is_present": True,
        "raw_section": section,
        "customer_charge_usd_per_month": customer_charge,
        "energy_rate_usd_per_kwh": energy_rate,
        "tva_fuel_rate_usd_per_kwh": tva_rate,
    }


def parse_residential_supplemental(section: str) -> Dict[str, Any]:
    """
    Parse "SUPPLEMENTAL RESIDENTIAL RATE – SCHEDULE SRS" block.
    """
    if not section:
        return {"is_present": False, "raw_section": None}

    part_a_match = re.search(
        r"Part\s*A[:\s]*\$?([\d,]*\.?\d+)",
        section,
        re.IGNORECASE,
    )
    part_b_match = re.search(
        r"Part\s*B[:\s]*\$?([\d,]*\.?\d+)",
        section,
        re.IGNORECASE,
    )

    part_a = _clean_float(part_a_match.group(1)) if part_a_match else None
    part_b = _clean_float(part_b_match.group(1)) if part_b_match else None

    energy_match = re.search(
        r"Energy\s+Charge[:\s]*\$?([\d.,]*\.?\d+)\$?",
        section,
        re.IGNORECASE,
    )
    energy_rate = _clean_float(energy_match.group(1)) if energy_match else None

    tva_match = re.search(
        r"TVA\s+Fuel\s+Charge[:\s]*\$?([\d.,]*\.?\d+)\$?",
        section,
        re.IGNORECASE,
    )
    tva_rate = _clean_float(tva_match.group(1)) if tva_match else None

    return {
        "is_present": True,
        "raw_section": section,
        "customer_charge_part_a_usd_per_month": part_a,
        "customer_charge_part_b_usd_per_month": part_b,
        "energy_rate_usd_per_kwh": energy_rate,
        "tva_fuel_rate_usd_per_kwh": tva_rate,
    }


def parse_residential_seasonal(section: Optional[str]) -> Dict[str, Any]:
    """
    Forward-compatible placeholder.
    Currently, the 2025 CEMC PDF does NOT define a seasonal residential schedule.
    """
    if not section:
        return {
            "is_present": False,
            "raw_section": None,
            "summer_rate_usd_per_kwh": None,
            "winter_rate_usd_per_kwh": None,
            "summer_months": None,
            "winter_months": None,
        }

    return {
        "is_present": True,
        "raw_section": section,
        "summer_rate_usd_per_kwh": None,
        "winter_rate_usd_per_kwh": None,
        "summer_months": None,
        "winter_months": None,
    }


def parse_residential_tou(section: Optional[str]) -> Dict[str, Any]:
    """
    Forward-compatible placeholder for a future Residential TOU schedule.
    """
    if not section:
        return {
            "is_present": False,
            "raw_section": None,
            "on_peak_rate_usd_per_kwh": None,
            "off_peak_rate_usd_per_kwh": None,
            "shoulder_rate_usd_per_kwh": None,
            "on_peak_hours": None,
            "off_peak_hours": None,
            "shoulder_hours": None,
        }

    return {
        "is_present": True,
        "raw_section": section,
        "on_peak_rate_usd_per_kwh": None,
        "off_peak_rate_usd_per_kwh": None,
        "shoulder_rate_usd_per_kwh": None,
        "on_peak_hours": None,
        "off_peak_hours": None,
        "shoulder_hours": None,
    }


def parse_pdf(pdf_path: str) -> Dict[str, Any]:
    """
    Top-level parser.
    """
    text = pdf_to_text(pdf_path)

    rs_section = _find_section(
        text,
        r"RESIDENTIAL\s+RATE\s+[-–]\s+SCHEDULE\s+RS",
        next_header_patterns=[
            r"SUPPLEMENTAL\s+RESIDENTIAL\s+RATE\s+[-–]\s+SCHEDULE\s+SRS",
            r"GENERAL\s+POWER\s+RATE",
        ],
    )

    srs_section = _find_section(
        text,
        r"SUPPLEMENTAL\s+RESIDENTIAL\s+RATE\s+[-–]\s+SCHEDULE\s+SRS",
        next_header_patterns=[
            r"GENERAL\s+POWER\s+RATE",
        ],
    )

    seasonal_section = _find_section(
        text,
        r"SEASONAL\s+RESIDENTIAL\s+RATE\s+[-–]\s+SCHEDULE",
        next_header_patterns=[
            r"GENERAL\s+POWER\s+RATE",
            r"RESIDENTIAL\s+TIME",
        ],
    )

    tou_section = _find_section(
        text,
        r"RESIDENTIAL\s+TIME\s+OF\s+USE\s+RATE\s+[-–]\s+SCHEDULE",
        next_header_patterns=[
            r"GENERAL\s+POWER\s+RATE",
        ],
    )

    residential_standard = parse_residential_standard(rs_section)
    residential_supplemental = parse_residential_supplemental(srs_section)
    residential_seasonal = parse_residential_seasonal(seasonal_section)
    residential_tou = parse_residential_tou(tou_section)

    return {
        "raw_text": text,
        "residential_standard": residential_standard,
        "residential_supplemental": residential_supplemental,
        "residential_seasonal": residential_seasonal,
        "residential_tou": residential_tou,
    }
