package configuration

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type ResourceOutputs map[string]map[string]string

type MaterializedSecret struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
}

type WorkloadConfiguration struct {
	Environment map[string]string    `json:"environment"`
	Secrets     []MaterializedSecret `json:"secrets"`
}

type Resolved struct {
	Workloads        map[string]WorkloadConfiguration `json:"workloads"`
	Materializations []SecretMaterialization          `json:"materializations,omitempty"`
}

type SecretMaterialization struct {
	Provider        string `json:"provider"`
	Namespace       string `json:"namespace"`
	SecretName      string `json:"secretName"`
	SecretKey       string `json:"secretKey"`
	RemoteReference string `json:"remoteReference"`
	RemoteProperty  string `json:"remoteProperty"`
}

type Resolver struct{}

func NewResolver() *Resolver { return &Resolver{} }

func (r *Resolver) Resolve(snapshot domain.DeploymentInputSnapshot, outputs ResourceOutputs) (Resolved, error) {
	workloads := make(map[string]domain.Workload, len(snapshot.ApplicationDefinition.Workloads))
	resolved := Resolved{Workloads: make(map[string]WorkloadConfiguration, len(snapshot.ApplicationDefinition.Workloads))}
	for _, workload := range snapshot.ApplicationDefinition.Workloads {
		workloads[workload.ID] = workload
		resolved.Workloads[workload.ID] = WorkloadConfiguration{Environment: map[string]string{}}
	}
	for _, value := range snapshot.EnvironmentConfiguration.Values {
		config, ok := resolved.Workloads[value.WorkloadID]
		if !ok {
			return Resolved{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: unknown workload %s", value.WorkloadID)
		}
		var resolvedValue string
		switch value.Source {
		case domain.ValueDirect:
			resolvedValue = value.DirectValue
		case domain.ValueResourceOutput:
			resource, ok := outputs[value.ResourceRequirementID]
			if !ok {
				return Resolved{}, fmt.Errorf("RESOURCE_OUTPUT_NOT_AVAILABLE: requirement %s", value.ResourceRequirementID)
			}
			resolvedValue, ok = resource[value.ResourceOutputName]
			if !ok {
				return Resolved{}, fmt.Errorf("RESOURCE_OUTPUT_NOT_AVAILABLE: %s.%s", value.ResourceRequirementID, value.ResourceOutputName)
			}
		case domain.ValueWorkloadOutput:
			source, ok := workloads[value.ReferencedWorkloadID]
			if !ok {
				return Resolved{}, fmt.Errorf("WORKLOAD_OUTPUT_NOT_AVAILABLE: workload %s", value.ReferencedWorkloadID)
			}
			output, ok := findPlanTimeOutput(source, value.WorkloadOutputName)
			if !ok {
				return Resolved{}, fmt.Errorf("WORKLOAD_OUTPUT_NOT_AVAILABLE: %s.%s", source.ID, value.WorkloadOutputName)
			}
			var err error
			resolvedValue, err = serviceURL(snapshot, source, output)
			if err != nil {
				return Resolved{}, err
			}
		default:
			return Resolved{}, fmt.Errorf("INVALID_CONFIGURATION_REFERENCE: unsupported source %s", value.Source)
		}
		config.Environment[value.Name] = resolvedValue
		resolved.Workloads[value.WorkloadID] = config
	}
	for _, secret := range snapshot.EnvironmentConfiguration.Secrets {
		config, ok := resolved.Workloads[secret.WorkloadID]
		if !ok {
			return Resolved{}, fmt.Errorf("INVALID_SECRET_REFERENCE: unknown workload %s", secret.WorkloadID)
		}
		if secret.Target != snapshot.RenderContext.TargetID || secret.Namespace != snapshot.RenderContext.Namespace || secret.SecretName == "" || secret.Key == "" {
			return Resolved{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s does not match deployment target", secret.ID)
		}
		source := secret.Source
		if source == "" {
			source = domain.SecretKubernetes
		}
		switch source {
		case domain.SecretKubernetes:
			if secret.UID == "" || secret.ResourceRequirementID != "" || secret.ResourceOutputName != "" {
				return Resolved{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s is not a complete Kubernetes Secret identity", secret.ID)
			}
		case domain.SecretResource:
			resource, ok := outputs[secret.ResourceRequirementID]
			if !ok {
				return Resolved{}, fmt.Errorf("RESOURCE_OUTPUT_NOT_AVAILABLE: secret %s requirement %s", secret.ID, secret.ResourceRequirementID)
			}
			remoteReference, ok := resource[secret.ResourceOutputName]
			if !ok || remoteReference == "" || secret.RemoteProperty == "" {
				return Resolved{}, fmt.Errorf("RESOURCE_OUTPUT_NOT_AVAILABLE: secret %s output %s", secret.ID, secret.ResourceOutputName)
			}
			resolved.Materializations = append(resolved.Materializations, SecretMaterialization{
				Provider:        snapshot.RenderContext.CloudProvider,
				Namespace:       secret.Namespace,
				SecretName:      secret.SecretName,
				SecretKey:       secret.Key,
				RemoteReference: remoteReference,
				RemoteProperty:  secret.RemoteProperty,
			})
		default:
			return Resolved{}, fmt.Errorf("INVALID_SECRET_REFERENCE: %s has unsupported source %s", secret.ID, source)
		}
		config.Secrets = append(config.Secrets, MaterializedSecret{Name: secret.Name, SecretName: secret.SecretName, Key: secret.Key})
		sort.Slice(config.Secrets, func(i, j int) bool { return config.Secrets[i].Name < config.Secrets[j].Name })
		resolved.Workloads[secret.WorkloadID] = config
	}
	sort.Slice(resolved.Materializations, func(i, j int) bool {
		if resolved.Materializations[i].SecretName != resolved.Materializations[j].SecretName {
			return resolved.Materializations[i].SecretName < resolved.Materializations[j].SecretName
		}
		return resolved.Materializations[i].SecretKey < resolved.Materializations[j].SecretKey
	})
	return resolved, nil
}

func findPlanTimeOutput(workload domain.Workload, name string) (domain.WorkloadOutputDefinition, bool) {
	for _, output := range workload.ExposedOutputs {
		if output.Name == name && output.Availability == domain.OutputPlanTime {
			return output, true
		}
	}
	return domain.WorkloadOutputDefinition{}, false
}

var invalidDNS = regexp.MustCompile(`[^a-z0-9-]+`)

func WorkloadServiceName(snapshot domain.DeploymentInputSnapshot, workload domain.Workload) (string, error) {
	if snapshot.RenderContext.NamingPolicy != "mvp-v1" {
		return "", fmt.Errorf("UNSUPPORTED_NAMING_POLICY: %s", snapshot.RenderContext.NamingPolicy)
	}
	name := strings.ToLower(snapshot.ApplicationDefinition.Name + "-" + workload.Name)
	name = invalidDNS.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 63 {
		name = strings.TrimRight(name[:63], "-")
	}
	if name == "" {
		return "", fmt.Errorf("INVALID_WORKLOAD_NAME: %s", workload.Name)
	}
	return name, nil
}

func serviceURL(snapshot domain.DeploymentInputSnapshot, workload domain.Workload, output domain.WorkloadOutputDefinition) (string, error) {
	port := output.Port
	if port == 0 {
		port = workload.Port
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("INVALID_WORKLOAD_OUTPUT: %s.%s has invalid port", workload.ID, output.Name)
	}
	serviceName, err := WorkloadServiceName(snapshot, workload)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, snapshot.RenderContext.Namespace, port), nil
}
