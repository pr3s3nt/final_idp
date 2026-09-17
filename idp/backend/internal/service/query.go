package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"sdp/internal/domain"
	"sdp/internal/domain/resourceoutput"
	"sdp/internal/integration/cd"
	"sdp/internal/integration/kubernetes"
	"sdp/internal/persistence"
)

// QueryService is the Deployment Query Service (the UC-04 subset UC-03 needs
// to track a deployment). It only reads.
type QueryService struct {
	Repositories
	Orch            *Orchestrator
	Kube            WorkloadStatusProvider
	CD              cd.Integration
	ResourceOutputs *resourceoutput.Collector
}

type ImageRow struct {
	WorkloadID      string `json:"workloadId"`
	Workload        string `json:"workload"`
	Image           string `json:"image"`
	InclusionReason string `json:"inclusionReason"`
	Wave            int    `json:"wave"`
	// RunningHere reports whether this deployment's image is what currently runs.
	RunningHere bool   `json:"runningHere"`
	ReplacedBy  string `json:"replacedBy,omitempty"`
}

type InfraRow struct {
	Component  string `json:"component"`
	Definition string `json:"definition"`
	Mode       string `json:"mode"`
	Status     string `json:"status"`
	Reference  string `json:"reference"`
	InstanceID string `json:"instanceId"`
}

type LiveStatus struct {
	CD        *cd.Status                            `json:"cd,omitempty"`
	Workloads map[string]*kubernetes.WorkloadStatus `json:"workloads,omitempty"`
	Endpoints map[string]string                     `json:"endpoints,omitempty"`
	Note      string                                `json:"note,omitempty"`
}

type DeploymentDetail struct {
	Deployment      *domain.Deployment      `json:"deployment"`
	ApplicationName string                  `json:"applicationName"`
	VersionNumber   int                     `json:"versionNumber"`
	JobStatus       string                  `json:"jobStatus"`
	PlanView        *PlanView               `json:"planView,omitempty"`
	PlanError       string                  `json:"planError,omitempty"`
	Record          *persistence.RecordView `json:"record,omitempty"`
	Steps           []persistence.StepView  `json:"steps"`
	Images          []ImageRow              `json:"images"`
	Removed         []string                `json:"removed"`
	Infrastructure  []InfraRow              `json:"infrastructure"`
	Live            *LiveStatus             `json:"live,omitempty"`
	Duration        string                  `json:"duration"`
}

func (q *QueryService) ListDeployments(ctx context.Context, application string) (*persistence.ApplicationSummary, []persistence.DeploymentSummary, error) {
	app, err := q.Apps.FindApplication(ctx, application)
	if err != nil {
		return nil, nil, err
	}
	list, err := q.Deployments.FindByApplication(ctx, app.ID)
	return app, list, err
}

// GetDeploymentDetail assembles record, progress, images, infrastructure and,
// only where it is truthful, live CD and workload status.
func (q *QueryService) GetDeploymentDetail(ctx context.Context, id string, live bool) (*DeploymentDetail, error) {
	d, err := q.Deployments.FindByIDWithPersistedInputs(ctx, id)
	if err != nil {
		return nil, err
	}
	version, err := q.Apps.FindVersion(ctx, d.ApplicationID, d.VersionID)
	if err != nil {
		return nil, err
	}
	names, err := q.Apps.ComponentNames(ctx, d.ApplicationID)
	if err != nil {
		return nil, err
	}
	out := &DeploymentDetail{Deployment: d, ApplicationName: version.ApplicationName, VersionNumber: version.VersionNumber,
		Images: []ImageRow{}, Removed: []string{}, Infrastructure: []InfraRow{}}
	out.Duration = d.UpdatedAt.Sub(d.CreatedAt).Round(time.Second).String()
	if out.JobStatus, _, err = q.Deployments.JobStatus(ctx, id); err != nil {
		return nil, err
	}
	if d.Status == domain.AwaitingConfirmation {
		if pv, err := q.Orch.GetPlanView(ctx, id); err != nil {
			out.PlanError = err.Error()
		} else {
			out.PlanView = pv
		}
	}
	if out.Steps, err = q.Deployments.GetDeploymentProgress(ctx, id); err != nil {
		return nil, err
	}
	if out.Record, err = q.Deployments.GetDeploymentRecord(ctx, id); err != nil {
		return nil, err
	}

	running, err := q.Workloads.FindWorkloadInstances(ctx, d.ApplicationID, d.Environment, d.Target)
	if err != nil {
		return nil, err
	}
	runningBy := map[string]domain.WorkloadInstance{}
	for _, wi := range running {
		runningBy[wi.WorkloadID] = wi
	}
	for _, wd := range d.WorkloadDeployments {
		row := ImageRow{WorkloadID: wd.WorkloadID, Workload: names[wd.WorkloadID], Image: wd.ImageRef(), InclusionReason: wd.InclusionReason, Wave: wd.WaveNumber}
		if wi, ok := runningBy[wd.WorkloadID]; ok {
			row.RunningHere = wi.CurrentWorkloadDeploymentID == wd.ID
			if !row.RunningHere {
				row.ReplacedBy = wi.DeploymentID
			}
		}
		out.Images = append(out.Images, row)
	}
	sort.Slice(out.Images, func(i, j int) bool {
		if out.Images[i].Wave != out.Images[j].Wave {
			return out.Images[i].Wave < out.Images[j].Wave
		}
		return out.Images[i].Workload < out.Images[j].Workload
	})

	if out.Record != nil {
		for _, cid := range out.Record.RemovedComponents {
			name := names[cid]
			if name == "" {
				name = platformName(d.ApplicationID, cid)
			}
			out.Removed = append(out.Removed, name)
		}
		instances, err := q.Resources.FindByIDs(ctx, out.Record.ResourceInstanceIDs)
		if err != nil {
			return nil, err
		}
		for _, ri := range instances {
			row := InfraRow{Component: names[ri.RequirementID], Status: string(ri.Status), Reference: ri.InfrastructureReference, InstanceID: ri.ID}
			if def, err := q.Catalog.FindByID(ctx, ri.DefinitionID); err == nil {
				row.Definition, row.Mode = def.Name, string(def.ManagementMode)
				if row.Component == "" {
					row.Component = def.ResourceType
				}
			}
			out.Infrastructure = append(out.Infrastructure, row)
		}
		sort.Slice(out.Infrastructure, func(i, j int) bool { return out.Infrastructure[i].Component < out.Infrastructure[j].Component })
	}
	if live {
		out.Live = q.liveStatus(ctx, d, out)
	}
	return out, nil
}

