# --- Build stage ---
FROM golang:1.23.4-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/ingestor \
    ./cmd/ingestor

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/ingestor .

USER nonroot:nonroot

EXPOSE 9100

ENTRYPOINT ["/app/ingestor"]