package render

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/configuration"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"gopkg.in/yaml.v3"
)

type Score struct{ Binary string }

func NewScore() *Score { return &Score{Binary: "score-k8s"} }

func (s *Score) Generate(ctx context.Context, deploymentID string, snapshot domain.DeploymentInputSnapshot, resolved configuration.Resolved) ([]byte, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("INVALID_DEPLOYMENT: deployment id is required")
	}
	tempDir, err := os.MkdirTemp("", "idp-score-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)
	if _, err := s.run(ctx, tempDir, "init", "--no-sample", "--no-default-provisioners"); err != nil {
		return nil, err
	}
	images := make(map[string]domain.WorkloadImage, len(snapshot.Images))
	for _, image := range snapshot.Images {
		images[image.WorkloadID] = image
	}
	workloads := append([]domain.Workload(nil), snapshot.ApplicationDefinition.Workloads...)
	sort.Slice(workloads, func(i, j int) bool { return workloads[i].ID < workloads[j].ID })
	paths := make([]string, 0, len(workloads))
	for _, workload := range workloads {
		image, ok := images[workload.ID]
		if !ok {
			return nil, fmt.Errorf("INVALID_IMAGE: workload %s has no image", workload.ID)
		}
		name, err := configuration.WorkloadServiceName(snapshot, workload)
		if err != nil {
			return nil, err
		}
		config, ok := resolved.Workloads[workload.ID]
		if !ok {
			return nil, fmt.Errorf("CONFIGURATION_NOT_RESOLVED: workload %s", workload.ID)
		}
		variables := map[string]string{}
		for key, value := range config.Environment {
			variables[key] = value
		}
		for _, secret := range config.Secrets {
			variables[secret.Name] = fmt.Sprintf("🔐💬%s_%s💬🔐", secret.SecretName, secret.Key)
		}
		spec := map[string]any{"apiVersion": "score.dev/v1b1", "metadata": map[string]any{"name": name}, "containers": map[string]any{"main": map[string]any{"image": workload.ImageRepository + "@" + image.Digest, "variables": variables}}}
		if workload.Port > 0 {
			spec["service"] = map[string]any{"ports": map[string]any{"http": map[string]any{"port": workload.Port, "targetPort": workload.Port}}}
		}
		content, err := yaml.Marshal(spec)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(tempDir, name+".score.yaml")
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	args := []string{"generate"}
	args = append(args, paths...)
	args = append(args, "--namespace", snapshot.RenderContext.Namespace, "--output", filepath.Join(tempDir, "manifests.yaml"))
	if _, err := s.run(ctx, tempDir, args...); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(tempDir, "manifests.yaml"))
	if err != nil {
		return nil, err
	}
	return adaptManifest(raw, deploymentID, snapshot, images, resolved)
}

func adaptManifest(raw []byte, deploymentID string, snapshot domain.DeploymentInputSnapshot, images map[string]domain.WorkloadImage, resolved configuration.Resolved) ([]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var documents []map[string]any
	expectedImages := map[string]string{}
	for _, workload := range snapshot.ApplicationDefinition.Workloads {
		name, err := configuration.WorkloadServiceName(snapshot, workload)
		if err != nil {
			return nil, err
		}
		expectedImages[name] = workload.ImageRepository + "@" + images[workload.ID].Digest
	}
	seenDeployments := map[string]bool{}
	for {
		var object map[string]any
		err := decoder.Decode(&object)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode score manifest: %w", err)
		}
		if len(object) == 0 {
			continue
		}
		metadata, ok := object["metadata"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("INVALID_MANIFEST: metadata missing")
		}
		metadata["namespace"] = snapshot.RenderContext.Namespace
		if object["kind"] == "Deployment" {
			name := fmt.Sprint(metadata["name"])
			expected, known := expectedImages[name]
			if !known {
				return nil, fmt.Errorf("INVALID_MANIFEST: unexpected Deployment %s", name)
			}
			spec, err := requiredMap(object, "spec")
			if err != nil {
				return nil, fmt.Errorf("INVALID_MANIFEST: Deployment %s: %w", name, err)
			}
			template, err := requiredMap(spec, "template")
			if err != nil {
				return nil, fmt.Errorf("INVALID_MANIFEST: Deployment %s: %w", name, err)
			}
			templateMetadata := ensureMap(template, "metadata")
			annotations := ensureMap(templateMetadata, "annotations")
			annotations["idp.deployment-id"] = deploymentID
			podSpec, err := requiredMap(template, "spec")
			if err != nil {
				return nil, fmt.Errorf("INVALID_MANIFEST: Deployment %s: %w", name, err)
			}
			containers, ok := podSpec["containers"].([]any)
			if !ok || len(containers) != 1 {
				return nil, fmt.Errorf("INVALID_MANIFEST: Deployment %s must have one container", name)
			}
			container, ok := containers[0].(map[string]any)
			if !ok || container["image"] != expected {
				return nil, fmt.Errorf("INVALID_MANIFEST: Deployment %s image is not the expected digest", name)
			}
			seenDeployments[name] = true
		}
		documents = append(documents, object)
	}
	for name := range expectedImages {
		if !seenDeployments[name] {
			return nil, fmt.Errorf("INVALID_MANIFEST: expected Deployment %s missing", name)
		}
	}
	for _, materialization := range resolved.Materializations {
		object, err := externalSecretObject(deploymentID, snapshot, materialization)
		if err != nil {
			return nil, err
		}
		documents = append(documents, object)
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	defer encoder.Close()
	for _, object := range documents {
		if err := encoder.Encode(object); err != nil {
			return nil, err
		}
	}
	return output.Bytes(), nil
}

