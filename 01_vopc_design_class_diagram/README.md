# Step 1: VOPC / Design class diagrams

MVP behavior follows [scope](../MVP_SCOPE.md), [deployment design](../MVP_DEPLOYMENT_DESIGN.md) and contracts C1–C13. UC-03/04 diagrams are current execution/query profiles; UC-01/02 diagrams preserve the future full editor, while MVP uses fixture import.

| File | Scope |
|---|---|
| design_class_diagram.puml | Consolidated components, including deferred editor classes and current MVP adapters/snapshot/recovery |
| vopc_uc01.puml | Future application editor; fixture C1 is MVP writer |
| vopc_uc02.puml | Future configuration editor; staging remains R3 deferred |
| vopc_uc03.puml | Current MVP prepare/confirm/worker/recovery ownership |
| vopc_uc04.puml | Current read-only query with revision-aware provider contracts |

Operations belong to their receiving component. Solid arrows are collaborators; interface realization uses <|..; typed transient plan objects are data carriers. Grouped Sources and Materializers in UC-03 are presentation shorthand for the corresponding consolidated repositories/target-config-secret components.

MVP changes:
- Snapshot Builder owns source snapshot assembly. Deployment Repository persists it; worker reads immutable inputs rather than source repositories.
- Confirm includes expectedPlanFingerprint and request idempotency hash; scope guard and job/record/steps share one accept transaction.
- Worker owns fresh execution under exclusive host lock. Deployment Repository owns claim, phase transitions and atomic completion/failure; there is no independent completeJob commit.
- Resource Instance Repository returns all statuses. Reconciler reserves state/identity before provider calls; collector reads durable provider state via Provisioner Adapter.
- Recovery Service owns interruption inspection/audit, retains failed execution and permits a new deployment only after scope verification.
- Secret Reference Adapter validates permanent immutable UID/key references; future Secret Store staging interface is not called by MVP.
- Go Argo CD Adapter publishes OCI artifact and persists intent before upserting the owned Application. Query verifies source/digest/UID before health; OCI Registry is an external participant.
- Full contracts carry precise signatures/preconditions/transactions; ellipses in diagrams are presentation abbreviations, not unspecified protocol.

## Render

Use SVG for the consolidated/domain/ERD and long sequence views; the default PNG size limit can crop large diagrams. Example with the available container: `docker run --rm -v "$PWD:/data:ro" -v /tmp/idp-diagrams:/out -w /data plantuml/plantuml:1.2025.10 -tsvg -o /out '**/*.puml'` (create the output directory first and grant write access to the container user).
