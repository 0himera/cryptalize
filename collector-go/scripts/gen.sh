#!/usr/bin/env bash
# collector-go/scripts/gen.sh
#
# Generates Go + gRPC stubs from all .proto files under proto/.
# Run from the repository root:   bash collector-go/scripts/gen.sh
#
# Requires:
#   protoc              (system package — e.g. apt install protobuf-compiler)
#   protoc-gen-go       (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)
#   protoc-gen-go-grpc  (go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PROTO_ROOT="${REPO_ROOT}/proto"
OUT_DIR="${REPO_ROOT}/collector-go/internal/proto"
GOPATH_BIN="$(go env GOPATH)/bin"

# Ensure plugins are on PATH.
export PATH="${PATH}:${GOPATH_BIN}"

echo "▶ Checking required tools..."
for tool in protoc protoc-gen-go protoc-gen-go-grpc; do
  if ! command -v "${tool}" &>/dev/null; then
    echo "  ✗ '${tool}' not found. See script header for install instructions."
    exit 1
  fi
done
echo "  ✓ All tools present."

# Create output directory if needed.
mkdir -p "${OUT_DIR}"

echo "▶ Running protoc..."
find "${PROTO_ROOT}" -name "*.proto" | while read -r proto_file; do
  echo "  Compiling: ${proto_file#"${REPO_ROOT}/"}"
  protoc \
    --proto_path="${PROTO_ROOT}" \
    --go_out="${OUT_DIR}" \
    --go_opt=paths=source_relative \
    --go-grpc_out="${OUT_DIR}" \
    --go-grpc_opt=paths=source_relative \
    "${proto_file#"${PROTO_ROOT}/"}"
done

echo "▶ Done. Generated files are in: ${OUT_DIR}"
