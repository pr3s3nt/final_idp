# Persistence Classification

| Object(s) | Classification | Owner/table | Lý do |
|---|---|---|---|
| Application Definition, Workload, embedded Workload Output Definition, Resource Requirement, definitions, Dependency | PERSISTENT | Application Repository | Cấu trúc logic; output metadata nằm trong `workload.exposed_outputs`; referenced definitions dùng retire. |
| Application Specification | PERSISTENT | Specification Repository | Artifact versioned. |
| Environment Configuration, Environment Variable, Configuration Value, Secret/reference | PERSISTENT | Environment Configuration Repository | Chỉ logical/direct non-secret value và opaque reference. |
| Resource Definition | PERSISTENT (platform-managed) | `resource_definition` | Catalog ngoài bốn Developer UC. |
| Deployment, Workload Deployment, Deployment Context | PERSISTENT | Deployment Repository | Input snapshot, lifecycle và fingerprint. |
| Resource Instance, Resource Instance Binding | PERSISTENT | Resource Instance Repository | Instance giữ original ownership/provider identity; exact active binding là canonical reusable lookup path. Shared-consumer binding do platform administration pre-authorize. |
| Deployment Record, three Deployment Steps | PERSISTENT | Deployment Repository | Audit/lifecycle và ba bước nội bộ. |
| Deployment Execution Job | PERSISTENT | `deployment_execution_job` | Transactional outbox, idempotency, lease và retry. |
| ApplicationDefinitionDraft, ConfigurationDefinitionDraft | CLIENT-OWNED DTO | Web UI only | Không có backend draft store và không giữ qua request. |
| Deployment Graph, Resource Resolution | TRANSIENT | Worker/request execution | Rebuild từ persisted inputs. |
| Infrastructure Plan, Infrastructure Plan Item, Override Definition | TRANSIENT | Planner execution | Typed canonical fingerprint input; không persist payload. |
| Resource Output, Workload Output | TRANSIENT | Output resolvers | Resource output đọc sau reconcile; workload output tính plan-time trước configuration resolution. |
| Resolved Configuration, Resolved Specification | TRANSIENT | Worker execution | Không lưu resolved credential/endpoint. |
| `CD_SYNCED`, `APPLICATION_READY` markers | DERIVED VIEW | UC-04 aggregator | Suy ra live từ CD/Kubernetes, không phải row. |

`deployment_execution_job.override_values` chỉ chứa selected values cần để worker áp dụng lại sau khi rebuild plan; nó không chứa Infrastructure Plan, item hay override-definition schema. `accepted_plan_fingerprint` buộc worker kiểm lại plan trước side effect.

Secret Store nằm ngoài DB transaction. `stageSecret` tạo opaque idempotent reference có TTL; save failure gọi `revoke`, save success gọi idempotent `promote`. Crash/retry được giới hạn bằng TTL và orphan reconciliation. Đây là compensation, không phải distributed atomicity.
