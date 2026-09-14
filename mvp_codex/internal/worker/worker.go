package worker

import (
	"context"
	"fmt"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/configuration"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/provisioner"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/render"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
)

type Store interface {
	ClaimNext(context.Context, string) (postgres.ClaimedJob, bool, error)
	PlanningData(context.Context, domain.DeploymentInputSnapshot) ([]domain.ResourceDefinition, map[string]domain.ResourceInstance, error)
	ReserveResource(context.Context, string, string, domain.DeploymentInputSnapshot, domain.InfrastructurePlanItem) error
	MarkProvisioning(context.Context, string, string, string) error
	CheckpointReady(context.Context, string, string, string, string) error
	AdvanceStep(context.Context, string, string, string, string, string) error
	SavePublicationIntent(context.Context, string, string, string, string, string) error
	CompleteExecution(context.Context, string, string, string, string) error
	FailExecution(context.Context, string, string, string, string, bool) error
}

type Artifact struct{ URI, Digest string }
type Acknowledgment struct{ Reference, Digest string }
type ArtifactPublisher interface {
	Publish(context.Context, string, []byte) (Artifact, error)
}
type Delivery interface {
	Apply(context.Context, string, domain.DeploymentContext, Artifact) (Acknowledgment, error)
}

type Worker struct {
	Store     Store
	Engine    *platform.Engine
	Registry  *provisioner.Registry
	Renderer  *render.Score
	Artifacts ArtifactPublisher
	Delivery  Delivery
}

func (w *Worker) RunOnce(ctx context.Context, workerRunID string) (bool, error) {
	job, found, err := w.Store.ClaimNext(ctx, workerRunID)
	if err != nil || !found {
		return found, err
	}
	fail := func(code string, cause error, recovery bool) (bool, error) {
		persistErr := w.Store.FailExecution(ctx, job.JobID, workerRunID, code, cause.Error(), recovery)
		if persistErr != nil {
			return true, fmt.Errorf("%v; persist failure: %w", cause, persistErr)
		}
		return true, cause
	}
	definitions, instances, err := w.Store.PlanningData(ctx, job.Snapshot)
	if err != nil {
		return fail("PLAN_PRECHECK_FAILED", err, false)
	}
	prepared, err := w.Engine.Prepare(job.Snapshot, definitions, instances)
	if err != nil {
		return fail("PLAN_PRECHECK_FAILED", err, false)
	}
	if prepared.Plan.Fingerprint != job.AcceptedPlanFingerprint {
		return fail("PLAN_STALE_AFTER_ACCEPT", fmt.Errorf("accepted plan fingerprint changed"), false)
	}
	outputs := configuration.ResourceOutputs{}
	for _, item := range prepared.Plan.Items {
		if err := w.Store.ReserveResource(ctx, job.JobID, workerRunID, job.Snapshot, item); err != nil {
			return fail("RESOURCE_RESERVATION_FAILED", err, true)
		}
		provider, _, err := w.Registry.Resolve(item.ProvisionerReference)
		if err != nil {
			return fail("PROVISIONER_RESOLUTION_FAILED", err, false)
		}
		request := provisioner.ApplyRequest{Item: item, Context: job.Snapshot.RenderContext, SecretRefs: job.Snapshot.EnvironmentConfiguration.Secrets}
		if item.Action == domain.PlanCreate {
			if err := w.Store.MarkProvisioning(ctx, job.JobID, workerRunID, item.ReferencedResourceInstanceID); err != nil {
				return fail("RESOURCE_CHECKPOINT_FAILED", err, true)
			}
			result, err := provider.Apply(ctx, request)
			if err != nil {
				return fail("PROVISIONER_APPLY_FAILED", err, true)
			}
			if err := w.Store.CheckpointReady(ctx, job.JobID, workerRunID, item.ReferencedResourceInstanceID, result.InfrastructureReference); err != nil {
				return fail("RESOURCE_CHECKPOINT_FAILED", err, true)
			}
			outputs[item.ResourceRequirementID] = result.Outputs
		} else {
			inspection, err := provider.Inspect(ctx, request)
			if err != nil {
				return fail("RESOURCE_DRIFT", err, true)
			}
			if !inspection.Exists || !inspection.Compatible {
				return fail("RESOURCE_DRIFT", fmt.Errorf("reused resource is absent or incompatible"), true)
			}
			outputs[item.ResourceRequirementID] = inspection.Outputs
		}
	}
	if err := w.Store.AdvanceStep(ctx, job.JobID, workerRunID, "INFRASTRUCTURE_READY", "CONFIGURATION_RESOLVED", "CONFIGURATION"); err != nil {
		return fail("CHECKPOINT_FAILED", err, true)
	}
	resolved, err := configuration.NewResolver().Resolve(job.Snapshot, outputs)
	if err != nil {
		return fail("CONFIGURATION_RESOLUTION_FAILED", err, false)
	}
	if err := w.Store.AdvanceStep(ctx, job.JobID, workerRunID, "CONFIGURATION_RESOLVED", "MANIFEST_GENERATED", "MANIFEST"); err != nil {
		return fail("CHECKPOINT_FAILED", err, true)
	}
	manifest, err := w.Renderer.Generate(ctx, job.DeploymentID, job.Snapshot, resolved)
	if err != nil {
		return fail("MANIFEST_GENERATION_FAILED", err, false)
	}
	if err := w.Store.AdvanceStep(ctx, job.JobID, workerRunID, "MANIFEST_GENERATED", "", "PUBLISH"); err != nil {
		return fail("CHECKPOINT_FAILED", err, true)
	}
	artifact, err := w.Artifacts.Publish(ctx, job.DeploymentID, manifest)
	if err != nil {
		return fail("ARTIFACT_PUBLISH_FAILED", err, true)
	}
	applicationName := "idp-" + job.Snapshot.ApplicationDefinition.Name + "-" + job.Snapshot.EnvironmentConfiguration.Environment
	if err := w.Store.SavePublicationIntent(ctx, job.JobID, workerRunID, artifact.URI, artifact.Digest, applicationName); err != nil {
		return fail("PUBLICATION_INTENT_FAILED", err, true)
	}
	ack, err := w.Delivery.Apply(ctx, applicationName, job.Snapshot.RenderContext, artifact)
	if err != nil {
		return fail("DELIVERY_FAILED", err, true)
	}
	if ack.Digest != artifact.Digest {
		return fail("DELIVERY_ACK_MISMATCH", fmt.Errorf("delivery acknowledged another digest"), true)
	}
	if err := w.Store.CompleteExecution(ctx, job.JobID, workerRunID, ack.Reference, ack.Digest); err != nil {
		return true, err
	}
	return true, nil
}
