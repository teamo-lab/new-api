#!/usr/bin/env bash
set -euo pipefail

REGISTRY="uswccr.ccs.tencentyun.com/floatai/newapi"
LOCAL_TAG="newapi"

VERSION="${1:?Usage: ./build.sh <version>  e.g. ./build.sh v0.6.0}"

echo ">>> Version: $VERSION"
echo "$VERSION" > VERSION

echo ">>> Building Docker image..."
docker build -t "$LOCAL_TAG:latest" -t "$LOCAL_TAG:$VERSION" .

echo ">>> Tagging for registry..."
docker tag "$LOCAL_TAG:latest" "$REGISTRY:latest"
docker tag "$LOCAL_TAG:$VERSION" "$REGISTRY:$VERSION"

echo ">>> Pushing to registry..."
docker push "$REGISTRY:latest"
docker push "$REGISTRY:$VERSION"

echo ">>> Done: $REGISTRY:$VERSION"
