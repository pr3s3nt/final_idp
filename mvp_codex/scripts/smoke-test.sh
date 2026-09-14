#!/usr/bin/env bash
set -euo pipefail

base_url=${1:-http://127.0.0.1:18080}
note="idp-smoke-$(date +%s)"
created=$(curl --fail --silent --show-error -H 'Content-Type: application/json' -d "{\"body\":\"${note}\"}" "$base_url/api/notes")
listed=$(curl --fail --silent --show-error "$base_url/api/notes")
jq -e --arg note "$note" 'any(.[]; .body == $note)' <<<"$listed" >/dev/null
printf 'created=%s\nverified_note=%s\n' "$created" "$note"
