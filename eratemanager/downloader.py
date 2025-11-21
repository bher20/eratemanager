import requests
from datetime import datetime
from pathlib import Path

PDF_URL = "https://cemc.org/wp-content/uploads/2025/11/RATES-2025-FCA202-202511.pdf"
CACHE_DIR = Path(".cache_cemc")
CACHE_DIR.mkdir(exist_ok=True)


def download_pdf(force: bool = False) -> Path:
    """
    Download the CEMC rates PDF (cached).

    Returns:
        Path to the local PDF file.
    """
    local_name = CACHE_DIR / Path(PDF_URL).name

    if local_name.exists() and not force:
        return local_name

    resp = requests.get(PDF_URL, timeout=30)
    resp.raise_for_status()

    local_name.write_bytes(resp.content)

    meta_path = local_name.with_suffix(".meta")
    meta_path.write_text(
        f"fetched: {datetime.utcnow().isoformat()}Z\n"
        f"source: {PDF_URL}\n"
    )

    return local_name
