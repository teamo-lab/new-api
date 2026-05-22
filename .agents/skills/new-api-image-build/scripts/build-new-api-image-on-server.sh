#!/usr/bin/env bash
set -euo pipefail

host="root@43.153.35.31"
repo="/home/work/new-api"
version=""

usage() {
  cat <<'USAGE'
Usage:
  build-new-api-image-on-server.sh <version>

Options:
  -h, --help          Show this help.

Fixed behavior:
  SSH target: root@43.153.35.31
  Repository: /home/work/new-api
  Source: origin/dev
  Build: ./build.sh <version>
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ -z "$version" ]]; then
        version="$1"
        shift
      else
        echo "Unknown argument: $1" >&2
        usage >&2
        exit 2
      fi
      ;;
  esac
done

if [[ -z "$version" ]]; then
  echo "--version is required" >&2
  usage >&2
  exit 2
fi

ssh -o BatchMode=yes -o ConnectTimeout=10 "$host" 'bash -s' -- "$repo" "$version" <<'REMOTE_SCRIPT'
set -euo pipefail

repo="$1"
version="$2"

cd "$repo"

echo ">>> Host: $(hostname)"
echo ">>> Repository: $(pwd)"
echo ">>> Source: origin/dev"
echo ">>> Version: $version"

git fetch origin dev --tags

if [[ -n "$(git status --short)" ]]; then
  echo "Refusing to build with local changes:" >&2
  git status --short >&2
  exit 1
fi

git checkout dev
git pull --ff-only origin dev

./build.sh "$version"
REMOTE_SCRIPT
