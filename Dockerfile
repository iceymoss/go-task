## syntax=docker/dockerfile:1

# -------- build stage --------
FROM golang:1.24.4-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a static-ish binary (CGO disabled) for small runtime images.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN mkdir -p /out && CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/go-task-scheduler ./cmd/scheduler


# -------- runtime stage --------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

WORKDIR /app

# Non-root user
RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /out/go-task-scheduler /app/go-task-scheduler
COPY configs /app/configs

USER app

EXPOSE 8080

# NOTE: the app expects `.env` to exist (main.go calls godotenv.Load()).
# Provide it via bind-mount or bake it into a derived image.
CMD ["/app/go-task-scheduler"]

