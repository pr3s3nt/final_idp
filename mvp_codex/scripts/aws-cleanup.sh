#!/usr/bin/env bash
set -uo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
runtime_dir="$repo_root/.runtime/aws"
terraform_dir="$repo_root/terraform/bootstrap-aws"
bootstrap_state="$runtime_dir/bootstrap.tfstate"
kind_kubeconfig="$repo_root/.runtime/kind/kubeconfig"
kind_context="kind-idp-codex-mvp"
cleanup_failed=0
aws_target_exists=false

if [[ -f "$runtime_dir/env" ]]; then
  source "$runtime_dir/env"
fi

if [[ -s "$runtime_dir/bootstrap-inventory.json" ]]; then
  inventory_cluster=$(jq -r .cluster_name "$runtime_dir/bootstrap-inventory.json")
  inventory_region=$(jq -r .aws_region "$runtime_dir/bootstrap-inventory.json")
  if aws eks describe-cluster --region "$inventory_region" --name "$inventory_cluster" >/dev/null 2>&1; then
    aws_target_exists=true
  fi
fi

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

if kind get clusters 2>/dev/null | grep -Fxq "${kind_context#kind-}" && [[ -n "${IDP_ARGO_KUBECONFIG:-}" && -s "${IDP_ARGO_KUBECONFIG:-}" ]]; then
  kubectl --kubeconfig "$IDP_ARGO_KUBECONFIG" --context "$IDP_ARGO_CONTEXT" -n argocd \
    delete application idp-idp-notes-dev --ignore-not-found --wait=true || cleanup_failed=1
fi
if [[ "$aws_target_exists" == "true" && -n "${IDP_AWS_KUBECONFIG:-}" && -s "${IDP_AWS_KUBECONFIG:-}" ]]; then
  kubectl --kubeconfig "$IDP_AWS_KUBECONFIG" --context "$IDP_AWS_CONTEXT" \
    delete namespace idp-demo-dev --ignore-not-found --wait=true --timeout=300s || cleanup_failed=1
fi

if [[ -d "$repo_root/.state/aws/resources" ]]; then
  while IFS= read -r -d '' state_file; do
    var_file="$(dirname "$state_file")/inputs.auto.tfvars.json"
    terraform -chdir="$repo_root/terraform/modules/postgres-aurora" init -backend=false -input=false >/dev/null || cleanup_failed=1
    terraform -chdir="$repo_root/terraform/modules/postgres-aurora" destroy -input=false -auto-approve -lock=true \
      -state="$state_file" -var-file="$var_file" || cleanup_failed=1
  done < <(find "$repo_root/.state/aws/resources" -type f -name terraform.tfstate -print0)
fi

if [[ -s "$bootstrap_state" && -s "$runtime_dir/bootstrap.auto.tfvars.json" ]]; then
  terraform -chdir="$terraform_dir" init -backend=false -input=false >/dev/null || cleanup_failed=1
  terraform -chdir="$terraform_dir" destroy -input=false -auto-approve -lock=true \
    -state="$bootstrap_state" -var-file="$runtime_dir/bootstrap.auto.tfvars.json" || cleanup_failed=1
fi

if kind get clusters 2>/dev/null | grep -Fxq "${kind_context#kind-}" && [[ -s "$kind_kubeconfig" ]]; then
  kubectl --kubeconfig "$kind_kubeconfig" --context "$kind_context" -n argocd \
    delete secret idp-aws-oci idp-aws-cluster --ignore-not-found >/dev/null 2>&1 || cleanup_failed=1
fi

