package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/pr3s3nt/final_idp/uc03/internal/config"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/manifest"
	"github.com/pr3s3nt/final_idp/uc03/internal/domain/resourceoutput"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/cd"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/imageregistry"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/kubernetes"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/provisioner"
	"github.com/pr3s3nt/final_idp/uc03/internal/integration/secretstore"
	"github.com/pr3s3nt/final_idp/uc03/internal/persistence"
	"github.com/pr3s3nt/final_idp/uc03/internal/service"
	"github.com/pr3s3nt/final_idp/uc03/internal/web"
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
	}
	secrets, err := secretstore.NewEncryptedFile(filepath.Join(cfg.DataDir, "secrets"), cfg.SecretKey)
	if err != nil {
		return err
	}
	registry := imageregistry.Checker{}
	kube := &kubernetes.Adapter{}
	prov := &provisioner.Terraform{ModulesDir: cfg.ModulesDir, DataDir: cfg.DataDir}
	outputs := &resourceoutput.Collector{Provisioner: prov, Secrets: secrets}
	argo := &cd.ArgoCD{RepoURL: cfg.GitOpsRepo, Branch: cfg.GitOpsBranch, WriteKeyFile: cfg.GitOpsKeyFile,
		ReadKeyFile: cfg.GitOpsReadKeyFile, KnownHostsFile: filepath.Join(filepath.Dir(cfg.GitOpsKeyFile), "known_hosts"),
		WorkDir: filepath.Join(cfg.DataDir, "gitops"), Kube: kube}
	orch := &service.Orchestrator{Repositories: repos, Images: registry, Secrets: secrets}

	switch cmd {
	case "serve":
		query := &service.QueryService{Repositories: repos, Orch: orch, Kube: kube, CD: argo, ResourceOutputs: outputs}
		srv := &web.Server{Orch: orch, Query: query, Registry: registry}
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
		if cfg.GitOpsRepo == "" || cfg.GitOpsKeyFile == "" || cfg.GitOpsReadKeyFile == "" {
			return fmt.Errorf("IDP_GITOPS_REPO, IDP_GITOPS_SSH_KEY_FILE and IDP_GITOPS_READ_SSH_KEY_FILE are required by the worker")
		}
		hash := sha256.Sum256([]byte("config-hash:" + cfg.SecretKey))
		worker := &service.Worker{Orch: orch, Provisioner: prov, ResourceOutputs: outputs, Kube: kube, CD: argo,
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
