---
id: IMPLEMENTATION-CODE-MAP
artifact: design-to-code-map
status: current
last_reviewed: 2026-09-17
---

# Design-to-code map

| Design concern | Implementation entry point |
|---|---|
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