func requiredMap(parent map[string]any, key string) (map[string]any, error) {
	value, ok := parent[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("manifest key %s is not a map", key)
	}
	return value, nil
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

func externalSecretObject(deploymentID string, snapshot domain.DeploymentInputSnapshot, materialization configuration.SecretMaterialization) (map[string]any, error) {
	if !strings.EqualFold(materialization.Provider, "aws") {
		return nil, fmt.Errorf("UNSUPPORTED_SECRET_PROVIDER: %s", materialization.Provider)
	}
	if materialization.Namespace != snapshot.RenderContext.Namespace || materialization.SecretName == "" || materialization.SecretKey == "" {
		return nil, fmt.Errorf("INVALID_SECRET_MATERIALIZATION: destination does not match deployment namespace")
	}
	if !strings.HasPrefix(materialization.RemoteReference, "arn:") || !strings.Contains(materialization.RemoteReference, ":secretsmanager:") {
		return nil, fmt.Errorf("INVALID_SECRET_MATERIALIZATION: AWS Secrets Manager ARN is required")
	}
	apiVersion := snapshot.RenderContext.AdapterVersions["externalSecretsApiVersion"]
	if apiVersion != "external-secrets.io/v1" && apiVersion != "external-secrets.io/v1beta1" {
		return nil, fmt.Errorf("UNSUPPORTED_SECRET_ADAPTER: explicit External Secrets API version is required")
	}
	storeName := snapshot.RenderContext.AdapterVersions["clusterSecretStore"]
	if storeName == "" {
		return nil, fmt.Errorf("INVALID_SECRET_ADAPTER: clusterSecretStore is required")
	}
	labels := map[string]any{
		"app.kubernetes.io/managed-by": "idp-argocd",
		"idp.deployment-id":            deploymentID,
	}
	return map[string]any{
		"apiVersion": apiVersion,
		"kind":       "ExternalSecret",
		"metadata": map[string]any{
			"name":      materialization.SecretName,
			"namespace": materialization.Namespace,
			"labels":    labels,
		},
		"spec": map[string]any{
			"refreshInterval": "1h",
			"secretStoreRef": map[string]any{
				"name": storeName,
				"kind": "ClusterSecretStore",
			},
			"target": map[string]any{
				"name":           materialization.SecretName,
				"creationPolicy": "Owner",
			},
			"data": []any{map[string]any{
				"secretKey": materialization.SecretKey,
				"remoteRef": map[string]any{
					"key":      materialization.RemoteReference,
					"property": materialization.RemoteProperty,
				},
			}},
		},
	}, nil
}

func (s *Score) run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, s.Binary, args...)
	command.Dir = dir
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("score-k8s %s failed: %w: %s", args[0], err, stderr.String())
	}
	return stdout.Bytes(), nil
}
