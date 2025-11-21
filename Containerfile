# Containerfile for eratemanager FastAPI service
# Build with:
#   buildah bud -t ghcr.io/bher20/eratemanager:0.1.0 .

FROM python:3.11-slim

# Prevent Python from writing .pyc files & buffering stdout/stderr
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# Install system dependencies (if needed by pdfplumber or others)
RUN apt-get update && apt-get install -y --no-install-recommends \
      build-essential \
      libffi-dev \
      libjpeg62-turbo-dev \
      libpng-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy only metadata first to leverage Docker/Buildah layer caching
COPY pyproject.toml setup.cfg README.md ./

# Copy the package and supporting files
COPY eratemanager ./eratemanager
COPY snapshots ./snapshots

# Optional: if you want tests/tools in the image, uncomment:
# COPY tests ./tests
# COPY tools ./tools

# Install in "normal" mode (not editable) into the image
RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir .

# Expose FastAPI port
EXPOSE 8000

# Default command: run uvicorn
CMD ["uvicorn", "eratemanager.api:app", "--host", "0.0.0.0", "--port", "8000"]
