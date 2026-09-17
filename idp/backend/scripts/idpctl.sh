#!/usr/bin/env bash
# Small client for the IDP JSON API, used by the demo and the e2e runs.
#   idpctl.sh deploy <app> <version> <env> <target> <workload=tag,...> [overrides-json]
#                                         # catalog version: IDP_CATALOG_VERSION (empty = form default)
#   idpctl.sh teardown <app> <env> <target>
#   idpctl.sh confirm <deployment-id> [overrides-json]
#   idpctl.sh plan <deployment-id>        # waves and removals of a pending plan
#   idpctl.sh status <deployment-id>      # prints the status only
#   idpctl.sh wait <deployment-id>        # waits for SUCCEEDED or FAILED
#   idpctl.sh show <deployment-id>        # images, steps, removals, errors, live status
set -euo pipefail
B=${IDP_URL:-http://127.0.0.1:8088}
J='content-type: application/json'

plan_summary='if .problems then {problems} else {id: .deployment.ID, status: .deployment.Status, catalog: .plan.catalogVersion, fingerprint: .fingerprint,
  waves: [.plan.waves[] | "wave \(.number): " + ([.items[] | "\(.action) \(.name)" + (if .definitionName then " [\(.definitionName)]" else " \(.imageVersion)" end)] | join(", "))],
  removals: [.plan.removals[] | "removal \(.number): " + ([.items[] | "\(.action) \(.name)" + (if .dataLossWarning then " (DATA LOSS)" else "" end)] | join(", "))],
  mayRedeploy: [.plan.potentialRedeploy[].name]} end'

case "${1:-}" in
deploy)
  app=$2 version=$3 env=$4 target=$5 pairs=$6 overrides=${7:-'{}'}
  workloads=$(tr ',' '\n' <<<"$pairs" | cut -d= -f1 | jq -R . | jq -sc .)
  images=$(tr ',' '\n' <<<"$pairs" | jq -R 'split("=") | {(.[0]): .[1]}' | jq -sc add)
  body=$(jq -nc --arg a "$app" --arg v "$version" --arg c "${IDP_CATALOG_VERSION:-}" --arg e "$env" --arg t "$target" --argjson w "$workloads" --argjson i "$images" \
    '{application:$a, version:$v, catalogVersion:$c, environment:$e, target:$t, workloads:$w, images:$i}')
  resp=$(curl -s -X POST "$B/api/deployments" -H "$J" -d "$body")
  jq -c "$plan_summary" <<<"$resp"
  id=$(jq -r '.deployment.ID // empty' <<<"$resp")
  [ -n "$id" ] || exit 1
  curl -s -X POST "$B/api/deployments/$id/confirm" -H "$J" -d "{\"overrides\": $overrides}" | jq -c .
  ;;
teardown)
  resp=$(curl -s -X POST "$B/api/teardowns" -H "$J" -d "$(jq -nc --arg a "$2" --arg e "$3" --arg t "$4" '{application:$a, environment:$e, target:$t}')")
  jq -c "$plan_summary" <<<"$resp"
  ;;
confirm)
  curl -s -X POST "$B/api/deployments/$2/confirm" -H "$J" -d "{\"overrides\": ${3:-{\}}}" | jq -c .
  ;;
plan)
  curl -s "$B/api/deployments/$2?live=false" | jq -c '.planView as $p | {status: .deployment.Status, planError} + (if $p then ($p | '"$plan_summary"') else {} end)'
  ;;
status)
  curl -s "$B/api/deployments/$2?live=false" | jq -r .deployment.Status
  ;;
wait)
  while true; do
    s=$(curl -s "$B/api/deployments/$2?live=false" | jq -r .deployment.Status)
    case "$s" in SUCCEEDED|FAILED) echo "$s"; break ;; esac
    sleep 5
  done
  ;;
show)
  curl -s "$B/api/deployments/$2" | jq '{kind: .deployment.Kind, status: .deployment.Status, version: .versionNumber, catalog: .deployment.CatalogVersionNumber,
    env: .deployment.Environment, target: .deployment.Target, job: .jobStatus, error: .record.ErrorSummary,
    images: [.images[] | "wave \(.wave) \(.workload) \(.image) \(.inclusionReason)\(if .runningHere then " (running)" else "" end)"],
    removed: .removed, infrastructure: [.infrastructure[] | "\(.component) \(.definition) \(.mode) \(.status)"],
    steps: [.steps[] | "\(.Wave) \(.Component) \(.Name) \(.Status)\(if .ErrorSummary != "" then ": " + .ErrorSummary else "" end)"],
    live: .live}'
  ;;
*)
  sed -n '2,10p' "$0"
  exit 2
  ;;
esac
