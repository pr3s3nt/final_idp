// Package infraplanner is the Infrastructure Planner: it compares the graph
// with current Resource and Workload Instances (found by owner key) to decide
// create/update/reuse/link and, for a new version or a teardown, what to
// remove, destroy or unlink. It also owns the override policy.
package infraplanner

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceresolver"
)

type Input struct {
	Kind          domain.DeploymentKind
	Graph         *domain.DeploymentGraph
	Waves         *domain.WavePlan // nil for TEARDOWN
	Environment   domain.Environment
	Images        map[string]string // workload ID -> image version (selected workloads)
	Instances     []domain.ResourceInstance
	Running       []domain.WorkloadInstance
	Configuration *domain.EnvironmentConfiguration
	Resolver      *resourceresolver.Resolver // to look up definitions of instances being removed
}

// PlanInfrastructureChanges builds the transient plan whose fingerprint guards
// confirmation and execution.
func PlanInfrastructureChanges(in Input) (*domain.Plan, error) {
	g := in.Graph
	v := g.Version
	p := &domain.Plan{
		Kind: in.Kind, ApplicationID: v.ApplicationID, ApplicationName: v.ApplicationName,
		VersionID: v.VersionID, VersionNumber: v.VersionNumber, Environment: in.Environment,
		Target: g.Context.Target, Context: g.Context, Waves: []domain.PlanWave{}, Removals: []domain.PlanWave{},
		PotentialRedeploy: []domain.PlanComponent{},
	}
	problems := &domain.ValidationError{}

	instanceByOwner := map[string]*domain.ResourceInstance{}
	for i := range in.Instances {
		instanceByOwner[in.Instances[i].RequirementID] = &in.Instances[i]
	}

	if in.Kind == domain.KindDeploy {
		p.FullDeployment = in.Waves.FullDeployment
		for wi, ids := range in.Waves.Waves {
			wave := domain.PlanWave{Number: wi, Items: []domain.PlanItem{}}
			for _, id := range ids {
				n := g.Nodes[id]
				item := domain.PlanItem{ComponentID: id, Name: n.Name, Kind: n.Kind, DependsOn: nonNil(n.DependsOn)}
				if n.Kind == domain.NodeWorkload {
					w := v.Workload(id)
					item.Action, item.ImageRepository, item.ImageVersion, item.InclusionReason = domain.ActionDeploy, w.ImageRepository, in.Images[id], domain.Selected
				} else if err := planResource(&item, n, g.Resolutions[id], instanceByOwner[id]); err != nil {
					problems.Problems = append(problems.Problems, *err)
				}
				wave.Items = append(wave.Items, item)
			}
			p.Waves = append(p.Waves, wave)
		}
		for _, id := range in.Waves.PotentialRedeploy {
			p.PotentialRedeploy = append(p.PotentialRedeploy, domain.PlanComponent{ID: id, Name: g.Nodes[id].Name, Kind: domain.NodeWorkload})
		}
		p.ConfigurationDigest = configurationDigest(in.Configuration, in.Waves.Scope)
	}

	// Removals: a full deployment of a version removes running components the
	// version no longer contains; a teardown removes everything.
	if in.Kind == domain.KindTeardown || in.Waves.FullDeployment {
		removals, err := planRemovals(in, instanceByOwner)
		if err != nil {
			problems.Problems = append(problems.Problems, *err)
		}
		p.Removals = removals
	}
	if err := problems.OrNil(); err != nil {
		return nil, err
	}
	return p, nil
}

func planResource(item *domain.PlanItem, n *domain.GraphNode, res *domain.ResourceResolution, current *domain.ResourceInstance) *domain.Problem {
	d := res.Definition
	item.ResourceType, item.Platform = n.ResourceType, n.Platform
	item.DefinitionID, item.DefinitionName, item.ManagementMode = d.ID, d.Name, d.ManagementMode
	item.ProvisionerReference, item.ExistingReference = d.ProvisionerReference, d.ExistingResourceReference
	item.DefinitionDigest = DefinitionDigest(d)
	item.AllowedOverrides = d.AllowedOverrides

	baseline := map[string]any{}
	if current != nil {
		item.ResourceInstanceID = current.ID
		baseline = current.AppliedOverrides
		if def := current.DefinitionID; def != d.ID {
			return &domain.Problem{Code: domain.CodeDefinitionChanged, Message: fmt.Sprintf(
				"%s is running on resource definition %s but now resolves to %s; remove the application from this environment and target first",
				n.Name, def, d.Name)}
		}
	}
	item.Parameters = Merge(d.DefaultParameters, baseline)

	switch d.ManagementMode {
	case domain.Existing:
		if current == nil {
			item.Action = domain.ActionLink
		} else {
			item.Action = domain.ActionReuse
		}
	case domain.Managed:
		switch {
		case current == nil:
			item.Action = domain.ActionCreate
		case current.Status != domain.RIReady:
			item.Action = domain.ActionUpdate // retry a failed or interrupted apply
		case current.AppliedInputFingerprint != InputFingerprint(d, item.Parameters):
			item.Action = domain.ActionUpdate
		default:
			item.Action = domain.ActionReuse
		}
	}
	return nil
}

