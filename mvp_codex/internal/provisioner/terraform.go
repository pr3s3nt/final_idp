package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type Terraform struct {
	Binary         string
	RepositoryRoot string
	StateRoot      string
	Modules        map[string]TerraformModule
}

type TerraformModule struct {
	Directory                     string
	TargetKind                    string
	InfrastructureReferenceOutput string
}

const (
	targetKubernetes = "kubernetes"
	targetAurora     = "aurora"
)

func NewTerraform(repositoryRoot, stateRoot string) *Terraform {
	return &Terraform{
		Binary:         "terraform",
		RepositoryRoot: repositoryRoot,
		StateRoot:      stateRoot,
		Modules: map[string]TerraformModule{
			"modules/postgres-kubernetes@v1": {Directory: "terraform/modules/postgres-kubernetes", TargetKind: targetKubernetes, InfrastructureReferenceOutput: "infrastructure_reference"},
			"modules/postgres-aurora@v1":     {Directory: "terraform/modules/postgres-aurora", TargetKind: targetAurora, InfrastructureReferenceOutput: "infrastructure_reference"},
		},
	}
}

func (t *Terraform) Apply(ctx context.Context, request ApplyRequest) (ApplyResult, error) {
	_, module, moduleDir, stateFile, err := t.resolve(request.Item)
	if err != nil {
		return ApplyResult{}, err
	}
	if request.Item.Action != domain.PlanCreate {
		return ApplyResult{}, fmt.Errorf("UNSUPPORTED_MVP_OPERATION: terraform apply action %s", request.Item.Action)
	}
	variables, err := terraformVariables(request, module)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o700); err != nil {
		return ApplyResult{}, fmt.Errorf("create durable state directory: %w", err)
	}
	varFile := filepath.Join(filepath.Dir(stateFile), "inputs.auto.tfvars.json")
	b, err := json.MarshalIndent(variables, "", "  ")
	if err != nil {
		return ApplyResult{}, fmt.Errorf("marshal terraform variables: %w", err)
	}
	if err := os.WriteFile(varFile, b, 0o600); err != nil {
		return ApplyResult{}, fmt.Errorf("write terraform variables: %w", err)
	}
	if _, err := t.run(ctx, moduleDir, "init", "-backend=false", "-input=false"); err != nil {
		return ApplyResult{}, err
	}
	if _, err := t.run(ctx, moduleDir, "apply", "-input=false", "-auto-approve", "-lock=true", "-state="+stateFile, "-var-file="+varFile); err != nil {
		return ApplyResult{}, err
	}
	expectedOutputs := append([]string(nil), request.Item.ExposedOutputs...)
	if module.InfrastructureReferenceOutput != "" {
		expectedOutputs = append(expectedOutputs, module.InfrastructureReferenceOutput)
	}
	outputs, err := t.readOutputs(ctx, moduleDir, stateFile, expectedOutputs)
	if err != nil {
		return ApplyResult{}, err
	}
	infrastructureReference := request.Item.ReferencedResourceInstanceID
	if module.InfrastructureReferenceOutput != "" {
		infrastructureReference = outputs[module.InfrastructureReferenceOutput]
		delete(outputs, module.InfrastructureReferenceOutput)
	}
	if infrastructureReference == "" {
		return ApplyResult{}, fmt.Errorf("TERRAFORM_OUTPUT_MISSING: infrastructure reference is empty")
	}
	return ApplyResult{InfrastructureReference: infrastructureReference, Outputs: outputs}, nil
}

func (t *Terraform) Inspect(ctx context.Context, request ApplyRequest) (InspectorResult, error) {
	_, _, moduleDir, stateFile, err := t.resolve(request.Item)
	if err != nil {
		return InspectorResult{}, err
	}
	if _, err := os.Stat(stateFile); err != nil {
		if os.IsNotExist(err) {
			return InspectorResult{Exists: false}, nil
		}
		return InspectorResult{}, err
	}
	outputs, err := t.readOutputs(ctx, moduleDir, stateFile, request.Item.ExposedOutputs)
	if err != nil {
		return InspectorResult{}, err
	}
	return InspectorResult{Exists: true, Compatible: true, Outputs: outputs}, nil
}

func (t *Terraform) Outputs(ctx context.Context, item domain.InfrastructurePlanItem) (map[string]string, error) {
	_, _, moduleDir, stateFile, err := t.resolve(item)
	if err != nil {
		return nil, err
	}
	return t.readOutputs(ctx, moduleDir, stateFile, item.ExposedOutputs)
}

