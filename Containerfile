###############################
# Stage 1: UI Builder (Node)
###############################
FROM docker.io/library/node:20-alpine AS ui-builder

WORKDIR /app/ui-svelte-vite

# Copy entire folder (Buildah-safe)
COPY ui-svelte-vite/ .

RUN npm install
RUN npm run build


###############################
# Stage 2: Go Builder
###############################
# IMPORTANT: fully qualify all image names
FROM docker.io/library/golang:1.24-alpine AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy the built Svelte UI
COPY --from=ui-builder /app/ui-svelte-vite/../internal/ui/static/svelte-dist \
  ./internal/ui/static/svelte-dist

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/eratemanager ./cmd/eratemanager


###############################
# Stage 3: Runtime
###############################
FROM gcr.io/distroless/static

# Copy CA certificates for TLS verification
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /app/eratemanager /eratemanager
COPY --from=go-builder /app/internal /internal

EXPOSE 8000
CMD ["/eratemanager"]
