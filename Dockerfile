FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o wangshu ./cmd

FROM python:3.12-alpine

RUN apk --no-cache add ca-certificates tzdata git nodejs npm

WORKDIR /app

COPY --from=builder /build/wangshu /app/wangshu

RUN chmod +x /app/wangshu

VOLUME ["/root/.wangshu"]

ENTRYPOINT ["/app/wangshu"]
