FROM docker.io/library/golang:1.24.4-alpine AS builder

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/eratemanager ./cmd/eratemanager

FROM docker.io/library/alpine:3.20
WORKDIR /app
COPY --from=builder /out/eratemanager .
EXPOSE 8000
CMD ["./eratemanager"]
