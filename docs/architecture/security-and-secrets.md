---
id: ARCH-SECURITY-SECRETS
artifact: security-and-secrets-architecture
status: current
last_reviewed: 2026-09-17
---

# Security and Secret architecture

This document owns the current storage and materialization rules for secret values in the UC-03 implementation.

## Storage boundary

The Secret Store implementation keeps each value in an encrypted file using AES-256-GCM. It derives the 256-bit encryption key from `IDP_SECRET_KEY` with SHA-256, uses a fresh random GCM nonce per write, authenticates the opaque reference as associated data, stores directories with mode `0700`, and stores encrypted files with mode `0600`.

Callers persist only opaque references of the form `idpsecret://<id>`. Plaintext secret values, Git hosting tokens, delivery-repository private keys, and cluster credentials must not be written to the IDP database, Delivery Repository, Deployment Record, deployment-step detail, or application logs. `IDP_SECRET_KEY` must be supplied to every process that reads the same store and is not written by the Secret Store itself.

## Browser storage

UC-01 declares Secret names only; it has no input for a Secret value. Browser
`sessionStorage` holds only non-sensitive draft content, as decided in
[ADR-016](../decisions/ADR-016-client-owned-drafts.md) and
[ADR-017](../decisions/ADR-017-react-web-frontend.md). Tokens, credentials and
plaintext Secret values are never written to browser storage.

## Workload materialization

During deployment, Secret Materializer resolves authorized references in memory and creates or updates a Kubernetes `Secret` through the Kubernetes API. Generated workload manifests refer to that object through `secretKeyRef`; plaintext never enters Git desired state. A keyed HMAC of the materialized secret inputs is placed in the pod-template annotation so secret changes trigger rollout without exposing the value.

The target namespace owns the materialized Kubernetes Secret. Cleanup follows workload/namespace lifecycle; orphan compensation before successful save remains tracked in [D07](../backlog/D07-orphaned-secret.md).

## Implementation anchors

- Encrypted store: `idp/backend/internal/integration/secretstore/secretstore.go`
- Manifest references and HMAC: `idp/backend/internal/domain/manifest/pipeline.go`
- Kubernetes write path: the selected CD/Kubernetes adapter invoked by `idp/backend/internal/service/worker.go`
