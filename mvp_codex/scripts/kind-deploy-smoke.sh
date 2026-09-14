#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
runtime_dir="$repo_root/.runtime/kind"
source "$runtime_dir/env"
export IDP_LISTEN="127.0.0.1:18081"

stop_processes() {
  for pid_file in "$runtime_dir/port-forward.pid" "$runtime_dir/worker.pid" "$runtime_dir/api.pid"; do
    if [[ -f "$pid_file" ]]; then
      pid=$(<"$pid_file")
      if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
      fi
      rm -f "$pid_file"
    fi
  done
}
trap stop_processes EXIT

go test ./...
mkdir -p "$repo_root/bin"
go build -o "$repo_root/bin/idp-admin" ./cmd/idp-admin
go build -o "$repo_root/bin/idp-api" ./cmd/idp-api
go build -o "$repo_root/bin/idp-worker" ./cmd/idp-worker

"$repo_root/bin/idp-admin" -command seed -fixture "$IDP_KIND_FIXTURE" -database-url "$DATABASE_URL"
"$repo_root/bin/idp-admin" -command plan -fixture "$IDP_KIND_FIXTURE" -state-root "$IDP_STATE_ROOT" \
  >"$runtime_dir/plan.json"
jq -e '.plan.items | length == 1 and .[0].provisionerReference == "terraform://modules/postgres-kubernetes@v1" and .[0].action == "CREATE"' \
  "$runtime_dir/plan.json" >/dev/null

rm -f "$IDP_WORKER_LOCK"
"$repo_root/bin/idp-api" >"$runtime_dir/api.log" 2>&1 &
echo $! >"$runtime_dir/api.pid"
"$repo_root/bin/idp-worker" >"$runtime_dir/worker.log" 2>&1 &
echo $! >"$runtime_dir/worker.pid"

for _ in $(seq 1 60); do
  curl --fail --silent http://127.0.0.1:18081/healthz >/dev/null 2>&1 && break
  sleep 1
done
curl --fail --silent http://127.0.0.1:18081/healthz >/dev/null

jq '{applicationId:.snapshot.applicationDefinition.id,environment:.snapshot.environmentConfiguration.environment,context:.snapshot.renderContext,images:.snapshot.images}' \
  "$IDP_KIND_FIXTURE" >"$runtime_dir/create-request.json"
curl --fail --silent --show-error \
  -H "Authorization: Bearer ${IDP_TOKEN}" -H 'Content-Type: application/json' \
  --data-binary @"$runtime_dir/create-request.json" http://127.0.0.1:18081/deployments \
  >"$runtime_dir/prepare-response.json"
deployment_id=$(jq -er .deploymentId "$runtime_dir/prepare-response.json")
plan_fingerprint=$(jq -er .planFingerprint "$runtime_dir/prepare-response.json")
idempotency_key="kind-${deployment_id}"
jq -n --arg fingerprint "$plan_fingerprint" '{expectedPlanFingerprint:$fingerprint,overrides:{}}' \
  >"$runtime_dir/confirm-request.json"
for attempt in 1 2; do
  curl --fail --silent --show-error \
    -H "Authorization: Bearer ${IDP_TOKEN}" -H 'Content-Type: application/json' \
    -H "Idempotency-Key: ${idempotency_key}" \
    --data-binary @"$runtime_dir/confirm-request.json" \
    "http://127.0.0.1:18081/deployments/${deployment_id}/confirm" \
    >"$runtime_dir/confirm-response-${attempt}.json"
done
jq -e --slurpfile second "$runtime_dir/confirm-response-2.json" '.trackingId == $second[0].trackingId' \
  "$runtime_dir/confirm-response-1.json" >/dev/null

for _ in $(seq 1 180); do
  curl --fail --silent --show-error -H "Authorization: Bearer ${IDP_TOKEN}" \
    "http://127.0.0.1:18081/deployments/${deployment_id}" >"$runtime_dir/deployment.json"
  execution_status=$(jq -r '.execution.status // "PENDING"' "$runtime_dir/deployment.json")
  [[ "$execution_status" == "SUCCEEDED" ]] && break
  if [[ "$execution_status" == "FAILED" ]]; then
    jq . "$runtime_dir/deployment.json" >&2
    exit 1
  fi
  sleep 2
done
jq -e '.execution.status == "SUCCEEDED" and (.steps | length == 3) and all(.steps[]; .status == "SUCCEEDED")' \
  "$runtime_dir/deployment.json" >/dev/null

for _ in $(seq 1 120); do
  sync_status=$(kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n argocd \
    get application idp-idp-notes-dev -o jsonpath='{.status.sync.status}' 2>/dev/null || true)
  health_status=$(kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n argocd \
    get application idp-idp-notes-dev -o jsonpath='{.status.health.status}' 2>/dev/null || true)
  [[ "$sync_status" == "Synced" && "$health_status" == "Healthy" ]] && break
  sleep 2
done
[[ "${sync_status:-}" == "Synced" && "${health_status:-}" == "Healthy" ]]

kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n idp-demo-dev \
  wait --for=condition=Available deployment/idp-notes-backend deployment/idp-notes-frontend --timeout=300s
kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n idp-demo-dev \
  port-forward --address=127.0.0.1 service/idp-notes-frontend 28080:8080 \
  >"$runtime_dir/port-forward.log" 2>&1 &
echo $! >"$runtime_dir/port-forward.pid"
for _ in $(seq 1 60); do
  curl --fail --silent http://127.0.0.1:28080/healthz >/dev/null 2>&1 && break
  sleep 1
done
"$repo_root/scripts/smoke-test.sh" http://127.0.0.1:28080 | tee "$runtime_dir/smoke.txt"

kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n argocd \
  get application idp-idp-notes-dev -o json >"$runtime_dir/argocd-application.json"
kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n idp-demo-dev \
  get deployment,statefulset,pod,service,pvc -o json >"$runtime_dir/workloads.json"
note_count=$(kubectl --kubeconfig "$IDP_KIND_KUBECONFIG" --context "$IDP_KIND_CONTEXT" -n idp-demo-dev \
  exec statefulset/idp-notes-postgres -- psql -U notes -d notes -Atc 'select count(*) from note;')
jq -n \
  --slurpfile deployment "$runtime_dir/deployment.json" \
  --slurpfile application "$runtime_dir/argocd-application.json" \
  --slurpfile workloads "$runtime_dir/workloads.json" \
  --arg note_count "$note_count" \
  --rawfile smoke "$runtime_dir/smoke.txt" \
  '{capturedAt:(now|todate),target:"kind-idp-codex-mvp",deployment:$deployment[0],argocd:{name:$application[0].metadata.name,sync:$application[0].status.sync.status,health:$application[0].status.health.status,revision:$application[0].status.sync.revision},workloads:$workloads[0],databaseNoteCount:($note_count|tonumber),smoke:$smoke}' \
  >"$runtime_dir/evidence.json"
chmod 600 "$runtime_dir/evidence.json"

echo "kind acceptance passed; evidence saved to $runtime_dir/evidence.json"
