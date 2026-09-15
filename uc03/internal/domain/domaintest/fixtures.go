// Package domaintest builds in-memory domain fixtures for unit tests.
package domaintest

import (
	"github.com/google/uuid"

	"github.com/pr3s3nt/final_idp/uc03/internal/domain"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceresolver"
)

func f(v float64) *float64 { return &v }

// CatalogV1 is the catalog version of Catalog().
var CatalogV1 = domain.CatalogVersion{ID: "cat-1", Number: 1}

// Catalog mirrors fixtures/catalog/v1.yaml.
func Catalog() *resourceresolver.Resolver {
	defs := catalogDefinitions()
	for i := range defs {
		defs[i].CatalogVersionID = CatalogV1.ID
	}
	return &resourceresolver.Resolver{Definitions: defs, All: defs}
}

func catalogDefinitions() []domain.ResourceDefinition {
	local := []map[string]string{{"target": "kind-local", "cloudProvider": "local"}}
	aws := []map[string]string{{"target": "aws", "cloudProvider": "aws", "region": "ap-southeast-1"}}
	prefix16 := 16
	return []domain.ResourceDefinition{
		{ID: "d-kind", Name: "kind-internal-cluster", ResourceType: "k8s-cluster", ManagementMode: domain.Existing, ProvisionerReference: "none://existing",
			ExistingResourceReference: "idpsecret://platform/kind-internal-cluster", SupportedContexts: local,
			DefaultParameters: map[string]any{"image_registry_mirror": "localhost:5055"},
			ExposedOutputs:    []string{"cluster_name", "image_registry_mirror", "cluster_kind"}, SensitiveOutputs: []string{"kubeconfig"}},
		{ID: "d-pg-k8s", Name: "postgres-k8s", ResourceType: "PostgreSQL", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/postgres-k8s",
			SupportedContexts: local, DefaultParameters: map[string]any{"storage_gb": 1.0, "password_revision": 0.0},
			AllowedOverrides: map[string]domain.OverrideRule{
				"storage_gb":        {Type: "integer", Min: f(1), Max: f(5), IncreaseOnly: true},
				"password_revision": {Type: "integer", Min: f(0), Max: f(1000), IncreaseOnly: true}},
			ExposedOutputs: []string{"host", "port", "database", "username"}, SensitiveOutputs: []string{"password"}, Requires: []string{"k8s-cluster"}},
		{ID: "d-redis-k8s", Name: "redis-k8s", ResourceType: "Redis", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/redis-k8s",
			SupportedContexts: local, DefaultParameters: map[string]any{}, ExposedOutputs: []string{"host", "port"}, Requires: []string{"k8s-cluster"}},
		{ID: "d-pg-shared", Name: "postgres-shared-staging", ResourceType: "PostgreSQL", ManagementMode: domain.Existing,
			ProvisionerReference: "none://existing", ExistingResourceReference: "idpsecret://platform/shared-postgres-staging",
			SupportedContexts: local, ApplicabilityConditions: map[string][]string{"application": {"reporting-app"}, "environment": {"STAGING"}},
			DefaultParameters: map[string]any{}, ExposedOutputs: []string{"host", "port", "database", "username"}, SensitiveOutputs: []string{"password"}},
		{ID: "d-net", Name: "aws-network", ResourceType: "network", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/aws-network",
			SupportedContexts: aws, DefaultParameters: map[string]any{"vpc_cidr": "10.60.0.0/16", "az_count": 2.0},
			AllowedOverrides: map[string]domain.OverrideRule{"vpc_cidr": {Type: "cidr", CIDRPrefixLen: &prefix16, CIDRWithin: "10.0.0.0/8", Immutable: true}},
			ExposedOutputs:   []string{"vpc_id"}},
		{ID: "d-eks", Name: "eks-cluster", ResourceType: "k8s-cluster", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/eks-cluster",
			SupportedContexts: aws, DefaultParameters: map[string]any{"kubernetes_version": "1.36", "node_count": 2.0},
			AllowedOverrides: map[string]domain.OverrideRule{
				"kubernetes_version": {Type: "enum", Enum: []string{"1.34", "1.35", "1.36"}, IncreaseOnly: true, MaxStep: f(1)},
				"node_count":         {Type: "integer", Min: f(1), Max: f(3)}},
			ExposedOutputs: []string{"cluster_name", "endpoint"}, Requires: []string{"network"}},
		{ID: "d-aurora", Name: "aurora-postgresql", ResourceType: "PostgreSQL", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/aurora-postgresql",
			SupportedContexts: aws, DefaultParameters: map[string]any{"min_acu": 0.5, "max_acu": 2.0},
			ExposedOutputs: []string{"host", "port", "database", "username"}, SensitiveOutputs: []string{"password"}, Requires: []string{"network"}},
		{ID: "d-elasticache", Name: "redis-elasticache", ResourceType: "Redis", ManagementMode: domain.Managed, ProvisionerReference: "terraform://modules/redis-elasticache",
			SupportedContexts: aws, DefaultParameters: map[string]any{}, ExposedOutputs: []string{"host", "port"}, Requires: []string{"network"}},
	}
}

