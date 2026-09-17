---
id: DEMO-APPS-README
artifact: code-area-readme
status: current
last_reviewed: 2026-09-17
---

# Demo applications

These small Go services are workloads deployed by the IDP during demos and
end-to-end verification. They are not part of the IDP backend or frontend.

The backend image-build helper at
[`idp/backend/prerequisites/build-push-images.sh`](../idp/backend/prerequisites/build-push-images.sh)
builds the `backend`, `frontend`, and `worker` commands with the expected demo
version metadata.
