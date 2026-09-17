// Package targetadapter is the Target Manifest Adapter: target-specific changes
// applied after score-k8s renders the base manifest.
package targetadapter

import (
	"fmt"
	"strings"

	"github.com/pr3s3nt/final_idp/idp/backend/internal/domain"
)

// LogicalRegistryHost is the registry host used in Application Definitions
// (UC-01 examples). Each target maps it to the registry its cluster pulls from.
const LogicalRegistryHost = "registry.company.local"

// RegistryMirror returns the registry the target cluster pulls images from,
// taken from the resolved k8s-cluster definition's parameters.
func RegistryMirror(g *domain.DeploymentGraph) string {
	res, ok := g.Resolutions[domain.PlatformRequirementID(g.Version.ApplicationID, domain.ResourceTypeCluster)]
	if !ok {
		return ""
	}
	return fmt.Sprint(res.Definition.DefaultParameters["image_registry_mirror"])
}

// ResolveImage maps a logical image repository and tag to the reference the
// target cluster pulls. Repositories on other hosts are used unchanged.
func ResolveImage(repository, tag, mirror string) string {
	repo := repository
	if mirror != "" && mirror != "<nil>" {
		if rest, ok := strings.CutPrefix(repository, LogicalRegistryHost+"/"); ok {
			repo = strings.TrimSuffix(mirror, "/") + "/" + rest
		}
	}
	return repo + ":" + tag
}
