// Package appvalidator is the UC-01 Application Definition Validator. It checks
// one complete submitted definition in memory (A1); checks that need stored
// state, such as base version and component ownership, belong to the
// Application Repository.
package appvalidator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"idp/internal/domain"
)

// Definition is the content of one Application Definition Draft to validate.
type Definition struct {
	Name         string
	Description  string
	Workloads    []domain.Workload
	Resources    []domain.ResourceRequirement
	Dependencies []Dependency
}

// Dependency is a submitted depends-on relation between two component IDs.
// Key identifies it in field paths; it is not a stable ID.
type Dependency struct {
	Key      string
	SourceID string
	TargetID string
}

const (
	maxComponentName   = 63
	maxDefinitionName  = 255
	maxType            = 100
	maxImageRepository = 1024
)

var (
	// UC-01 naming rules (specification: Quy tắc đặt tên).
	componentName  = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	definitionName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

	// Image repository without tag or digest, following the distribution
	// reference grammar: [host[:port]/]path-component[/path-component...].
	imageRepository = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])(?:\.(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]))*(?::[0-9]+)?/)?[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$`)
)

// Validate returns every problem found, or nil. It also resolves the target
// type of each dependency for persistence.
func Validate(d Definition) ([]domain.Dependency, error) {
	v := &domain.ValidationError{}

	checkComponentName(v, "name", "Application name", d.Name)

	if len(d.Workloads) == 0 {
		v.AddField("workloads", domain.CodeNoWorkload, "an application needs at least one workload")
	}

	ids := map[string]bool{}
	checkID := func(field, label, id string) {
		switch {
		case strings.TrimSpace(id) == "":
			v.AddField(field, domain.CodeInvalidComponentID, "%s has no ID", label)
		case uuid.Validate(id) != nil:
			v.AddField(field, domain.CodeInvalidComponentID, "%s ID %q is not a UUID", label, id)
		case ids[strings.ToLower(id)]:
			v.AddField(field, domain.CodeInvalidComponentID, "%s ID %q is used by another component", label, id)
		}
		ids[strings.ToLower(id)] = true
	}

	names := map[string]string{} // workload and resource names share one namespace
	kinds := map[string]string{} // component ID -> WORKLOAD | RESOURCE
	nameOf := map[string]string{}
	checkUnique := func(field, label, name string) {
		if name == "" {
			return
		}
		if other, dup := names[name]; dup {
			v.AddField(field, domain.CodeDuplicateName, "%s name %q is already used by %s", label, name, other)
			return
		}
		names[name] = strings.ToLower(label) + " " + name
	}

	for _, w := range d.Workloads {
		base := "workloads." + w.ID
		label := "Workload"
		if w.Name != "" {
			label = fmt.Sprintf("Workload %q", w.Name)
		}
		checkID(base+".id", label, w.ID)
		checkComponentName(v, base+".name", "Workload name", w.Name)
		checkUnique(base+".name", "Workload", w.Name)
		checkType(v, base+".type", "Workload type", w.Type)
		checkImageRepository(v, base+".imageRepository", w.ImageRepository)
		if w.Port != nil && (*w.Port < 1 || *w.Port > 65535) {
			v.AddField(base+".port", domain.CodeInvalidPort, "port %d is outside 1..65535", *w.Port)
		}
		outputs := map[string]bool{}
		for i, o := range w.ExposedOutputs {
			field := fmt.Sprintf("%s.outputs.%d", base, i)
			checkComponentName(v, field, "Output name", o)
			if o != "" && outputs[o] {
				v.AddField(field, domain.CodeDuplicateName, "%s declares output %q twice", label, o)
			}
			outputs[o] = true
		}
		// Environment Variables and Secrets of one workload share one namespace.
		configNames := map[string]string{}
		checkDefinitions := func(group, kind string, defs []domain.ConfigDefinition) {
			for _, c := range defs {
				field := base + "." + group + "." + c.ID
				checkID(field+".id", kind, c.ID)
				checkDefinitionName(v, field+".name", kind+" name", c.Name)
				if c.Name == "" {
					continue
				}
				if other, dup := configNames[c.Name]; dup {
					v.AddField(field+".name", domain.CodeDuplicateName, "%s already declares %s %q", label, other, c.Name)
					continue
				}
				configNames[c.Name] = strings.ToLower(kind)
			}
		}
		checkDefinitions("variables", "Environment Variable", w.Variables)
		checkDefinitions("secrets", "Secret", w.Secrets)
		kinds[w.ID], nameOf[w.ID] = domain.TargetWorkload, w.Name
	}

	for _, r := range d.Resources {
		base := "resources." + r.ID
		label := "Resource"
		if r.Name != "" {
			label = fmt.Sprintf("Resource %q", r.Name)
		}
		checkID(base+".id", label, r.ID)
		checkComponentName(v, base+".name", "Resource name", r.Name)
		checkUnique(base+".name", "Resource", r.Name)
		checkType(v, base+".type", "Resource type", r.ResourceType)
		kinds[r.ID], nameOf[r.ID] = domain.TargetResource, r.Name
	}

	var resolved []domain.Dependency
	seen := map[string]bool{}
	edges := map[string][]string{} // workload -> workload
	for i, dep := range d.Dependencies {
		key := dep.Key
		if key == "" {
			key = fmt.Sprint(i)
		}
		field := "dependencies." + key
		srcKind, srcOK := kinds[dep.SourceID]
		tgtKind, tgtOK := kinds[dep.TargetID]
		switch {
		case dep.SourceID == "" || dep.TargetID == "":
			v.AddField(field, domain.CodeMissingRequiredField, "a dependency needs a source and a target")
			continue
		case !srcOK:
			v.AddField(field, domain.CodeDependencyUnresolved, "dependency source %q does not exist in the application", dep.SourceID)
			continue
		case srcKind != domain.TargetWorkload:
			v.AddField(field, domain.CodeInvalidDependency, "dependency source %q is a resource; only a workload can depend on another component", nameOf[dep.SourceID])
			continue
		case !tgtOK:
			v.AddField(field, domain.CodeDependencyUnresolved, "dependency target %q does not exist in the application", dep.TargetID)
			continue
		case dep.SourceID == dep.TargetID:
			v.AddField(field, domain.CodeDependencyCycle, "workload %q cannot depend on itself", nameOf[dep.SourceID])
			continue
		}
		pair := dep.SourceID + ">" + dep.TargetID
		if seen[pair] {
			v.AddField(field, domain.CodeDuplicateDependency, "%q already depends on %q", nameOf[dep.SourceID], nameOf[dep.TargetID])
			continue
		}
		seen[pair] = true
		if tgtKind == domain.TargetWorkload {
			edges[dep.SourceID] = append(edges[dep.SourceID], dep.TargetID)
		}
		resolved = append(resolved, domain.Dependency{SourceWorkloadID: dep.SourceID, TargetType: tgtKind, TargetID: dep.TargetID})
	}
	if cycle := findCycle(d.Workloads, edges); cycle != nil {
		path := make([]string, len(cycle))
		for i, id := range cycle {
			path[i] = nameOf[id]
		}
		v.AddField("dependencies", domain.CodeDependencyCycle, "depends-on relations form a cycle: %s", strings.Join(path, " → "))
	}

	if err := v.OrNil(); err != nil {
		return nil, err
	}
	return resolved, nil
}

func checkComponentName(v *domain.ValidationError, field, label, name string) {
	switch {
	case name == "":
		v.AddField(field, domain.CodeMissingRequiredField, "%s is required", label)
	case len(name) > maxComponentName || !componentName.MatchString(name):
		v.AddField(field, domain.CodeInvalidName, "%s %q must use lowercase letters, digits and '-', start and end with a letter or digit, and have at most %d characters", label, name, maxComponentName)
	}
}

func checkDefinitionName(v *domain.ValidationError, field, label, name string) {
	switch {
	case name == "":
		v.AddField(field, domain.CodeMissingRequiredField, "%s is required", label)
	case len(name) > maxDefinitionName || !definitionName.MatchString(name):
		v.AddField(field, domain.CodeInvalidName, "%s %q must use letters, digits and '_' and must not start with a digit", label, name)
	}
}

func checkType(v *domain.ValidationError, field, label, value string) {
	switch {
	case strings.TrimSpace(value) == "":
		v.AddField(field, domain.CodeMissingRequiredField, "%s is required", label)
	case len(value) > maxType || strings.TrimSpace(value) != value:
		v.AddField(field, domain.CodeInvalidInput, "%s must have at most %d characters and no leading or trailing spaces", label, maxType)
	}
}

func checkImageRepository(v *domain.ValidationError, field, repo string) {
	switch {
	case repo == "":
		v.AddField(field, domain.CodeMissingRequiredField, "Image repository is required")
	case strings.Contains(repo, "@") || strings.Contains(repo[strings.LastIndex(repo, "/")+1:], ":"):
		v.AddField(field, domain.CodeInvalidImageRepository, "image repository %q must not include a tag or digest; the image version is chosen at deployment (UC-03)", repo)
	case len(repo) > maxImageRepository || !imageRepository.MatchString(repo):
		v.AddField(field, domain.CodeInvalidImageRepository, "image repository %q is not a valid repository reference, for example registry.company.local/shop-backend", repo)
	}
}

// findCycle returns one workload cycle (first node repeated at the end), or nil.
func findCycle(workloads []domain.Workload, edges map[string][]string) []string {
	const (
		unvisited = iota
		active
		done
	)
	state := map[string]int{}
	var stack []string
	var visit func(id string) []string
	visit = func(id string) []string {
		state[id] = active
		stack = append(stack, id)
		for _, next := range edges[id] {
			switch state[next] {
			case active:
				for i, s := range stack {
					if s == next {
						return append(append([]string{}, stack[i:]...), next)
					}
				}
			case unvisited:
				if c := visit(next); c != nil {
					return c
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = done
		return nil
	}
	for _, w := range workloads {
		if state[w.ID] == unvisited {
			if c := visit(w.ID); c != nil {
				return c
			}
		}
	}
	return nil
}