// IDs are stable component IDs shared by every version of shop-app.
var (
	AppID      = "5d2c1a64-6f1b-4f36-9c1e-3c5c4a8f0a01"
	Backend    = "00000000-0000-0000-0000-00000000b001"
	Worker     = "00000000-0000-0000-0000-00000000b002"
	Frontend   = "00000000-0000-0000-0000-00000000b003"
	Postgres   = "00000000-0000-0000-0000-00000000c001"
	Redis      = "00000000-0000-0000-0000-00000000c002"
	V1         = "00000000-0000-0000-0000-0000000000a1"
	V2         = "00000000-0000-0000-0000-0000000000a2"
	ClusterID  = domain.PlatformRequirementID(AppID, "k8s-cluster")
	NetworkID  = domain.PlatformRequirementID(AppID, "network")
	KindLocal  = domain.DeploymentContext{Target: "kind-local", CloudProvider: "local"}
	AWS        = domain.DeploymentContext{Target: "aws", CloudProvider: "aws", Region: "ap-southeast-1"}
	portBack   = 8080
	portWorker = 8081
	portFront  = 3000
)

func def(name string) domain.ConfigDefinition {
	return domain.ConfigDefinition{ID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(name)).String(), Name: name, Required: true}
}

// ShopV1 is frontend -> backend -> postgresql; worker -> redis, postgresql.
func ShopV1() *domain.ApplicationVersion {
	return &domain.ApplicationVersion{
		ApplicationID: AppID, ApplicationName: "shop-app", VersionID: V1, VersionNumber: 1,
		Resources: []domain.ResourceRequirement{{ID: Postgres, Name: "postgresql", ResourceType: "PostgreSQL"}, {ID: Redis, Name: "redis", ResourceType: "Redis"}},
		Workloads: []domain.Workload{
			{ID: Backend, Name: "backend", ImageRepository: "registry.company.local/shop-backend", Port: &portBack, ExposedOutputs: []string{"endpoint"},
				Variables: []domain.ConfigDefinition{def("backend.DB_HOST")}, Secrets: []domain.ConfigDefinition{def("backend.DB_PASSWORD")}},
			{ID: Worker, Name: "worker", ImageRepository: "registry.company.local/shop-worker", Port: &portWorker,
				Variables: []domain.ConfigDefinition{def("worker.REDIS_HOST")}},
			{ID: Frontend, Name: "frontend", ImageRepository: "registry.company.local/shop-frontend", Port: &portFront, ExposedOutputs: []string{"endpoint"},
				Variables: []domain.ConfigDefinition{def("frontend.BACKEND_URL")}},
		},
		Dependencies: []domain.Dependency{
			{ID: "dep1", SourceWorkloadID: Backend, TargetType: domain.TargetResource, TargetID: Postgres},
			{ID: "dep2", SourceWorkloadID: Worker, TargetType: domain.TargetResource, TargetID: Redis},
			{ID: "dep3", SourceWorkloadID: Worker, TargetType: domain.TargetResource, TargetID: Postgres},
			{ID: "dep4", SourceWorkloadID: Frontend, TargetType: domain.TargetWorkload, TargetID: Backend},
		},
	}
}

// ShopV2 drops worker and redis.
func ShopV2() *domain.ApplicationVersion {
	v := ShopV1()
	v.VersionID, v.VersionNumber = V2, 2
	v.Resources = v.Resources[:1]
	v.Workloads = []domain.Workload{v.Workloads[0], v.Workloads[2]}
	v.Dependencies = []domain.Dependency{v.Dependencies[0], v.Dependencies[3]}
	return v
}

// Config binds every required definition of ShopV1.
func Config() *domain.EnvironmentConfiguration {
	return &domain.EnvironmentConfiguration{ID: "cfg", ApplicationID: AppID, Environment: domain.Staging,
		Variables: []domain.ConfiguredValue{
			{ID: "b1", WorkloadID: Backend, DefinitionID: def("backend.DB_HOST").ID, Name: "DB_HOST", Source: domain.SourceResourceOutput, RefID: Postgres, OutputName: "host"},
			{ID: "b2", WorkloadID: Worker, DefinitionID: def("worker.REDIS_HOST").ID, Name: "REDIS_HOST", Source: domain.SourceResourceOutput, RefID: Redis, OutputName: "host"},
			{ID: "b3", WorkloadID: Frontend, DefinitionID: def("frontend.BACKEND_URL").ID, Name: "BACKEND_URL", Source: domain.SourceWorkloadOutput, RefID: Backend, OutputName: "endpoint"},
		},
		Secrets: []domain.ConfiguredValue{
			{ID: "s1", WorkloadID: Backend, DefinitionID: def("backend.DB_PASSWORD").ID, Name: "DB_PASSWORD", Source: domain.SourceResourceOutput, RefID: Postgres, OutputName: "password"},
		},
	}
}

// Running returns HEALTHY workload instances of the given version.
func Running(versionID string, versionNumber int, workloadIDs ...string) []domain.WorkloadInstance {
	var out []domain.WorkloadInstance
	for _, id := range workloadIDs {
		out = append(out, domain.WorkloadInstance{ID: "wi-" + id, ApplicationID: AppID, WorkloadID: id, Environment: domain.Staging,
			Target: "kind-local", Status: domain.WIHealthy, RunningVersionID: versionID, RunningVersionNumber: versionNumber,
			ImageRepository: "registry.company.local/x", ImageVersion: "v1"})
	}
	return out
}

func All(v *domain.ApplicationVersion) []string {
	var ids []string
	for _, w := range v.Workloads {
		ids = append(ids, w.ID)
	}
	return ids
}
