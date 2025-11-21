from fastapi import FastAPI, HTTPException

from .downloader import download_pdf
from .normalizer import normalize
from .parser import parse_pdf

app = FastAPI(title="CEMC Rates API", version="0.1.0")

@app.get("/health")
async def health():
    return {"status": "ok"}


@app.get("/rates/cemc/residential")
async def get_cemc_residential(force: bool = False):
    """
    Returns normalized residential rates for CEMC.

    Query params:
      - force: if true, forces a re-download of the PDF.
    """
    try:
        pdf_path = download_pdf(force=force)
        parsed = parse_pdf(str(pdf_path))
        normalized = normalize(parsed)
        return normalized
    except Exception as exc:  # pragma: no cover - defensive
        raise HTTPException(status_code=500, detail=str(exc))

@app.get("/rates/cemc/residential")
async def get_cemc_residential(force: bool = False):
    """
    Returns normalized residential rates for CEMC.

    Query params:
      - force: if true, forces a re-download of the PDF.
    """
    try:
        pdf_path = download_pdf(force=force)
        parsed = parse_pdf(str(pdf_path))
        normalized = normalize(parsed)
        return normalized
    except Exception as exc:  # pragma: no cover - defensive
        raise HTTPException(status_code=500, detail=str(exc))