// planRemovals orders removals so dependents go first: workloads, then
// resources that require nothing still being removed after them.
func planRemovals(in Input, instanceByOwner map[string]*domain.ResourceInstance) ([]domain.PlanWave, *domain.Problem) {
	g := in.Graph
	var removeWorkloads []domain.PlanItem
	for _, wi := range in.Running {
		if in.Kind == domain.KindTeardown || g.Version.Workload(wi.WorkloadID) == nil {
			removeWorkloads = append(removeWorkloads, domain.PlanItem{
				ComponentID: wi.WorkloadID, Name: workloadName(in, wi), Kind: domain.NodeWorkload, Action: domain.ActionRemove,
				DependsOn: []string{}, ImageRepository: wi.ImageRepository, ImageVersion: wi.ImageVersion,
			})
		}
	}
	sort.Slice(removeWorkloads, func(i, j int) bool { return removeWorkloads[i].Name < removeWorkloads[j].Name })

	type removal struct {
		item     domain.PlanItem
		requires []string
		rtype    string
	}
	var resources []removal
	for _, ri := range in.Instances {
		if in.Kind != domain.KindTeardown {
			if _, stillInGraph := g.Nodes[ri.RequirementID]; stillInGraph {
				continue
			}
		}
		d := in.Resolver.DefinitionByID(ri.DefinitionID)
		if d == nil {
			return nil, &domain.Problem{Code: domain.CodeNoResourceDefinition, Message: fmt.Sprintf("resource instance %s uses definition %s, which is no longer in the catalog", ri.ID, ri.DefinitionID)}
		}
		action, loss := domain.ActionDestroy, true
		if d.ManagementMode == domain.Existing {
			action, loss = domain.ActionUnlink, false
		}
		resources = append(resources, removal{
			item: domain.PlanItem{
				ComponentID: ri.RequirementID, Name: resourceName(in, ri, d), Kind: domain.NodeResource, Action: action,
				DependsOn: []string{}, ResourceType: d.ResourceType, DefinitionID: d.ID, DefinitionName: d.Name,
				DefinitionDigest: DefinitionDigest(d), ManagementMode: d.ManagementMode, ProvisionerReference: d.ProvisionerReference,
				ExistingReference: d.ExistingResourceReference, ResourceInstanceID: ri.ID, DataLossWarning: loss,
				Parameters: Merge(d.DefaultParameters, ri.AppliedOverrides), Platform: isPlatform(ri, d),
			},
			requires: d.Requires, rtype: d.ResourceType,
		})
	}

	// Removal waves over resources: a resource can be destroyed once nothing
	// still present requires its type.
	var waves []domain.PlanWave
	if len(removeWorkloads) > 0 {
		waves = append(waves, domain.PlanWave{Items: removeWorkloads})
	}
	remaining := resources
	for len(remaining) > 0 {
		var now, later []removal
		for _, r := range remaining {
			blocked := false
			for _, other := range remaining {
				for _, req := range other.requires {
					if req == r.rtype && other.item.ComponentID != r.item.ComponentID {
						blocked = true
					}
				}
			}
			if blocked {
				later = append(later, r)
			} else {
				now = append(now, r)
			}
		}
		if len(now) == 0 { // cyclic requires in the catalog; remove the rest together
			now, later = later, nil
		}
		wave := domain.PlanWave{}
		for _, r := range now {
			wave.Items = append(wave.Items, r.item)
		}
		sort.Slice(wave.Items, func(i, j int) bool { return wave.Items[i].Name < wave.Items[j].Name })
		waves = append(waves, wave)
		remaining = later
	}
	for i := range waves {
		waves[i].Number = i
	}
	if waves == nil {
		waves = []domain.PlanWave{}
	}
	return waves, nil
}