if [[ -s "$runtime_dir/bootstrap-inventory.json" ]]; then
  cluster_name=$(jq -r .cluster_name "$runtime_dir/bootstrap-inventory.json")
  aws_region=$(jq -r .aws_region "$runtime_dir/bootstrap-inventory.json")
  account_id=$(jq -r .account_id "$runtime_dir/bootstrap-inventory.json")
  ecr_registry="${account_id}.dkr.ecr.${aws_region}.amazonaws.com"
  docker logout "$ecr_registry" >/dev/null 2>&1 || true
  if aws eks describe-cluster --region "$aws_region" --name "$cluster_name" >/dev/null 2>&1; then
    echo "cleanup verification failed: EKS cluster still exists: $cluster_name" >&2
    cleanup_failed=1
  fi
  while IFS= read -r repository_url; do
    repository_name=${repository_url#*/}
    if aws ecr describe-repositories --region "$aws_region" --repository-names "$repository_name" >/dev/null 2>&1; then
      echo "cleanup verification failed: ECR repository still exists: $repository_name" >&2
      cleanup_failed=1
    fi
  done < <(jq -r '.ecr_repositories[]' "$runtime_dir/bootstrap-inventory.json")
  vpc_count=$(aws ec2 describe-vpcs --region "$aws_region" \
    --filters Name=tag:idp.task,Values=uc3-codex-mvp-1 --query 'length(Vpcs)' --output text 2>/dev/null || echo 1)
  [[ "$vpc_count" == "0" ]] || { echo "cleanup verification failed: task-owned VPC remains" >&2; cleanup_failed=1; }
fi

if [[ -s "$runtime_dir/aurora-inventory.json" ]]; then
  cluster_arn=$(jq -r .infrastructure_reference "$runtime_dir/aurora-inventory.json")
  cluster_id=${cluster_arn##*:}
  if aws rds describe-db-clusters --region "${AWS_REGION:-ap-southeast-1}" --db-cluster-identifier "$cluster_id" >/dev/null 2>&1; then
    echo "cleanup verification failed: Aurora cluster still exists: $cluster_id" >&2
    cleanup_failed=1
  fi
fi

bootstrap_state_count=0
aurora_state_count=0
if [[ -s "$bootstrap_state" ]]; then
  bootstrap_state_count=$(terraform -chdir="$terraform_dir" state list -state="$bootstrap_state" 2>/dev/null | wc -l)
fi
if [[ -d "$repo_root/.state/aws/resources" ]]; then
  while IFS= read -r -d '' state_file; do
    count=$(terraform -chdir="$repo_root/terraform/modules/postgres-aurora" state list -state="$state_file" 2>/dev/null | wc -l)
    aurora_state_count=$((aurora_state_count + count))
  done < <(find "$repo_root/.state/aws/resources" -type f -name terraform.tfstate -print0)
fi
[[ "$bootstrap_state_count" == "0" ]] || { echo "cleanup verification failed: bootstrap state is not empty" >&2; cleanup_failed=1; }
[[ "$aurora_state_count" == "0" ]] || { echo "cleanup verification failed: Aurora state is not empty" >&2; cleanup_failed=1; }

managed_secret="ABSENT"
manual_snapshots=0
automated_snapshots=0
ec2_state="ABSENT"
node_volumes=0
nat_gateways=0
elastic_ips=0
application_load_balancers=0
classic_load_balancers=0
iam_roles=0
oidc_providers=0
tagging_api_mappings=0
if [[ -s "$runtime_dir/evidence.json" && -s "$runtime_dir/bootstrap-inventory.json" ]]; then
  secret_arn=$(jq -r '.aurora.credential_secret_arn // empty' "$runtime_dir/evidence.json")
  if [[ -n "$secret_arn" ]] && aws secretsmanager describe-secret --region "$aws_region" --secret-id "$secret_arn" >/dev/null 2>&1; then
    managed_secret="PRESENT"
    echo "cleanup verification failed: RDS-managed secret still exists" >&2
    cleanup_failed=1
  fi
  manual_snapshots=$(aws rds describe-db-cluster-snapshots --region "$aws_region" --snapshot-type manual \
    --query "length(DBClusterSnapshots[?DBClusterIdentifier=='$cluster_id'])" --output text 2>/dev/null || echo 1)
  automated_snapshots=$(aws rds describe-db-cluster-snapshots --region "$aws_region" --snapshot-type automated \
    --query "length(DBClusterSnapshots[?DBClusterIdentifier=='$cluster_id'])" --output text 2>/dev/null || echo 1)
  [[ "$manual_snapshots" == "0" && "$automated_snapshots" == "0" ]] || { echo "cleanup verification failed: RDS snapshots remain" >&2; cleanup_failed=1; }

  provider_id=$(jq -r '.nodes.items[0].spec.providerID // empty' "$runtime_dir/evidence.json")
  instance_id=${provider_id##*/}
  if [[ -n "$instance_id" ]]; then
    ec2_state=$(aws ec2 describe-instances --region "$aws_region" --instance-ids "$instance_id" \
      --query 'Reservations[0].Instances[0].State.Name' --output text 2>/dev/null || echo ABSENT)
    [[ "$ec2_state" == "terminated" || "$ec2_state" == "ABSENT" || "$ec2_state" == "None" ]] || {
      echo "cleanup verification failed: EKS node is not terminated: $instance_id ($ec2_state)" >&2
      cleanup_failed=1
    }
    volume_ids=$(aws ec2 describe-instances --region "$aws_region" --instance-ids "$instance_id" \
      --query 'Reservations[0].Instances[0].BlockDeviceMappings[].Ebs.VolumeId' --output text 2>/dev/null || true)
    if [[ -n "$volume_ids" && "$volume_ids" != "None" ]]; then
      node_volumes=$(aws ec2 describe-volumes --region "$aws_region" --volume-ids $volume_ids \
        --query 'length(Volumes)' --output text 2>/dev/null || echo 0)
    fi
    [[ "$node_volumes" == "0" ]] || { echo "cleanup verification failed: EKS node volume remains" >&2; cleanup_failed=1; }
  fi

  vpc_id=$(jq -r .vpc_id "$runtime_dir/bootstrap-inventory.json")
  nat_gateways=$(aws ec2 describe-nat-gateways --region "$aws_region" --filter Name=vpc-id,Values="$vpc_id" \
    --query 'length(NatGateways)' --output text 2>/dev/null || echo 1)
  elastic_ips=$(aws ec2 describe-addresses --region "$aws_region" --filters Name=tag:idp.task,Values=uc3-codex-mvp-1 \
    --query 'length(Addresses)' --output text 2>/dev/null || echo 1)
  application_load_balancers=$(aws elbv2 describe-load-balancers --region "$aws_region" \
    --query "length(LoadBalancers[?VpcId=='$vpc_id'])" --output text 2>/dev/null || echo 1)
  classic_load_balancers=$(aws elb describe-load-balancers --region "$aws_region" \
    --query "length(LoadBalancerDescriptions[?VPCId=='$vpc_id'])" --output text 2>/dev/null || echo 1)
  [[ "$nat_gateways" == "0" && "$elastic_ips" == "0" && "$application_load_balancers" == "0" && "$classic_load_balancers" == "0" ]] || {
    echo "cleanup verification failed: a network billing resource remains" >&2
    cleanup_failed=1
  }

  iam_roles=$(aws iam list-roles --query "length(Roles[?starts_with(RoleName, '$cluster_name')])" --output text 2>/dev/null || echo 1)
  endpoint=$(jq -r .cluster_endpoint "$runtime_dir/bootstrap-inventory.json")
  oidc_id=${endpoint#https://}
  oidc_id=${oidc_id%%.*}
  oidc_providers=$(aws iam list-open-id-connect-providers \
    --query "length(OpenIDConnectProviderList[?contains(Arn, '$oidc_id')])" --output text 2>/dev/null || echo 1)
  [[ "$iam_roles" == "0" && "$oidc_providers" == "0" ]] || { echo "cleanup verification failed: IAM/OIDC resource remains" >&2; cleanup_failed=1; }
fi

# Resource Groups Tagging can briefly retain the tags of an already-deleted
# EC2 security-group rule. Clear only that known tombstone shape after the
# native VPC checks prove the parent network no longer exists. Any other ARN is
# treated as a real cleanup failure and remains tagged for investigation.
if [[ -s "$runtime_dir/bootstrap-inventory.json" ]]; then
  mapfile -t tagged_arns < <(aws resourcegroupstaggingapi get-resources --region "$aws_region" \
    --tag-filters Key=idp.task,Values=uc3-codex-mvp-1 --query 'ResourceTagMappingList[].ResourceARN' --output text 2>/dev/null | tr '\t' '\n')
  for tagged_arn in "${tagged_arns[@]}"; do
    [[ -n "$tagged_arn" ]] || continue
    if [[ "$vpc_count" == "0" && "$tagged_arn" == arn:aws:ec2:"$aws_region":"$account_id":security-group-rule/* ]]; then
      aws resourcegroupstaggingapi untag-resources --region "$aws_region" --resource-arn-list "$tagged_arn" \
        --tag-keys idp.managed-by idp.task >/dev/null || cleanup_failed=1
    else
      echo "cleanup verification failed: tagged task resource remains: $tagged_arn" >&2
      cleanup_failed=1
    fi
  done
  tagging_api_mappings=$(aws resourcegroupstaggingapi get-resources --region "$aws_region" \
    --tag-filters Key=idp.task,Values=uc3-codex-mvp-1 --query 'length(ResourceTagMappingList)' --output text 2>/dev/null || echo 1)
  [[ "$tagging_api_mappings" == "0" ]] || { echo "cleanup verification failed: task tag mappings remain" >&2; cleanup_failed=1; }
fi

if [[ "$cleanup_failed" -ne 0 ]]; then
  echo "AWS cleanup is incomplete; inspect the errors above and retained Terraform state" >&2
  exit 1
fi

jq -n \
  --arg region "${aws_region:-${AWS_REGION:-ap-southeast-1}}" \
  --arg managed_secret "$managed_secret" \
  --arg ec2_state "$ec2_state" \
  --argjson bootstrap_state "$bootstrap_state_count" \
  --argjson aurora_state "$aurora_state_count" \
  --argjson manual_snapshots "$manual_snapshots" \
  --argjson automated_snapshots "$automated_snapshots" \
  --argjson node_volumes "$node_volumes" \
  --argjson nat_gateways "$nat_gateways" \
  --argjson elastic_ips "$elastic_ips" \
  --argjson application_load_balancers "$application_load_balancers" \
  --argjson classic_load_balancers "$classic_load_balancers" \
  --argjson iam_roles "$iam_roles" \
  --argjson oidc_providers "$oidc_providers" \
  --argjson tagging_api_mappings "$tagging_api_mappings" \
  '{verifiedAt:(now|todate),region:$region,task:"uc3-codex-mvp-1",terraformBootstrapResources:$bootstrap_state,terraformAuroraResources:$aurora_state,eksCluster:"ABSENT",auroraCluster:"ABSENT",rdsManagedSecret:$managed_secret,manualSnapshots:$manual_snapshots,automatedSnapshots:$automated_snapshots,ec2NodeState:$ec2_state,nodeVolumesPresent:$node_volumes,vpcPresent:0,natGateways:$nat_gateways,elasticIPs:$elastic_ips,applicationLoadBalancers:$application_load_balancers,classicLoadBalancers:$classic_load_balancers,iamRoles:$iam_roles,oidcProviders:$oidc_providers,taggingApiMappings:$tagging_api_mappings}' \
  >"$runtime_dir/cleanup-evidence.json"
chmod 600 "$runtime_dir/cleanup-evidence.json"

echo "AWS cleanup verified: no task EKS, ECR, Aurora, VPC, snapshot, volume, load balancer, IAM, managed secret, or tag mapping remains"
