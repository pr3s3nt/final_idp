package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fingerprint"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/identity"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
)

type DeploymentService struct {
	store  *postgres.Store
	engine *platform.Engine
}

type ConfirmDeploymentRequest struct {
	ExpectedPlanFingerprint string         `json:"expectedPlanFingerprint"`
	Overrides               map[string]any `json:"overrides"`
}

type ConfirmDeploymentResponse struct {
	TrackingID string                     `json:"trackingId,omitempty"`
	Code       string                     `json:"code,omitempty"`
	Plan       *domain.InfrastructurePlan `json:"plan,omitempty"`
}

func NewDeploymentService(store *postgres.Store, stateRoot string) *DeploymentService {
	return &DeploymentService{store: store, engine: platform.NewEngine(stateRoot)}
}

type CreateDeploymentRequest struct {
	ApplicationID string                   `json:"applicationId"`
	Environment   string                   `json:"environment"`
	Context       domain.DeploymentContext `json:"context"`
	Images        []domain.WorkloadImage   `json:"images"`
}

type CreateDeploymentResponse struct {
	DeploymentID     string                    `json:"deploymentId"`
	InputFingerprint string                    `json:"inputFingerprint"`
	Plan             domain.InfrastructurePlan `json:"plan"`
	PlanFingerprint  string                    `json:"planFingerprint"`
	Algorithm        string                    `json:"algorithm"`
	GraphOrder       []string                  `json:"graphOrder"`
}

var imageDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func (s *DeploymentService) Create(ctx context.Context, request CreateDeploymentRequest) (CreateDeploymentResponse, error) {
	if err := validateCreate(request); err != nil {
		return CreateDeploymentResponse{}, err
	}
	snapshot, definitions, instances, err := s.store.LoadSource(ctx, postgres.CreateInput{ApplicationID: request.ApplicationID, Environment: request.Environment, Context: request.Context, Images: request.Images})
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	prepared, err := s.engine.Prepare(snapshot, definitions, instances)
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	deploymentID, err := identity.NewUUID()
	if err != nil {
		return CreateDeploymentResponse{}, err
	}
	if err := s.store.PersistPrepared(ctx, deploymentID, prepared); err != nil {
		return CreateDeploymentResponse{}, err
	}
	return CreateDeploymentResponse{DeploymentID: deploymentID, InputFingerprint: prepared.Snapshot.InputFingerprint, Plan: prepared.Plan, PlanFingerprint: prepared.Plan.Fingerprint, Algorithm: prepared.Plan.Algorithm, GraphOrder: prepared.Graph.TopologicalOrder}, nil
}

func (s *DeploymentService) Confirm(ctx context.Context, deploymentID, idempotencyKey string, request ConfirmDeploymentRequest) (ConfirmDeploymentResponse, error) {
	if deploymentID == "" || idempotencyKey == "" || request.ExpectedPlanFingerprint == "" {
		return ConfirmDeploymentResponse{}, fmt.Errorf("INVALID_REQUEST: deployment id, expected fingerprint and Idempotency-Key are required")
	}
	if len(request.Overrides) != 0 {
		return ConfirmDeploymentResponse{}, fmt.Errorf("UNSUPPORTED_MVP_OPERATION: overrides must be empty")
	}
	requestFingerprint, err := fingerprint.Sum(struct {
		DeploymentID string         `json:"deploymentId"`
		Expected     string         `json:"expectedPlanFingerprint"`
		Overrides    map[string]any `json:"overrides"`
	}{deploymentID, request.ExpectedPlanFingerprint, map[string]any{}})
	if err != nil {
		return ConfirmDeploymentResponse{}, err
	}
	result, err := s.store.Confirm(ctx, deploymentID, request.ExpectedPlanFingerprint, idempotencyKey, requestFingerprint, s.engine)
	if err != nil {
		return ConfirmDeploymentResponse{}, err
	}
	if result.PlanChanged {
		return ConfirmDeploymentResponse{Code: "PLAN_CHANGED", Plan: &result.Plan}, nil
	}
	return ConfirmDeploymentResponse{TrackingID: result.TrackingID}, nil
}

func (s *DeploymentService) Get(ctx context.Context, deploymentID string) (postgres.DeploymentDetail, error) {
	if deploymentID == "" {
		return postgres.DeploymentDetail{}, fmt.Errorf("INVALID_REQUEST: deployment id is required")
	}
	return s.store.GetDeploymentDetail(ctx, deploymentID)
}

func validateCreate(request CreateDeploymentRequest) error {
	if request.ApplicationID == "" || request.Environment == "" {
		return fmt.Errorf("INVALID_REQUEST: applicationId and environment are required")
	}
	context := request.Context
	if context.TargetID == "" || context.ClusterIdentity == "" {
		return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: target and cluster identity are required")
	}
	if strings.Contains(context.ClusterIdentity, "REPLACE_") || context.Namespace == "" || context.NamingPolicy != "mvp-v1" {
		return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: concrete cluster identity, namespace and mvp-v1 naming policy are required")
	}
	switch strings.ToLower(context.CloudProvider) {
	case "aws":
		if context.Region == "" || placeholder(context.Region) {
			return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: explicit AWS region is required")
		}
		if err := validateAWSProvisionerInputs(context.ProvisionerInputs); err != nil {
			return err
		}
	case "kind", "local":
		if context.AdapterVersions["kubeconfigPath"] == "" || context.AdapterVersions["kubeContext"] == "" {
			return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: kind/local requires explicit kubeconfigPath and kubeContext")
		}
	default:
		return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: unsupported cloudProvider %q", context.CloudProvider)
	}
	if len(request.Images) == 0 {
		return fmt.Errorf("INVALID_IMAGES: at least one image is required")
	}
	seen := map[string]struct{}{}
	for _, image := range request.Images {
		if image.WorkloadID == "" || image.Tag == "" || !imageDigestPattern.MatchString(image.Digest) {
			return fmt.Errorf("INVALID_IMAGE: workload, tag and immutable sha256 digest are required")
		}
		if _, exists := seen[image.WorkloadID]; exists {
			return fmt.Errorf("INVALID_IMAGE: duplicate workload %s", image.WorkloadID)
		}
		seen[image.WorkloadID] = struct{}{}
	}
	return nil
}

var securityGroupPattern = regexp.MustCompile(`^sg-[0-9a-f]{8,17}$`)

func validateAWSProvisionerInputs(inputs map[string]any) error {
	subnetGroup, ok := inputs["dbSubnetGroupName"].(string)
	if !ok || subnetGroup == "" || placeholder(subnetGroup) {
		return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: concrete dbSubnetGroupName is required")
	}
	groups, ok := inputs["vpcSecurityGroupIds"].([]any)
	if !ok || len(groups) == 0 {
		// Programmatic callers commonly provide []string rather than decoded JSON.
		if stringsList, stringOK := inputs["vpcSecurityGroupIds"].([]string); stringOK {
			groups = make([]any, len(stringsList))
			for index, value := range stringsList {
				groups[index] = value
			}
		} else {
			return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: vpcSecurityGroupIds is required")
		}
	}
	for _, raw := range groups {
		group, ok := raw.(string)
		if !ok || !securityGroupPattern.MatchString(group) {
			return fmt.Errorf("INVALID_DEPLOYMENT_TARGET: invalid VPC security-group ID")
		}
	}
	return nil
}

func placeholder(value string) bool {
	return strings.Contains(strings.ToUpper(value), "REPLACE_")
}