func isPlatform(ri domain.ResourceInstance, d *domain.ResourceDefinition) bool {
	return ri.RequirementID == domain.PlatformRequirementID(ri.ApplicationID, d.ResourceType)
}

func workloadName(in Input, wi domain.WorkloadInstance) string {
	if n, ok := in.Graph.Nodes[wi.WorkloadID]; ok {
		return n.Name
	}
	if name, ok := in.Graph.ComponentNames[wi.WorkloadID]; ok {
		return name
	}
	return "workload " + wi.WorkloadID[:8]
}

func resourceName(in Input, ri domain.ResourceInstance, d *domain.ResourceDefinition) string {
	if n, ok := in.Graph.Nodes[ri.RequirementID]; ok {
		return n.Name
	}
	if isPlatform(ri, d) {
		return d.ResourceType
	}
	if name, ok := in.Graph.ComponentNames[ri.RequirementID]; ok {
		return name
	}
	return d.ResourceType + " " + ri.RequirementID[:8]
}

// DefinitionDigest fingerprints every plan-affecting field of a definition.
func DefinitionDigest(d *domain.ResourceDefinition) string {
	return domain.HashCanonical(map[string]any{
		"provisionerReference": d.ProvisionerReference, "supportedContexts": d.SupportedContexts,
		"defaultParameters": d.DefaultParameters, "allowedOverrides": d.AllowedOverrides,
		"managementMode": d.ManagementMode, "existingResourceReference": d.ExistingResourceReference,
		"applicabilityConditions": d.ApplicabilityConditions, "requires": d.Requires,
	})
}

// InputFingerprint identifies what was last applied to a managed instance.
func InputFingerprint(d *domain.ResourceDefinition, parameters map[string]any) string {
	return domain.HashCanonical(map[string]any{"definitionId": d.ID, "provisionerReference": d.ProvisionerReference, "parameters": parameters})
}

func configurationDigest(c *domain.EnvironmentConfiguration, scope map[string]bool) string {
	if c == nil {
		return ""
	}
	type b struct{ Workload, Name, Source, Ref, Output, ValueDigest string }
	var bindings []b
	for _, v := range append(append([]domain.ConfiguredValue{}, c.Variables...), c.Secrets...) {
		if !scope[v.WorkloadID] {
			continue
		}
		digest := ""
		if v.Source == domain.SourceDirect {
			digest = domain.HashCanonical(v.DirectValue)
		}
		bindings = append(bindings, b{v.WorkloadID, v.Name, v.Source, v.RefID + v.SecretRef, v.OutputName, digest})
	}
	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].Workload+bindings[i].Name < bindings[j].Workload+bindings[j].Name
	})
	return domain.HashCanonical(bindings)
}

// Merge returns base overlaid with each overlay, without mutating the inputs.
func Merge(base map[string]any, overlays ...map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for _, o := range overlays {
		for k, v := range o {
			out[k] = v
		}
	}
	return out
}

// ApplyInfrastructureOverrides validates Developer override values against each
// definition's allowed_overrides and computes final parameters and the new
// override baseline. overrides is keyed by component ID or component name.
func ApplyInfrastructureOverrides(p *domain.Plan, overrides map[string]map[string]any) error {
	problems := &domain.ValidationError{}
	items := map[string]*domain.PlanItem{}
	for _, it := range p.Items() {
		if it.Kind == domain.NodeResource {
			items[it.ComponentID], items[it.Name] = it, it
			it.FinalParameters = Merge(it.Parameters)
			it.NewBaseline = baselineOf(it)
		}
	}
	for _, key := range sortedMapKeys(overrides) {
		it, ok := items[key]
		if !ok {
			problems.Add(domain.CodeInvalidOverride, "no resource %q in this deployment plan accepts overrides", key)
			continue
		}
		for _, param := range sortedMapKeys(overrides[key]) {
			value := overrides[key][param]
			rule, ok := it.AllowedOverrides[param]
			if !ok {
				problems.Add(domain.CodeInvalidOverride, "%s: parameter %q may not be overridden", it.Name, param)
				continue
			}
			normalized, err := validateValue(rule, value)
			if err != nil {
				problems.Add(domain.CodeInvalidOverride, "%s.%s: %v", it.Name, param, err)
				continue
			}
			current, hasCurrent := it.Parameters[param]
			if equalValues(current, normalized) {
				continue
			}
			if rule.Immutable && it.ResourceInstanceID != "" && hasCurrent {
				problems.Add(domain.CodeImmutableParameter, "%s.%s cannot change after the resource was created (current %v)", it.Name, param, current)
				continue
			}
			if rule.IncreaseOnly && hasCurrent && it.ResourceInstanceID != "" {
				if msg := checkIncrease(rule, current, normalized); msg != "" {
					problems.Add(domain.CodeInvalidOverride, "%s.%s: %s", it.Name, param, msg)
					continue
				}
			}
			it.FinalParameters[param] = normalized
			it.NewBaseline[param] = normalized
		}
	}
	if err := problems.OrNil(); err != nil {
		return err
	}
	for _, it := range items {
		if it.Action == domain.ActionReuse && it.ManagementMode == domain.Managed && !equalValues(it.FinalParameters, it.Parameters) {
			it.Action = domain.ActionUpdate
		}
	}
	return nil
}

