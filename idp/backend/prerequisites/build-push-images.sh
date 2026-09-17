#!/usr/bin/env bash
# CI stand-in: builds the demo workloads and pushes tags v1, v2 and broken to a
# registry. The registry is the one a target cluster pulls from:
#   ./build-push-images.sh localhost:5055
#   ./build-push-images.sh <account>.dkr.ecr.ap-southeast-1.amazonaws.com/idp-uc03-demo
set -euo pipefail
REGISTRY=${1:?registry}
ROOT=$(cd "$(dirname "$0")/../../../demo-apps" && pwd)
OUT=$(mktemp -d)
trap 'rm -rf "$OUT"' EXIT

cd "$ROOT"
go mod download
for app in backend frontend worker; do
  for tag in v1 v2 broken; do
    version=$tag broken=false
    [ "$tag" = broken ] && version=v2-broken broken=true
    mkdir -p "$OUT/$app-$tag"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
      -ldflags "-s -w -X github.com/pr3s3nt/final_idp/demo-apps/internal/common.Version=$version -X github.com/pr3s3nt/final_idp/demo-apps/internal/common.Broken=$broken" \
      -o "$OUT/$app-$tag/app" "./cmd/$app"
    cat > "$OUT/$app-$tag/Dockerfile" <<'EOF'
FROM alpine:latest
COPY app /app
USER 65532
ENTRYPOINT ["/app"]
EOF
    image="$REGISTRY/shop-$app:$tag"
    docker build -q -t "$image" "$OUT/$app-$tag" >/dev/null
    docker push -q "$image" >/dev/null
    echo "pushed $image"
  done
done
