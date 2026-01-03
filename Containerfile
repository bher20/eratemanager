###############################
# Stage 1: UI Builder (Node)
###############################
FROM docker.io/library/node:20-alpine AS ui-builder

WORKDIR /app/ui-react

# Copy entire folder (Buildah-safe)
COPY ui-react/ .

RUN npm install
RUN npm run build

###############################
# Stage 2: Go Builder
###############################
FROM docker.io/library/golang:1.24-alpine AS go-builder

# Install ca-certificates to ensure we have the latest CA bundle
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Add missing intermediate certificates (e.g., GoDaddy G2 for WHUD)
COPY certs/*.pem /usr/local/share/ca-certificates/
RUN cat /usr/local/share/ca-certificates/*.pem >> /etc/ssl/certs/ca-certificates.crt || true

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy the built React UI
# The Vite build outputs to ../internal/ui/static/react-app (relative to ui-react)
# In the builder stage this is /app/internal/ui/static/react-app
COPY --from=ui-builder /app/internal/ui/static/react-app ./internal/ui/static/react-app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/eratemanager ./cmd/eratemanager

###############################
# Stage 3: Runtime
###############################
FROM gcr.io/distroless/static

# Copy CA certificates including our custom intermediate certs
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /app/eratemanager /eratemanager
COPY --from=go-builder /app/internal /internal

EXPOSE 8000
CMD ["/eratemanager"]
