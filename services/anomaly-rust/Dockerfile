# --- Build stage ---
FROM rust:1-bookworm AS builder

WORKDIR /app

# Cache dependencies separately from source for faster rebuilds
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release && rm -rf src

COPY . .
RUN touch src/main.rs && cargo build --release

# --- Runtime stage ---
FROM gcr.io/distroless/cc-debian12:nonroot

WORKDIR /app
COPY --from=builder /app/target/release/anomaly-engine .

USER nonroot:nonroot

EXPOSE 9200

ENTRYPOINT ["/app/anomaly-engine"]