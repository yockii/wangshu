#!/bin/sh
set -e

ARCH=${2:-loong64}

export CGO_ENABLED=1
export CGO_CFLAGS="-w"
export GOARCH=loong64
export GOOS=linux

if [ -d "frontend" ] && [ -f "frontend/package.json" ] && [ ! -d "frontend/dist" ]; then
    (cd frontend && rm -rf node_modules package-lock.json && npm install --silent --force && npm run build --silent)
fi

APP=${APP_NAME:-$(basename $(pwd))}
mkdir -p bin

TAGS="production"
if [ -n "$EXTRA_TAGS" ]; then
    TAGS="${TAGS},${EXTRA_TAGS}"
fi

LDFLAGS="-s -w"
if [ -n "$VERSION" ]; then
    LDFLAGS="${LDFLAGS} -X github.com/yockii/wangshu/pkg/constant.Version=${VERSION}"
fi

go build -tags "$TAGS" -trimpath -buildvcs=false -ldflags="$LDFLAGS" -o bin/${APP}-linux-${GOARCH} .
echo "Built: bin/${APP}-linux-${GOARCH}"
