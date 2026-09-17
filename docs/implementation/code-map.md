---
id: IMPLEMENTATION-CODE-MAP
artifact: design-to-code-map
status: current
last_reviewed: 2026-09-17
---

# Design-to-code map

| Design concern | Implementation entry point |
|---|---|
| Deployment request and confirmation | [`uc03/internal/service/orchestrator.go`](../../uc03/internal/service/orchestrator.go) |
| Background execution and teardown | [`uc03/internal/service/worker.go`](../../uc03/internal/service/worker.go) |
| Query path | [`uc03/internal/service/query.go`](../../uc03/internal/service/query.go) |
| Domain model and plan | [`uc03/internal/domain/model.go`](../../uc03/internal/domain/model.go), [`plan.go`](../../uc03/internal/domain/plan.go) |
| Canonical plan fingerprint | [`uc03/internal/domain/canonical.go`](../../uc03/internal/domain/canonical.go) |
| Graph and wave planning | [`graphbuilder`](../../uc03/internal/domain/graphbuilder/), [`waveplanner`](../../uc03/internal/domain/waveplanner/) |
| Resource resolution/planning | [`resourceresolver`](../../uc03/internal/domain/resourceresolver/), [`infraplanner`](../../uc03/internal/domain/infraplanner/) |
| Configuration/output resolution | [`configresolver`](../../uc03/internal/domain/configresolver/), [`resourceoutput`](../../uc03/internal/domain/resourceoutput/) |
| Manifest generation and target adaptation | [`manifest`](../../uc03/internal/domain/manifest/), [`targetadapter`](../../uc03/internal/domain/targetadapter/) |
| Persistence | [`uc03/internal/persistence`](../../uc03/internal/persistence/) |
| CD providers | [`uc03/internal/integration/cd`](../../uc03/internal/integration/cd/) |
| Provisioning | [`uc03/internal/integration/provisioner`](../../uc03/internal/integration/provisioner/) |
| Kubernetes status/runtime access | [`uc03/internal/integration/kubernetes`](../../uc03/internal/integration/kubernetes/) |
| Delivery repository creation | [`uc03/internal/integration/deliveryrepo`](../../uc03/internal/integration/deliveryrepo/) |
| Secret storage | [`uc03/internal/integration/secretstore`](../../uc03/internal/integration/secretstore/) |
| HTTP/API/UI | [`uc03/internal/web`](../../uc03/internal/web/) |
| Schema migrations | [`uc03/migrations`](../../uc03/migrations/) |