func (t *Terraform) resolve(item domain.InfrastructurePlanItem) (Reference, TerraformModule, string, string, error) {
	reference, err := ParseReference(item.ProvisionerReference)
	if err != nil {
		return Reference{}, TerraformModule{}, "", "", err
	}
	key := reference.Module + "@" + reference.Version
	module, ok := t.Modules[key]
	if !ok {
		return Reference{}, TerraformModule{}, "", "", fmt.Errorf("TERRAFORM_MODULE_NOT_ALLOWLISTED: %s", key)
	}
	moduleDir, err := secureJoin(t.RepositoryRoot, module.Directory)
	if err != nil {
		return Reference{}, TerraformModule{}, "", "", err
	}
	stateFile, err := secureJoin(t.StateRoot, item.ProviderStateReference)
	if err != nil {
		return Reference{}, TerraformModule{}, "", "", err
	}
	return reference, module, moduleDir, stateFile, nil
}

func terraformVariables(request ApplyRequest, module TerraformModule) (map[string]any, error) {
	vars := make(map[string]any, len(request.Item.ResolvedParameters)+5)
	for key, value := range request.Item.ResolvedParameters {
		vars[snakeCase(key)] = value
	}
	switch module.TargetKind {
	case targetKubernetes:
		vars["kubeconfig_path"] = request.Context.AdapterVersions["kubeconfigPath"]
		vars["kube_context"] = request.Context.AdapterVersions["kubeContext"]
		if vars["kubeconfig_path"] == "" || vars["kube_context"] == "" {
			return nil, fmt.Errorf("INVALID_DEPLOYMENT_TARGET: explicit kubeconfigPath and kubeContext are required")
		}
		for _, secret := range request.SecretRefs {
			if secret.Name == "DB_PASSWORD" {
				vars["db_secret_name"] = secret.SecretName
			}
		}
		if vars["db_secret_name"] == nil {
			return nil, fmt.Errorf("INVALID_SECRET_REFERENCE: DB_PASSWORD reference is required for the Kubernetes PostgreSQL definition")
		}
	case targetAurora:
		if !strings.EqualFold(request.Context.CloudProvider, "aws") || request.Context.Region == "" {
			return nil, fmt.Errorf("INVALID_DEPLOYMENT_TARGET: Aurora requires an explicit AWS region")
		}
		vars["aws_region"] = request.Context.Region
		vars["resource_instance_id"] = request.Item.ReferencedResourceInstanceID
		for _, key := range []string{"dbSubnetGroupName", "vpcSecurityGroupIds"} {
			value, ok := request.Context.ProvisionerInputs[key]
			if !ok {
				return nil, fmt.Errorf("INVALID_DEPLOYMENT_TARGET: Aurora requires provisioner input %s", key)
			}
			vars[snakeCase(key)] = value
		}
	default:
		return nil, fmt.Errorf("TERRAFORM_MODULE_NOT_ALLOWLISTED: unsupported target kind %q", module.TargetKind)
	}
	return vars, nil
}

func (t *Terraform) readOutputs(ctx context.Context, moduleDir, stateFile string, expected []string) (map[string]string, error) {
	b, err := t.run(ctx, moduleDir, "output", "-json", "-state="+stateFile)
	if err != nil {
		return nil, err
	}
	var raw map[string]struct {
		Value     any  `json:"value"`
		Sensitive bool `json:"sensitive"`
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode terraform outputs: %w", err)
	}
	outputs := make(map[string]string, len(expected))
	for _, name := range expected {
		output, ok := raw[name]
		if !ok {
			return nil, fmt.Errorf("TERRAFORM_OUTPUT_MISSING: %s", name)
		}
		if output.Sensitive {
			return nil, fmt.Errorf("TERRAFORM_OUTPUT_SENSITIVE: %s cannot enter resolved configuration", name)
		}
		outputs[name] = fmt.Sprint(output.Value)
	}
	return outputs, nil
}

func (t *Terraform) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, t.Binary, args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "TF_IN_AUTOMATION=1")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("terraform %s failed: %w: %s", args[0], err, redact(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func secureJoin(root, name string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	name = strings.TrimPrefix(filepath.Clean(name), string(filepath.Separator))
	joined := filepath.Join(rootAbs, name)
	if joined != rootAbs && !strings.HasPrefix(joined, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("INVALID_PATH: %q escapes configured root", name)
	}
	return joined, nil
}

func snakeCase(value string) string {
	var result strings.Builder
	for i, r := range value {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + ('a' - 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func redact(stderr string) string {
	if len(stderr) > 2048 {
		stderr = stderr[len(stderr)-2048:]
	}
	return strings.TrimSpace(stderr)
}
