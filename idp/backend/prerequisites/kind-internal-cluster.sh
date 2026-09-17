#!/usr/bin/env bash
# Platform-owned internal Kubernetes cluster for target kind-local. It exists
# before any deployment and the IDP never creates or destroys it: the catalog
# definition kind-internal-cluster (EXISTING) points to its connection record
# idpsecret://platform/kind-internal-cluster, which this script writes.
#   ./kind-internal-cluster.sh           # create or update the cluster and register it
#   ./kind-internal-cluster.sh destroy   # delete the cluster
set -euo pipefail
cd "$(dirname "$0")/.."
WORK=var/platform/kind-internal-cluster
mkdir -p "$WORK" var/terraform/plugin-cache && chmod 700 var/platform
cp prerequisites/kind-internal-cluster/main.tf "$WORK/main.tf"
export TF_PLUGIN_CACHE_DIR=$PWD/var/terraform/plugin-cache
terraform -chdir="$WORK" init -input=false -no-color >/dev/null
if [ "${1:-}" = destroy ]; then
  terraform -chdir="$WORK" destroy -auto-approve -input=false -no-color
  exit 0
fi
terraform -chdir="$WORK" apply -auto-approve -input=false -no-color
RECORD=$(mktemp)
trap 'rm -f "$RECORD"' EXIT
chmod 600 "$RECORD"
terraform -chdir="$WORK" output -json |
  jq '{cluster_name: .cluster_name.value, cluster_kind: .cluster_kind.value,
       image_registry_mirror: .image_registry_mirror.value, kubeconfig: .kubeconfig.value}' > "$RECORD"
./bin/idp secret-put platform/kind-internal-cluster "$RECORD"
echo "internal cluster $(jq -r .cluster_name "$RECORD") registered"
