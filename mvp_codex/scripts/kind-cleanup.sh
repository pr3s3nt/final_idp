#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cluster_name="idp-codex-mvp"
registry_name="idp-codex-registry"
metadata_name="idp-codex-metadata"

for pid_file in "$repo_root/.runtime/kind/api.pid" "$repo_root/.runtime/kind/worker.pid" "$repo_root/.runtime/kind/port-forward.pid"; do
  if [[ -f "$pid_file" ]]; then
    pid=$(<"$pid_file")
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid"
      wait "$pid" 2>/dev/null || true
    fi
    rm -f "$pid_file"
  fi
done

if kind get clusters | grep -Fxq "$cluster_name"; then
  kind delete cluster --name "$cluster_name"
fi
for container in "$registry_name" "$metadata_name"; do
  if docker container inspect "$container" >/dev/null 2>&1; then
    docker rm -f "$container" >/dev/null
  fi
done

echo "removed task-owned kind cluster and local containers"
