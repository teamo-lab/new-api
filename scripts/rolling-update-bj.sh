#!/usr/bin/env bash
set -Eeuo pipefail

# Beijing rolling update entrypoint. Keep the common implementation in one
# place so CLB drain, health checks, rollback registration, and update order do
# not diverge between regions.
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
export ROLLING_UPDATE_ENV=${ROLLING_UPDATE_ENV:-"$SCRIPT_DIR/rolling-update-bj.env"}
exec "$SCRIPT_DIR/rolling-update.sh" "$@"
