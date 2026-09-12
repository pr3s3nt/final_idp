# Step 2: Domain Objects

Các aggregate root là **Application Definition**, **Environment Configuration**, **Deployment** và **Resource Instance**. Draft chỉnh sửa UC-01/UC-02 là DTO do Web UI sở hữu; application service không giữ draft giữa các request.

| Object | Thuộc tính chính | Vai trò / ownership |
|---|---|---|
| Application Definition | `applicationId`, `name`, `description`, `retiredAt`, timestamps | Aggregate cấu trúc logic; referenced definition được retire. |
| Workload | `workloadId`, `name`, `type`, `imageRepository`, `port`, `exposedOutputs`, `retiredAt` | Entity thuộc Application Definition. Output definition ghi availability/resolution kind; referenced workload được retire. |
| Workload Output Definition | `outputName`, `availability`, `resolutionKind` | Value object embedded trong Workload; resolver chỉ dùng `PLAN_TIME` trong cùng deployment. |
| Resource Requirement | `resourceRequirementId`, `name`, `resourceType`, `retiredAt` | Logical resource thuộc application; identity được giữ cho reuse/history. |
| Environment Variable Definition | `variableDefinitionId`, `name`, `required`, `retiredAt` | Requirement thuộc Workload. |
| Secret Definition | `secretDefinitionId`, `name`, `required`, `retiredAt` | Secret metadata thuộc Workload; không có plaintext. |
| Dependency | source và đúng một target workload/resource | Topology thuộc Application Definition. |
| Application Specification | format, content, version | Artifact sinh từ active definition. |
| Environment Configuration | application, environment, variables, secrets | Aggregate binding theo environment. |
| Configuration Value | `DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT` và payload tương ứng | Value object/reference được persist; không chứa resolved runtime value. |
| Secret | workload, definition, `secretReference` hoặc sensitive resource reference | Chỉ giữ opaque reference sau khi upload. |
| Resource Definition | provisioner, contexts, parameters, overrides, outputs, `retiredAt` | Catalog platform-managed; referenced definition được retire. |
| Deployment | application/configuration, target, fingerprint/algorithm, lifecycle status | Aggregate cho một execution. Lifecycle literal: `AWAITING_CONFIRMATION`, `QUEUED`, `RUNNING`, `SUBMITTED`, `FAILED`. |
| Workload Deployment | workload identity, image repository/version | Snapshot lịch sử; giữ FK tới Workload đã retire. |
| Deployment Context | cloud provider, region, target input | Value object persisted 1:1. |
| Deployment Graph | workload/resource/dependency/configuration IDs | Typed transient graph. |
| Resource Resolution | requirement/definition/decision | Typed transient decision. |
| Infrastructure Plan | `applicationId`, `environment`, `deploymentTarget`, ordered `items` | Typed transient root; payload không persist. |
| Infrastructure Plan Item | requirement, definition, `CREATE/UPDATE/REUSE`, parameters, owner/scope/sharing key, optional instance, override definitions | Canonical plan item. |
| Override Definition | key, value type, required, constraints | Canonical transient schema; khác với selected override values của execution job. |
| Resource Instance | definition, original owner application/environment/requirement, target, sharing scope/key, provider references, status | Durable infrastructure identity. Owner columns phục vụ ownership/integrity, không phải reusable lookup path. |
| Resource Instance Binding | instance, application, environment, logical requirement, target, role, sharing key, `retiredAt` | Persistent exact active lookup scope. Shared reuse cần pre-authorized consumer binding; replacement retires old binding. |
| Resource Output | instance, name, value, sensitivity | Runtime transient output từ provider sau infrastructure ready. |
| Workload Output | workload, name, resolved value, `PLAN_TIME` | Transient output do Workload Output Resolver tính từ graph/metadata/context, ví dụ Kubernetes Service DNS. Runtime-only output của cùng deployment bị từ chối để tránh vòng tròn. |
| Resolved Configuration / Specification | resolved values/dependencies/images | Transient trong worker execution. |
| Deployment Record | lifecycle status, delivery reference/status, target, error | Durable record; lifecycle dùng cùng enum với Deployment, còn external delivery status dùng enum riêng. |
| Deployment Step | one of three internal names, step status, timing/error | Chỉ persist `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`. |
| Deployment Execution Job | deployment, idempotency key, selected overrides, accepted fingerprint, lease/attempt/status | Durable job đồng thời là transactional outbox; worker có thể claim/retry an toàn. |

## Invariants

- Draft UC-01/UC-02 được gửi đầy đủ trong mỗi mutation request và được trả lại cho Web UI. Backend stateless giữa request.
- UI xóa plaintext secret ngay sau khi nhận opaque staged reference; repositories và log không nhận plaintext.
- `Infrastructure Plan`, item và override definition dùng canonical JSON: sort item/key ổn định, normalize number/unit, bỏ timestamp/ID sinh kỹ thuật. Fingerprint domain gồm action, requirement/definition/instance identity, target/scope/sharing, resolved parameters và bốn definition fields `provisionerReference`, `supportedContexts`, `defaultParameters`, `allowedOverrides`. Chỉ SHA-256 fingerprint và algorithm persist trên Deployment.
- Selected override values tối thiểu cần cho worker được lưu trong execution job, không phải plan payload; worker rebuild plan và kiểm lại fingerprint trước reconcile.
- Resource Instance có original owner tuple và ít nhất một OWNER binding. Reuse query luôn match exact active `(applicationId, environment, resourceRequirementId, deploymentTarget)` qua binding; không fallback sang owner columns. Cross-boundary reuse chỉ thấy instance qua `EXPLICIT_SHARED` + matching `sharingKey`/policy và `SHARED_CONSUMER` binding đã được platform pre-authorize ngoài deployment flow.
- Workload/definition đã được configuration hoặc deployment history tham chiếu chỉ được retire. Hard delete chỉ hợp lệ khi chưa từng có reference; FK lịch sử luôn `RESTRICT`.
- `CD_SYNCED` và `APPLICATION_READY` là view marker do UC-04 suy ra live; chúng không phải Deployment Step hay lifecycle status.
