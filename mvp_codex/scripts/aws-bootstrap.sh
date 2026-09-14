#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
runtime_dir="$repo_root/.runtime/aws"
terraform_dir="$repo_root/terraform/bootstrap-aws"
bootstrap_state="$runtime_dir/bootstrap.tfstate"
kind_kubeconfig="$repo_root/.runtime/kind/kubeconfig"
kind_context="kind-idp-codex-mvp"
expected_account="${IDP_AWS_ACCOUNT_ID:-452025861381}"
aws_region="${AWS_REGION:-ap-southeast-1}"
external_secrets_version="2.10.0"

for command in aws curl docker go helm jq kubectl terraform; do
  command -v "$command" >/dev/null || { echo "missing required command: $command" >&2; exit 1; }
done

[[ -s "$kind_kubeconfig" ]] || { echo "run kind-bootstrap.sh first" >&2; exit 1; }
kubectl --kubeconfig "$kind_kubeconfig" --context "$kind_context" get namespace argocd >/dev/null

actual_account=$(aws sts get-caller-identity --query Account --output text)
[[ "$actual_account" == "$expected_account" ]] || {
  echo "AWS account mismatch: expected $expected_account, got $actual_account" >&2
  exit 1
}

public_ip=$(curl --fail --silent --show-error https://checkip.amazonaws.com | tr -d '[:space:]')
[[ "$public_ip" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] || { echo "could not resolve a valid public IPv4 address" >&2; exit 1; }

mkdir -p "$runtime_dir"
chmod 700 "$repo_root/.runtime" "$runtime_dir"

jq -n \
  --arg account "$expected_account" \
  --arg region "$aws_region" \
  --arg access_cidr "${public_ip}/32" \
  '{checkedAt:(now|todate),accountId:$account,region:$region,eksVersion:"1.35",node:{type:"t3.small",count:1,capacity:"ON_DEMAND"},network:{natGateway:false,eksPublicAccessCidrs:[$access_cidr],auroraPublic:false},aurora:{engine:"Aurora PostgreSQL Serverless v2",minACU:0.5,maxACU:1},argoCD:"kind control plane",registry:"ECR",estimatedBaseHourlyUSD:0.2314,estimateBasis:{eksCluster:0.1,t3Small:0.0264,auroraAtMinACU:0.1,publicIPv4:0.005},excludes:["small EBS/ECR/Aurora storage","I/O and data transfer","tax"]}' \
  >"$runtime_dir/preflight.json"

jq -n \
  --arg region "$aws_region" \
  --arg account "$expected_account" \
  --arg cidr "${public_ip}/32" \
  '{aws_region:$region,expected_account_id:$account,cluster_version:"1.35",node_instance_type:"t3.small",cluster_public_access_cidrs:[$cidr]}' \
  >"$runtime_dir/bootstrap.auto.tfvars.json"
chmod 600 "$runtime_dir/bootstrap.auto.tfvars.json"

terraform -chdir="$terraform_dir" init -backend=false -input=false
terraform -chdir="$terraform_dir" apply -input=false -auto-approve -lock=true \
  -state="$bootstrap_state" -var-file="$runtime_dir/bootstrap.auto.tfvars.json"
terraform -chdir="$terraform_dir" output -json -state="$bootstrap_state" \
  | jq 'with_entries(.value = .value.value)' >"$runtime_dir/bootstrap-inventory.json"
chmod 600 "$runtime_dir/bootstrap-inventory.json"

cluster_name=$(jq -r .cluster_name "$runtime_dir/bootstrap-inventory.json")
cluster_endpoint=$(jq -r .cluster_endpoint "$runtime_dir/bootstrap-inventory.json")
cluster_ca=$(jq -r .cluster_ca_data "$runtime_dir/bootstrap-inventory.json")
eso_role_arn=$(jq -r .external_secrets_role_arn "$runtime_dir/bootstrap-inventory.json")
backend_repo=$(jq -r '.ecr_repositories["idp-notes-backend"]' "$runtime_dir/bootstrap-inventory.json")
frontend_repo=$(jq -r '.ecr_repositories["idp-notes-frontend"]' "$runtime_dir/bootstrap-inventory.json")
manifest_repo=$(jq -r '.ecr_repositories["idp-manifests"]' "$runtime_dir/bootstrap-inventory.json")
db_subnet_group=$(jq -r .db_subnet_group_name "$runtime_dir/bootstrap-inventory.json")
aurora_security_group=$(jq -r .aurora_security_group_id "$runtime_dir/bootstrap-inventory.json")
ecr_registry="${expected_account}.dkr.ecr.${aws_region}.amazonaws.com"
eks_kubeconfig="$runtime_dir/kubeconfig"
eks_context="$cluster_name"

aws eks update-kubeconfig --name "$cluster_name" --region "$aws_region" \
  --kubeconfig "$eks_kubeconfig" --alias "$eks_context" >/dev/null
chmod 600 "$eks_kubeconfig"
kubectl --kubeconfig "$eks_kubeconfig" --context "$eks_context" wait --for=condition=Ready node --all --timeout=600s
kubectl --kubeconfig "$eks_kubeconfig" --context "$eks_context" create namespace idp-demo-dev --dry-run=client -o yaml \
  | kubectl --kubeconfig "$eks_kubeconfig" --context "$eks_context" apply -f -

helm repo add external-secrets https://charts.external-secrets.io --force-update >/dev/null
helm repo update external-secrets >/dev/null
helm upgrade --install external-secrets external-secrets/external-secrets \
  --version "$external_secrets_version" \
  --namespace external-secrets --create-namespace \
  --set installCRDs=true \
  --set-string "serviceAccount.annotations.eks\.amazonaws\.com/role-arn=$eso_role_arn" \
  --kubeconfig "$eks_kubeconfig" --kube-context "$eks_context" --wait --timeout 10m

cat <<EOF | kubectl --kubeconfig "$eks_kubeconfig" --context "$eks_context" apply -f -
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: ${aws_region}
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: idp-argocd-manager
  namespace: idp-demo-dev
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: idp-argocd-manager
  namespace: idp-demo-dev
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: idp-argocd-manager
  namespace: idp-demo-dev
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: idp-argocd-manager
subjects:
- kind: ServiceAccount
  name: idp-argocd-manager
  namespace: idp-demo-dev
---
apiVersion: v1
kind: Secret
metadata:
  name: idp-argocd-manager-token
  namespace: idp-demo-dev
  annotations:
    kubernetes.io/service-account.name: idp-argocd-manager
type: kubernetes.io/service-account-token
EOF

for _ in $(seq 1 60); do
  cluster_token=$(kubectl --kubeconfig "$eks_kubeconfig" --context "$eks_context" -n idp-demo-dev \
    get secret idp-argocd-manager-token -o jsonpath='{.data.token}' 2>/dev/null | base64 -d || true)
  [[ -n "$cluster_token" ]] && break
  sleep 2
done
[[ -n "${cluster_token:-}" ]] || { echo "EKS service-account token was not populated" >&2; exit 1; }

cluster_client_config=$(jq -cn --arg token "$cluster_token" --arg ca "$cluster_ca" \
  '{bearerToken:$token,tlsClientConfig:{insecure:false,caData:$ca}}')
jq -n \
  --arg name "$cluster_name" \
  --arg server "$cluster_endpoint" \
  --arg config "$cluster_client_config" \
  '{apiVersion:"v1",kind:"Secret",metadata:{name:"idp-aws-cluster",namespace:"argocd",labels:{"argocd.argoproj.io/secret-type":"cluster"}},type:"Opaque",stringData:{name:$name,server:$server,namespaces:"idp-demo-dev",clusterResources:"false",config:$config}}' \
  | kubectl --kubeconfig "$kind_kubeconfig" --context "$kind_context" apply -f - >/dev/null
unset cluster_token cluster_client_config

ecr_password=$(aws ecr get-login-password --region "$aws_region")
printf '%s' "$ecr_password" | docker login --username AWS --password-stdin "$ecr_registry" >/dev/null
jq -n \
  --arg url "oci://${manifest_repo}" \
  --arg password "$ecr_password" \
  '{apiVersion:"v1",kind:"Secret",metadata:{name:"idp-aws-oci",namespace:"argocd",labels:{"argocd.argoproj.io/secret-type":"repository"}},type:"Opaque",stringData:{name:"idp-aws-oci",type:"oci",url:$url,username:"AWS",password:$password}}' \
  | kubectl --kubeconfig "$kind_kubeconfig" --context "$kind_context" apply -f - >/dev/null
unset ecr_password

docker build -f "$repo_root/apps/backend/Dockerfile" -t "$backend_repo:mvp" "$repo_root"
docker build -f "$repo_root/apps/frontend/Dockerfile" -t "$frontend_repo:mvp" "$repo_root"
docker push "$backend_repo:mvp" >/dev/null
docker push "$frontend_repo:mvp" >/dev/null
backend_digest=$(aws ecr describe-images --region "$aws_region" --repository-name "${backend_repo#*/}" \
  --image-ids imageTag=mvp --query 'imageDetails[0].imageDigest' --output text)
frontend_digest=$(aws ecr describe-images --region "$aws_region" --repository-name "${frontend_repo#*/}" \
  --image-ids imageTag=mvp --query 'imageDetails[0].imageDigest' --output text)

jq \
  --arg target "$cluster_name" \
  --arg cluster_arn "$(jq -r .cluster_arn "$runtime_dir/bootstrap-inventory.json")" \
  --arg region "$aws_region" \
  --arg argo_kubeconfig "$kind_kubeconfig" \
  --arg argo_context "$kind_context" \
  --arg destination "$cluster_endpoint" \
  --arg db_subnet_group "$db_subnet_group" \
  --arg aurora_security_group "$aurora_security_group" \
  --arg backend_repo "$backend_repo" \
  --arg frontend_repo "$frontend_repo" \
  --arg backend_digest "$backend_digest" \
  --arg frontend_digest "$frontend_digest" '
  .snapshot.applicationDefinition.workloads |= map(
    if .name == "backend" then .imageRepository = $backend_repo
    elif .name == "frontend" then .imageRepository = $frontend_repo else . end
  ) |
  .snapshot.environmentConfiguration.secrets[0].target = $target |
  .snapshot.renderContext.targetId = $target |
  .snapshot.renderContext.cloudProvider = "aws" |
  .snapshot.renderContext.region = $region |
  .snapshot.renderContext.clusterIdentity = $cluster_arn |
  .snapshot.renderContext.adapterVersions = {
    terraform: "1.9.8",
    argocd: "v3.5.2",
    argoKubeconfigPath: $argo_kubeconfig,
    argoKubeContext: $argo_context,
    argoDestinationServer: $destination,
    externalSecretsApiVersion: "external-secrets.io/v1",
    clusterSecretStore: "aws-secrets-manager"
  } |
  .snapshot.renderContext.provisionerInputs = {
    dbSubnetGroupName: $db_subnet_group,
    vpcSecurityGroupIds: [$aurora_security_group]
  } |
  .snapshot.images = [
    {workloadId:"22222222-2222-4222-8222-222222222222",tag:"mvp",digest:$backend_digest},
    {workloadId:"33333333-3333-4333-8333-333333333333",tag:"mvp",digest:$frontend_digest}
  ]
' "$repo_root/fixtures/uc3-demo.json" >"$runtime_dir/fixture.json"

source "$repo_root/.runtime/kind/env"
cat >"$runtime_dir/env" <<EOF
export AWS_REGION='${aws_region}'
export AWS_DEFAULT_REGION='${aws_region}'
export IDP_AWS_ACCOUNT_ID='${expected_account}'
export DATABASE_URL='${DATABASE_URL}'
export IDP_TOKEN='${IDP_TOKEN}'
export IDP_LISTEN='127.0.0.1:18082'
export IDP_REPOSITORY_ROOT='${repo_root}'
export IDP_STATE_ROOT='.state/aws/resources'
export IDP_ARTIFACT_REPOSITORY='${manifest_repo}'
export IDP_ARTIFACT_SOURCE_REPOSITORY='${manifest_repo}'
export IDP_ARTIFACT_PLAIN_HTTP='false'
export IDP_ARGO_NAMESPACE='argocd'
export IDP_WORKER_LOCK='${runtime_dir}/worker.lock'
export IDP_AWS_FIXTURE='${runtime_dir}/fixture.json'
export IDP_AWS_KUBECONFIG='${eks_kubeconfig}'
export IDP_AWS_CONTEXT='${eks_context}'
export IDP_ARGO_KUBECONFIG='${kind_kubeconfig}'
export IDP_ARGO_CONTEXT='${kind_context}'
EOF
chmod 600 "$runtime_dir/env"

echo "AWS target ready: source $runtime_dir/env"
