package provisioner

import (
	"testing"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

func TestAuroraVariablesUseOnlyAWSContext(t *testing.T) {
	request := ApplyRequest{
		Item: domain.InfrastructurePlanItem{
			ReferencedResourceInstanceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
			ResolvedParameters: map[string]any{
				"databaseName": "notes",
				"minCapacity":  "0.5",
			},
		},
		Context: domain.DeploymentContext{
			CloudProvider: "aws",
			Region:        "ap-southeast-1",
			ProvisionerInputs: map[string]any{
				"dbSubnetGroupName":   "idp-demo",
				"vpcSecurityGroupIds": []any{"sg-0123456789abcdef0"},
			},
		},
	}
	variables, err := terraformVariables(request, TerraformModule{TargetKind: targetAurora})
	if err != nil {
		t.Fatal(err)
	}
	if variables["aws_region"] != "ap-southeast-1" || variables["database_name"] != "notes" {
		t.Fatalf("variables = %#v", variables)
	}
	if _, exists := variables["kubeconfig_path"]; exists {
		t.Fatalf("Aurora variables unexpectedly contain Kubernetes credentials: %#v", variables)
	}
}

func TestKubernetesVariablesRequireExplicitContextAndSecretReference(t *testing.T) {
	request := ApplyRequest{
		Context: domain.DeploymentContext{AdapterVersions: map[string]string{
			"kubeconfigPath": "/tmp/kubeconfig",
			"kubeContext":    "kind-dev",
		}},
		SecretRefs: []domain.SecretReference{{Name: "DB_PASSWORD", SecretName: "notes-db"}},
	}
	variables, err := terraformVariables(request, TerraformModule{TargetKind: targetKubernetes})
	if err != nil {
		t.Fatal(err)
	}
	if variables["kube_context"] != "kind-dev" || variables["db_secret_name"] != "notes-db" {
		t.Fatalf("variables = %#v", variables)
	}
}

func TestAuroraVariablesRejectMissingNetworkIdentity(t *testing.T) {
	request := ApplyRequest{
		Item: domain.InfrastructurePlanItem{ReferencedResourceInstanceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"},
		Context: domain.DeploymentContext{
			CloudProvider: "aws",
			Region:        "ap-southeast-1",
		},
	}
	if _, err := terraformVariables(request, TerraformModule{TargetKind: targetAurora}); err == nil {
		t.Fatal("expected missing network inputs to fail closed")
	}
}
