# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X github.com/sirosfoundation/mtcvctm/cmd/mtcvctm/cmd.Version=${VERSION} -X github.com/sirosfoundation/mtcvctm/cmd/mtcvctm/cmd.Commit=${COMMIT}" \
    -o mtcvctm ./cmd/mtcvctm

# Final stage - minimal image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates git

WORKDIR /workspace

# Copy binary from builder
COPY --from=builder /build/mtcvctm /usr/local/bin/mtcvctm

# Set up non-root user
RUN adduser -D -u 1000 mtcvctm
USER mtcvctm

ENTRYPOINT ["mtcvctm"]
CMD ["--help"]
