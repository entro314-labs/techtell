# =============================================================================
# Builder stage
# =============================================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

ARG BUILD_VERSION=dev
ARG BUILD_DATE=unknown
ARG GIT_COMMIT=unknown

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w \
      -X main.version=${BUILD_VERSION} \
      -X main.date=${BUILD_DATE} \
      -X main.commit=${GIT_COMMIT} \
      -X main.builtBy=docker" \
    -o /usr/local/bin/techtell \
    ./cmd/techtell

# =============================================================================
# Runtime stage
# =============================================================================
FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /usr/local/bin/techtell /usr/local/bin/techtell

USER app
WORKDIR /workspace

LABEL org.opencontainers.image.title="techtell" \
      org.opencontainers.image.description="A TUI tool that scans your developer workspace and produces a tech DNA report" \
      org.opencontainers.image.source="https://github.com/entro314-labs/techtell" \
      org.opencontainers.image.vendor="entro314-labs"

ENTRYPOINT ["/usr/local/bin/techtell"]
CMD ["--version"]