func baselineOf(it *domain.PlanItem) map[string]any {
	out := map[string]any{}
	for k := range it.AllowedOverrides {
		if v, ok := it.Parameters[k]; ok {
			out[k] = v
		}
	}
	return out
}

func validateValue(rule domain.OverrideRule, value any) (any, error) {
	switch rule.Type {
	case "integer", "number":
		f, ok := toFloat(value)
		if !ok {
			return nil, fmt.Errorf("must be a number")
		}
		if rule.Type == "integer" && f != math.Trunc(f) {
			return nil, fmt.Errorf("must be an integer")
		}
		if rule.Min != nil && f < *rule.Min {
			return nil, fmt.Errorf("must be >= %v", *rule.Min)
		}
		if rule.Max != nil && f > *rule.Max {
			return nil, fmt.Errorf("must be <= %v", *rule.Max)
		}
		if rule.Type == "integer" {
			return int64(f), nil
		}
		return f, nil
	case "enum":
		s := fmt.Sprint(value)
		for _, e := range rule.Enum {
			if e == s {
				return s, nil
			}
		}
		return nil, fmt.Errorf("must be one of %v", rule.Enum)
	case "cidr":
		s := fmt.Sprint(value)
		ip, network, err := net.ParseCIDR(s)
		if err != nil || !ip.Equal(network.IP) {
			return nil, fmt.Errorf("must be a network CIDR such as 10.70.0.0/16")
		}
		if ones, _ := network.Mask.Size(); rule.CIDRPrefixLen != nil && ones != *rule.CIDRPrefixLen {
			return nil, fmt.Errorf("prefix length must be /%d", *rule.CIDRPrefixLen)
		}
		if rule.CIDRWithin != "" {
			_, within, err := net.ParseCIDR(rule.CIDRWithin)
			if err != nil || !within.Contains(network.IP) {
				return nil, fmt.Errorf("must be inside %s", rule.CIDRWithin)
			}
		}
		return network.String(), nil
	case "string", "":
		return fmt.Sprint(value), nil
	}
	return nil, fmt.Errorf("unsupported override type %q", rule.Type)
}

func checkIncrease(rule domain.OverrideRule, current, next any) string {
	if rule.Type == "enum" {
		ci, ni := indexOf(rule.Enum, fmt.Sprint(current)), indexOf(rule.Enum, fmt.Sprint(next))
		if ni < ci {
			return fmt.Sprintf("can only increase (current %v)", current)
		}
		if rule.MaxStep != nil && float64(ni-ci) > *rule.MaxStep {
			return fmt.Sprintf("can increase by at most %v step(s) from %v", *rule.MaxStep, current)
		}
		return ""
	}
	c, ok1 := toFloat(current)
	n, ok2 := toFloat(next)
	if ok1 && ok2 && n < c {
		return fmt.Sprintf("can only increase (current %v)", current)
	}
	return ""
}

func indexOf(list []string, s string) int {
	for i, e := range list {
		if e == s {
			return i
		}
	}
	return -1
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	}
	return 0, false
}

func equalValues(a, b any) bool {
	return string(domain.CanonicalJSON(a)) == string(domain.CanonicalJSON(b))
}

func sortedMapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
