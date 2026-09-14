#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
runtime_dir="$repo_root/.runtime/kind"
cluster_name="idp-codex-mvp"
registry_name="idp-codex-registry"
metadata_name="idp-codex-metadata"
registry_port="5002"
metadata_port="55434"
argo_version="v3.5.2"

mkdir -p "$runtime_dir"
chmod 700 "$repo_root/.runtime" "$runtime_dir"

if kind get clusters | grep -Fxq "$cluster_name"; then
  echo "Refusing to adopt existing kind cluster $cluster_name" >&2
  exit 1
fi
for container in "$registry_name" "$metadata_name"; do
  if docker container inspect "$container" >/dev/null 2>&1; then
    echo "Refusing to adopt existing container $container" >&2
    exit 1
  fi
done

metadata_password=$(openssl rand -hex 24)
idp_token=$(openssl rand -hex 24)
db_password=$(openssl rand -hex 24)

docker run -d --restart=always -p "127.0.0.1:${registry_port}:5000" --name "$registry_name" registry:2 >/dev/null
docker run -d --restart=always -p "127.0.0.1:${metadata_port}:5432" --name "$metadata_name" \
  -e POSTGRES_USER=idp -e POSTGRES_PASSWORD="$metadata_password" -e POSTGRES_DB=idp postgres:17-alpine >/dev/null

kind_config="$runtime_dir/kind-config.yaml"
cat >"$kind_config" <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
nodes:
- role: control-plane
EOF
kind create cluster --name "$cluster_name" --config "$kind_config" --wait 180s
docker network connect kind "$registry_name"

registry_dir="/etc/containerd/certs.d/localhost:${registry_port}"
for node in $(kind get nodes --name "$cluster_name"); do
  docker exec "$node" mkdir -p "$registry_dir"
  docker exec "$node" sh -c "printf '%s\n' 'server = \"http://localhost:${registry_port}\"' '[host.\"http://${registry_name}:5000\"]' '  capabilities = [\"pull\", \"resolve\"]' > '${registry_dir}/hosts.toml'"
done

kubeconfig_path="$runtime_dir/kubeconfig"
kind get kubeconfig --name "$cluster_name" >"$kubeconfig_path"
chmod 600 "$kubeconfig_path"
kube_context="kind-${cluster_name}"
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" create namespace idp-demo-dev
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" create namespace idp-system
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" create namespace argocd

registry_ip=$(docker inspect -f '{{(index .NetworkSettings.Networks "kind").IPAddress}}' "$registry_name")
cat >"$runtime_dir/registry-service.yaml" <<EOF
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: idp-system
spec:
  ports:
  - name: registry
    port: 5000
    targetPort: 5000
---
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: registry
  namespace: idp-system
  labels:
    kubernetes.io/service-name: registry
addressType: IPv4
ports:
- name: registry
  protocol: TCP
  port: 5000
endpoints:
- addresses: ["${registry_ip}"]
EOF
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" apply -f "$runtime_dir/registry-service.yaml"

kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" apply --server-side --force-conflicts \
  -n argocd -f "https://raw.githubusercontent.com/argoproj/argo-cd/${argo_version}/manifests/install.yaml"
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" wait -n argocd --for=condition=Available deployment --all --timeout=360s

cat >"$runtime_dir/argo-repository.yaml" <<'EOF'
apiVersion: v1
kind: Secret
metadata:
  name: idp-local-oci
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
stringData:
  name: idp-local-oci
  type: oci
  url: oci://registry.idp-system.svc.cluster.local:5000/idp/manifests
  insecure: "true"
  insecureOCIForceHttp: "true"
EOF
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" apply -f "$runtime_dir/argo-repository.yaml"
kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" create secret generic idp-notes-db-v1 \
  -n idp-demo-dev --from-literal=password="$db_password"
secret_uid=$(kubectl --kubeconfig "$kubeconfig_path" --context "$kube_context" get secret idp-notes-db-v1 -n idp-demo-dev -o jsonpath='{.metadata.uid}')

