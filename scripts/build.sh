#!/bin/bash
# Builds pile from this checkout: the app's packages, the generated contract
# (Go and TypeScript), the app's static build, and the server binary with the
# app inside it (bin/pile), and the other commands beside it in bin/. Needs go, buf, node with corepack, and
# protoc-gen-go and protoc-gen-connect-go on PATH.
set -euo pipefail
cd "$(dirname "$0")/.."
(cd app && corepack pnpm install --frozen-lockfile 2>/dev/null || corepack pnpm install)
buf generate
(cd app && corepack pnpm build)
go mod tidy
go vet ./...
go build -o bin/ ./cmd/...
echo "built bin/"