func platformName(applicationID, componentID string) string {
	for _, t := range []string{domain.ResourceTypeCluster, domain.ResourceTypeNetwork} {
		if domain.PlatformRequirementID(applicationID, t) == componentID {
			return t
		}
	}
	return componentID
}

// liveStatus asks CD and Kubernetes only when the answer belongs to this
// deployment: it has a delivery reference and its workloads are the running ones.
func (q *QueryService) liveStatus(ctx context.Context, d *domain.Deployment, detail *DeploymentDetail) *LiveStatus {
	ls := &LiveStatus{}
	if detail.Record == nil || detail.Record.DeliveryReference == "" || d.Kind == domain.KindTeardown {
		ls.Note = "no delivery for this deployment"
		return ls
	}
	var current []ImageRow
	for _, img := range detail.Images {
		if img.RunningHere {
			current = append(current, img)
		}
	}
	if len(current) == 0 {
		ls.Note = "workloads of this deployment are no longer the running ones"
		return ls
	}
	instances, err := q.Resources.FindResourceInstances(ctx, d.ApplicationID, d.Environment, d.Target)
	if err != nil {
		ls.Note = err.Error()
		return ls
	}
	clusterID := domain.PlatformRequirementID(d.ApplicationID, domain.ResourceTypeCluster)
	for i := range instances {
		ri := &instances[i]
		if ri.RequirementID != clusterID || ri.Status != domain.RIReady {
			continue
		}
		def, err := q.Catalog.FindByID(ctx, ri.DefinitionID)
		if err != nil {
			ls.Note = err.Error()
			return ls
		}
		outs, err := q.ResourceOutputs.CollectResourceOutputs(ctx, ri, def)
		if err != nil {
			ls.Note = "cluster outputs unavailable: " + err.Error()
			return ls
		}
		access, err := kubernetes.AccessFromOutputs(outs)
		if err != nil {
			ls.Note = err.Error()
			return ls
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		ns := Namespace(detail.ApplicationName, d.Environment)
		ds := cd.DesiredState{Cluster: access, Target: d.Target, Application: detail.ApplicationName, Environment: string(d.Environment), Namespace: ns}
		if st, err := q.CD.GetCDStatus(ctx, ds); err == nil {
			ls.CD = st
		} else {
			ls.Note = "CD status unavailable: " + err.Error()
		}
		ls.Workloads, ls.Endpoints = map[string]*kubernetes.WorkloadStatus{}, map[string]string{}
		for _, img := range current {
			if st, err := q.Kube.GetWorkloadStatus(ctx, access, ns, img.WorkloadID); err == nil && st != nil {
				ls.Workloads[img.Workload] = st
			}
			if outs, err := q.Kube.ReadWorkloadOutputs(ctx, access, ns, img.WorkloadID, []string{"endpoint"}); err == nil {
				ls.Endpoints[img.Workload] = outs["endpoint"].Value
			}
		}
		return ls
	}
	ls.Note = "the k8s-cluster of this environment and target is not ready"
	return ls
}

// ProblemsOf extracts A1 problems for API and UI responses.
func ProblemsOf(err error) []domain.Problem {
	if v, ok := err.(*domain.ValidationError); ok {
		return v.Problems
	}
	return []domain.Problem{{Code: "ERROR", Message: strings.TrimSpace(err.Error())}}
}
