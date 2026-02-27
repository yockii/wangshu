FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o yoclaw ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/yoclaw /app/yoclaw

RUN chmod +x /app/yoclaw

VOLUME ["/root/.yoClaw"]

ENTRYPOINT ["/app/yoclaw"]
