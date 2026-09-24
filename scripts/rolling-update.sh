#!/usr/bin/env bash
set -Eeuo pipefail

# Rolling update for the two Silicon Valley new-api nodes behind Tencent CLB.
# Usage:
#   ./scripts/rolling-update.sh [image]
#
# Put deployment-specific values in scripts/rolling-update.env (see the example
# file). The script intentionally refuses to run until CLB and node settings are
# complete, and it supports DRY_RUN=1 for validating the sequence first.

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ENV_FILE=${ROLLING_UPDATE_ENV:-"$SCRIPT_DIR/rolling-update.env"}

if [[ -f "$ENV_FILE" ]]; then
  # shellcheck disable=SC1090
  source "$ENV_FILE"
fi

IMAGE=${1:-${IMAGE:-uswccr.ccs.tencentyun.com/floatai/newapi:latest}}
CLB_REGION=${CLB_REGION:-na-siliconvalley}
CLB_ID=${CLB_ID:-}
CLB_LISTENER_ID=${CLB_LISTENER_ID:-}
CLB_LOCATION_ID=${CLB_LOCATION_ID:-}
CLB_TARGET_PORT=${CLB_TARGET_PORT:-3000}
DRAIN_SECONDS=${DRAIN_SECONDS:-15}
HEALTH_TIMEOUT_SECONDS=${HEALTH_TIMEOUT_SECONDS:-180}
CLB_TASK_TIMEOUT_SECONDS=${CLB_TASK_TIMEOUT_SECONDS:-120}
DRY_RUN=${DRY_RUN:-0}

# Each entry is SSH_TARGET,ENI_IP. The ENI_IP is the private IP registered in CLB.
NODES_CSV=${NODES_CSV:-"root@43.153.32.67,172.26.0.31 root@43.172.117.230,172.26.0.32"}

die() { echo "ERROR: $*" >&2; exit 1; }
log() { printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"; }

[[ -n "$CLB_ID" ]] || die "CLB_ID is required (set it in $ENV_FILE)"
[[ -n "$CLB_LISTENER_ID" ]] || die "CLB_LISTENER_ID is required (set it in $ENV_FILE)"
command -v tccli >/dev/null || die "tccli is required for CLB operations"
command -v ssh >/dev/null || die "ssh is required"

read -r -a NODE_ENTRIES <<< "$NODES_CSV"
[[ ${#NODE_ENTRIES[@]} -gt 0 ]] || die "NODES_CSV must contain at least one node"

run() {
  if (( DRY_RUN )); then
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

target_json() {
  local eni_ip=$1
  if [[ -n "$CLB_LOCATION_ID" ]]; then
    printf '[{"EniIp":"%s","Port":%s,"LocationId":"%s"}]' \
      "$eni_ip" "$CLB_TARGET_PORT" "$CLB_LOCATION_ID"
  else
    printf '[{"EniIp":"%s","Port":%s}]' "$eni_ip" "$CLB_TARGET_PORT"
  fi
}

clb_target() {
  local action=$1 eni_ip=$2 json response request_id status deadline
  json=$(target_json "$eni_ip")
  log "CLB $action $eni_ip:$CLB_TARGET_PORT"
  if (( DRY_RUN )); then
    run tccli clb "$action" \
      --region "$CLB_REGION" \
      --LoadBalancerId "$CLB_ID" \
      --ListenerId "$CLB_LISTENER_ID" \
      --Targets "$json"
    return 0
  fi

  if ! response=$(tccli clb "$action" \
    --region "$CLB_REGION" \
    --LoadBalancerId "$CLB_ID" \
    --ListenerId "$CLB_LISTENER_ID" \
    --Targets "$json" 2>&1); then
    printf '%s\n' "$response" >&2
    return 1
  fi

  request_id=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["Response"]["RequestId"])' <<<"$response") || {
    printf 'Unable to read CLB RequestId from response: %s\n' "$response" >&2
    return 1
  }
  deadline=$((SECONDS + CLB_TASK_TIMEOUT_SECONDS))
  while (( SECONDS < deadline )); do
    status=$(tccli clb DescribeTaskStatus \
      --region "$CLB_REGION" \
      --TaskId "$request_id" \
      | python3 -c 'import json,sys; print(json.load(sys.stdin)["Response"].get("Status", ""))')
    case "$status" in
      0) return 0 ;;
      1) echo "CLB task failed: $request_id" >&2; return 1 ;;
      2) sleep 2 ;;
      *) echo "Unknown CLB task status '$status' for $request_id" >&2; return 1 ;;
    esac
  done
  echo "Timed out waiting for CLB task: $request_id" >&2
  return 1
}

wait_remote_health() {
  local ssh_target=$1 deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS)) status
  log "Waiting for $ssh_target to become healthy"
  while (( SECONDS < deadline )); do
    if (( DRY_RUN )); then
      log "DRY_RUN: would check http://127.0.0.1:3000/api/status and Docker health"
      return 0
    fi
    if status=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$ssh_target" \
      'set -e; curl -fsS --max-time 5 http://127.0.0.1:3000/api/status >/dev/null; docker inspect -f "{{if .State.Health}}{{.State.Health.Status}}{{else}}running{{end}}" new-api' 2>/dev/null) \
      && [[ "$status" == healthy || "$status" == running ]]; then
      return 0
    fi
    sleep 5
  done
  return 1
}

update_node() {
  local ssh_target=$1
  log "Updating $ssh_target with $IMAGE"
  run ssh -o BatchMode=yes -o ConnectTimeout=10 "$ssh_target" \
    "set -e
     docker pull '$IMAGE'
     docker tag '$IMAGE' calciumion/new-api:latest
     cd /home/work/new-api
     docker-compose up -d --no-build --force-recreate new-api"
}

for entry in "${NODE_ENTRIES[@]}"; do
  IFS=',' read -r ssh_target eni_ip <<< "$entry"
  [[ -n "$ssh_target" && -n "$eni_ip" ]] || die "Invalid node entry: $entry"

  log "Starting rolling update for $ssh_target ($eni_ip)"
  clb_target DeregisterTargets "$eni_ip"
  log "Waiting ${DRAIN_SECONDS}s for existing connections to drain"
  run sleep "$DRAIN_SECONDS"

  if ! update_node "$ssh_target" || ! wait_remote_health "$ssh_target"; then
    log "Update failed for $ssh_target; attempting to register it back into CLB"
    clb_target RegisterTargets "$eni_ip" || true
    die "Rolling update stopped at $ssh_target"
  fi

  clb_target RegisterTargets "$eni_ip"
  log "$ssh_target is updated and registered back into CLB"
done

log "Rolling update completed successfully: $IMAGE"
