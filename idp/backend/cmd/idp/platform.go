package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"idp/internal/config"
	"idp/internal/domain/manifest"
	"idp/internal/domain/resourceoutput"
	"idp/internal/integration/cd"
	"idp/internal/integration/deliveryrepo"
	"idp/internal/integration/imageregistry"
	"idp/internal/integration/kubernetes"
	"idp/internal/integration/provisioner"
	"idp/internal/integration/secretstore"
	"idp/internal/persistence"
	"idp/internal/service"
	"idp/internal/web"
)

// runPlatform wires every UC-03 component to its concrete implementation.
func runPlatform(ctx context.Context, cfg *config.Config, db *persistence.DB, cmd string, args []string) error {
	repos := service.Repositories{
		Apps:        &persistence.ApplicationRepository{DB: db},
		Configs:     &persistence.EnvironmentConfigurationRepository{DB: db},
		Catalog:     &persistence.ResourceDefinitionCatalog{DB: db},
		Resources:   &persistence.ResourceInstanceRepository{DB: db},
		Workloads:   &persistence.WorkloadInstanceRepository{DB: db},
		Deployments: &persistence.DeploymentRepository{DB: db},
		Delivery:    &persistence.DeliveryRepositoryRegistry{DB: db},
	}
	secrets, err := secretstore.NewEncryptedFile(filepath.Join(cfg.DataDir, "secrets"), cfg.SecretKey)
	if err != nil {
		return err
	}
	registry := imageregistry.Checker{}
	kube := &kubernetes.Adapter{}
	prov := &provisioner.Terraform{ModulesDir: cfg.ModulesDir, DataDir: cfg.DataDir}
	outputs := &resourceoutput.Collector{Provisioner: prov, Secrets: secrets}
	deliveryDir := filepath.Join(cfg.DataDir, "delivery")
	if err := os.MkdirAll(deliveryDir, 0o700); err != nil {
		return err
	}
	provider := &deliveryrepo.GitHub{Pattern: cfg.DeliveryRepoPattern, Branch: cfg.DeliveryBranch,
		TokenReference: cfg.GitHostingTokenRef, Secrets: secrets, WorkDir: deliveryDir,
		APIBase: cfg.GitHostingAPI, SSHHost: cfg.GitHostingSSHHost}
	delivery := cd.Delivery{Provider: provider, Registry: repos.Delivery, Secrets: secrets, Branch: cfg.DeliveryBranch,
		KnownHostsFile: cfg.KnownHostsFile, WorkDir: deliveryDir, Kube: kube}
	// The CD system is chosen by the platform; UC-03 only ever sees the CD
	// abstraction, so either provider serves the same deployment flow.
	var cdProvider cd.Integration
	switch cfg.CDProvider {
	case "fleet":
		cdProvider = cd.NewFleet(delivery)
	case "argocd":
		cdProvider = cd.NewArgoCD(delivery)
	default:
		return fmt.Errorf("IDP_CD_PROVIDER must be fleet or argocd, not %q", cfg.CDProvider)
	}
	orch := &service.Orchestrator{Repositories: repos, Images: registry, Secrets: secrets}

	switch cmd {
	case "serve":
		query := &service.QueryService{Repositories: repos, Orch: orch, Kube: kube, CD: cdProvider, ResourceOutputs: outputs}
		apps := &service.ApplicationService{Apps: repos.Apps, Specs: &persistence.SpecificationRepository{DB: db}}
		srv := &web.Server{Orch: orch, Query: query, Registry: registry, Apps: apps, FrontendDir: cfg.FrontendDir}
		httpServer := &http.Server{Addr: cfg.ListenAddr, Handler: srv.Handler()}
		go func() {
			<-ctx.Done()
			httpServer.Close()
		}()
		log.Printf("IDP listening on http://%s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	case "worker":
		if cfg.DeliveryRepoPattern == "" {
			return fmt.Errorf("IDP_DELIVERY_REPO_PATTERN is required by the worker, for example pr3s3nt/idp-<app>-gitops")
		}
		hash := sha256.Sum256([]byte("config-hash:" + cfg.SecretKey))
		worker := &service.Worker{Orch: orch, Provisioner: prov, ResourceOutputs: outputs, Kube: kube, CD: cdProvider,
			Renderer: &manifest.ScoreRenderer{}, Secrets: secrets, HashKey: hash[:],
			HealthTimeout: cfg.HealthTimeout, SyncTimeout: cfg.HealthTimeout}
		return worker.Run(ctx, cfg.PollInterval)
	case "fail-orphaned-job":
		if len(args) < 2 {
			return fmt.Errorf("usage: idp fail-orphaned-job <deployment-id> <reason>")
		}
		return repos.Deployments.FailOrphanedJob(ctx, args[0], args[1])
	}
	return fmt.Errorf("unknown command %q", cmd)
}