backend_repo="localhost:${registry_port}/idp-notes-backend"
frontend_repo="localhost:${registry_port}/idp-notes-frontend"
docker build -f "$repo_root/apps/backend/Dockerfile" -t "$backend_repo:local" "$repo_root"
docker build -f "$repo_root/apps/frontend/Dockerfile" -t "$frontend_repo:local" "$repo_root"
docker push "$backend_repo:local" >/dev/null
docker push "$frontend_repo:local" >/dev/null
backend_digest=$(docker inspect --format '{{index .RepoDigests 0}}' "$backend_repo:local" | sed 's/.*@//')
frontend_digest=$(docker inspect --format '{{index .RepoDigests 0}}' "$frontend_repo:local" | sed 's/.*@//')
docker pull postgres:17-alpine >/dev/null
postgres_image=$(docker inspect --format '{{index .RepoDigests 0}}' postgres:17-alpine)

jq \
  --arg target "$cluster_name" \
  --arg context "$kube_context" \
  --arg kubeconfig "$kubeconfig_path" \
  --arg secret_uid "$secret_uid" \
  --arg backend_repo "$backend_repo" \
  --arg frontend_repo "$frontend_repo" \
  --arg backend_digest "$backend_digest" \
  --arg frontend_digest "$frontend_digest" \
  --arg postgres_image "$postgres_image" '
  .snapshot.applicationDefinition.workloads |= map(
    if .name == "backend" then .imageRepository = $backend_repo
    elif .name == "frontend" then .imageRepository = $frontend_repo else . end
  ) |
  .snapshot.environmentConfiguration.secrets[0] = {
    id: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
    workloadId: "22222222-2222-4222-8222-222222222222",
    name: "DB_PASSWORD",
    source: "KUBERNETES_SECRET",
    target: $target,
    namespace: "idp-demo-dev",
    secretName: "idp-notes-db-v1",
    uid: $secret_uid,
    key: "password"
  } |
  .snapshot.renderContext = {
    targetId: $target,
    cloudProvider: "kind",
    region: "local",
    clusterIdentity: $target,
    namespace: "idp-demo-dev",
    namingPolicy: "mvp-v1",
    rendererVersion: "score-k8s-0.15.0",
    adapterVersions: {
      terraform: "1.9.8",
      kubeconfigPath: $kubeconfig,
      kubeContext: $context,
      argocd: "v3.5.2",
      argoKubeconfigPath: $kubeconfig,
      argoKubeContext: $context,
      argoDestinationServer: "https://kubernetes.default.svc"
    }
  } |
  .snapshot.images = [
    {workloadId: "22222222-2222-4222-8222-222222222222", tag: "local", digest: $backend_digest},
    {workloadId: "33333333-3333-4333-8333-333333333333", tag: "local", digest: $frontend_digest}
  ] |
  .resourceDefinitions |= map(if .name == "postgres-kubernetes-mvp" then .defaultParameters.postgresImage = $postgres_image else . end)
' "$repo_root/fixtures/uc3-demo.json" >"$runtime_dir/fixture.json"

cat >"$runtime_dir/env" <<EOF
export DATABASE_URL='postgres://idp:${metadata_password}@127.0.0.1:${metadata_port}/idp?sslmode=disable'
export IDP_TOKEN='${idp_token}'
export IDP_REPOSITORY_ROOT='${repo_root}'
export IDP_STATE_ROOT='.state/kind/resources'
export IDP_ARTIFACT_REPOSITORY='localhost:${registry_port}/idp/manifests'
export IDP_ARTIFACT_SOURCE_REPOSITORY='registry.idp-system.svc.cluster.local:5000/idp/manifests'
export IDP_ARTIFACT_PLAIN_HTTP='true'
export IDP_ARGO_NAMESPACE='argocd'
export IDP_WORKER_LOCK='${repo_root}/.runtime/kind/worker.lock'
export IDP_KIND_FIXTURE='${runtime_dir}/fixture.json'
export IDP_KIND_KUBECONFIG='${kubeconfig_path}'
export IDP_KIND_CONTEXT='${kube_context}'
EOF
chmod 600 "$runtime_dir/env"

echo "kind environment ready: source $runtime_dir/env"
