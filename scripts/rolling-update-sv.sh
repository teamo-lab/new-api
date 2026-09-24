#!/usr/bin/env bash
set -Eeuo pipefail

# Silicon Valley rolling update entrypoint.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
export ROLLING_UPDATE_ENV=${ROLLING_UPDATE_ENV:-"$SCRIPT_DIR/rolling-update.env"}
exec "$SCRIPT_DIR/rolling-update.sh" "$@"
