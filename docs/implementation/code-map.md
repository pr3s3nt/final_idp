---
id: IMPLEMENTATION-CODE-MAP
artifact: design-to-code-map
status: current
last_reviewed: 2026-09-18
---

# Design-to-code map

| Design concern | Implementation entry point |
|---|---|
| UC-01 Application Builder shell, component workspaces and read-only Review | [`ApplicationEditorPage.tsx`](../../idp/frontend/src/features/application-definition/pages/ApplicationEditorPage.tsx), [`useApplicationDraft.ts`](../../idp/frontend/src/features/application-definition/hooks/useApplicationDraft.ts), [`BuilderNavigation.tsx`](../../idp/frontend/src/features/application-definition/components/BuilderNavigation.tsx), [`WorkloadWorkspace.tsx`](../../idp/frontend/src/features/application-definition/components/WorkloadWorkspace.tsx), [`ReviewWorkspace.tsx`](../../idp/frontend/src/features/application-definition/components/ReviewWorkspace.tsx) |
| UC-01 Application API, Service and draft validation | [`idp/backend/internal/web/applications.go`](../../idp/backend/internal/web/applications.go), [`internal/service/application.go`](../../idp/backend/internal/service/application.go), [`appvalidator`](../../idp/backend/internal/domain/appvalidator/) |
| UC-01 versioned save (compare-and-write) and specification | [`application_save.go`](../../idp/backend/internal/persistence/application_save.go), [`appspec`](../../idp/backend/internal/domain/appspec/), [`specification_repository.go`](../../idp/backend/internal/persistence/specification_repository.go) |
| Deployment request and confirmation | [`idp/backend/internal/service/orchestrator.go`](../../idp/backend/internal/service/orchestrator.go) |
| Background execution and teardown | [`idp/backend/internal/service/worker.go`](../../idp/backend/internal/service/worker.go) |
| Query path | [`idp/backend/internal/service/query.go`](../../idp/backend/internal/service/query.go) |
| Domain model and plan | [`idp/backend/internal/domain/model.go`](../../idp/backend/internal/domain/model.go), [`plan.go`](../../idp/backend/internal/domain/plan.go) |
| Canonical plan fingerprint | [`idp/backend/internal/domain/canonical.go`](../../idp/backend/internal/domain/canonical.go) |
| Graph and wave planning | [`graphbuilder`](../../idp/backend/internal/domain/graphbuilder/), [`waveplanner`](../../idp/backend/internal/domain/waveplanner/) |
| Resource resolution/planning | [`resourceresolver`](../../idp/backend/internal/domain/resourceresolver/), [`infraplanner`](../../idp/backend/internal/domain/infraplanner/) |
| Configuration/output resolution | [`configresolver`](../../idp/backend/internal/domain/configresolver/), [`resourceoutput`](../../idp/backend/internal/domain/resourceoutput/) |
| Manifest generation and target adaptation | [`manifest`](../../idp/backend/internal/domain/manifest/), [`targetadapter`](../../idp/backend/internal/domain/targetadapter/) |
| Persistence | [`idp/backend/internal/persistence`](../../idp/backend/internal/persistence/) |
| CD providers | [`idp/backend/internal/integration/cd`](../../idp/backend/internal/integration/cd/) |
| Provisioning | [`idp/backend/internal/integration/provisioner`](../../idp/backend/internal/integration/provisioner/) |
| Kubernetes status/runtime access | [`idp/backend/internal/integration/kubernetes`](../../idp/backend/internal/integration/kubernetes/) |
| Delivery repository creation | [`idp/backend/internal/integration/deliveryrepo`](../../idp/backend/internal/integration/deliveryrepo/) |
| Secret storage | [`idp/backend/internal/integration/secretstore`](../../idp/backend/internal/integration/secretstore/) |
| HTTP/API/UI | [`idp/backend/internal/web`](../../idp/backend/internal/web/) |
| Schema migrations | [`idp/backend/migrations`](../../idp/backend/migrations/) |
