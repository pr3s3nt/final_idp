# Step 5: State Machine

Step này mô tả lifecycle có trạng thái và transition không tầm thường của hai aggregate root: **Deployment** và **Resource Instance**. Tên state dễ đọc bám theo tiến trình UC-03/UC-04; literal viết hoa trong ngoặc là logical value của cột `status`.

Step 3 mới xác định `deployment.status`, `deployment_record.status`, `deployment_step.status` và `resource_instance.status` có kiểu `ENUM`, nhưng chưa chốt tập literal vật lý. Vì vậy state machine của Deployment chỉ tái sử dụng nguyên văn các logical state đã được Operation Contracts chốt: `AWAITING_CONFIRMATION`, `CONFIRMED`, `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `SUBMITTED` và `FAILED`. Tên progress của UC-04 không được suy diễn thành literal của `deployment.status`.

## Vì sao chỉ có hai state machine

**Deployment** cần state machine vì aggregate đi qua validation/confirmation, infrastructure reconciliation, configuration resolution và submission sang CD với các transition/failure point đã được operation contract định nghĩa. Manifest generation là processing phase trước submission, còn CD synchronization và workload health là progress/status được UC-04 tổng hợp chứ không phải các `deployment.status` transition đã được contract hóa. **Resource Instance** cũng có lifecycle độc lập vì infrastructure có thể được create, update hoặc reuse, có provisioning state, provider reference và failure state cần được theo dõi qua nhiều deployment.

Các domain object còn lại không cần state machine riêng. **Application Definition** chỉ có hành vi Draft/Saved đơn giản trong UC-01; tài liệu hiện tại không định nghĩa `status` cho object này. **Environment Configuration**, **Workload**, **Resource Requirement**, **Resource Definition**, **Workload Deployment**, **Deployment Context** và các configuration definition/binding cũng không có lifecycle enum hay transition nhiều bước trong ERD. **Deployment Record** và **Deployment Step** là dữ liệu ghi nhận/snapshot của lifecycle Deployment, không phải lifecycle aggregate độc lập. Các object `Deployment Graph`, `Resource Resolution`, `Infrastructure Plan`, `Resource Output`, `Resolved Configuration` và `Resolved Specification` là execution-scoped/transient nên không tạo state machine persistent riêng; riêng Infrastructure Plan chỉ persist SHA-256 fingerprint và algorithm tương ứng trên `deployment`, không persist plan payload.

## Deployment state mapping

| Diagram state | ERD enum value | Operation/event đưa object vào state |
|---|---|---|
| Created / Validating (execution only) | N/A — chưa có row `deployment` | `createDeployment` khởi tạo attempt; `validateDeploymentInput` chạy bên trong orchestration. |
| Rejected / Validation Failed (execution only) | N/A — transaction rollback, không có row `deployment` | A1 của `createDeployment`: trả validation error và kết thúc attempt trước persistence. Đây không phải `FAILED`. |
| Validated / Awaiting Confirmation | `AWAITING_CONFIRMATION` | `createDeployment` hoàn tất validation, dựng graph, resolve Resource Definition và lập infrastructure plan; đây là postcondition được persist của contract. |
| Validated / Awaiting Confirmation (`PLAN_CHANGED` self-transition) | `AWAITING_CONFIRMATION` | `confirmDeployment` rebuild plan và phát hiện fingerprint mismatch: refresh fingerprint, trả `PLAN_CHANGED`, re-present rebuilt plan để review/re-confirm, không apply override hoặc reconcile; status không đổi. |
| Infrastructure Reconciling | `CONFIRMED` | `confirmDeployment` với permitted overrides hợp lệ; `reconcileInfrastructure` là operation kế tiếp xử lý final plan. |
| Infrastructure Ready | `INFRASTRUCTURE_READY` | `reconcileInfrastructure` thành công cho toàn bộ create/update/reuse items. Đây cũng là progress step **Infrastructure Ready** của UC-04. |
| Configuration Resolved | `CONFIGURATION_RESOLVED` | `collectResourceOutputs` hoàn tất, sau đó `resolveEnvironmentConfiguration` thành công. Đây là progress step **Configuration Resolved** của UC-04. |
| Submitted to CD | `SUBMITTED` | Sau khi manifest generation/adaptation/materialization hoàn tất, `publishDesiredDeploymentState` được CD System accept; đúng postcondition của contract 8 và chưa có nghĩa CD sync hoàn tất hoặc workload Healthy. |
| Failed | `FAILED` | A2 tại `reconcileInfrastructure`, runtime configuration resolution, manifest generation/materialization hoặc CD delivery; `saveDeploymentRecord` persist `FAILED`, failed `Deployment Step`, `error_summary` và `related_component_reference` nếu xác định được. |

Nhánh A1 cần được đọc đúng theo contract: nếu `validateDeploymentInput` thất bại trước khi `createDeployment` hoàn tất, diagram đi tới outcome execution-only **Rejected / Validation Failed**; transaction rollback nên không có `deployment`, `deployment_record` hoặc `deployment_step`. Nếu override khi confirm không hợp lệ, Deployment vẫn ở `AWAITING_CONFIRMATION`. Node `FAILED` chỉ dành cho nhánh A2 sau khi Deployment đã tồn tại và thực sự persist `deployment.status = FAILED` cùng failed step và error summary để UC-04 đọc.

Tại confirm, fingerprint mismatch tạo self-transition `PLAN_CHANGED` trên `AWAITING_CONFIRMATION`: hệ thống re-present rebuilt plan, allowed overrides và fingerprint mới, không reconcile, và giữ nguyên status. Nếu fingerprint khớp nhưng request thua atomic CAS `AWAITING_CONFIRMATION` → `CONFIRMED`, hệ thống trả `DEPLOYMENT_ALREADY_CONFIRMED`; losing request cũng không tạo state transition và không reconcile.

`Deployment Record.status` được đồng bộ với current/final `Deployment.status` bởi `saveDeploymentRecord`. Với success path được contract hóa hiện tại, aggregate đi thẳng từ `CONFIGURATION_RESOLVED` sang `SUBMITTED` khi `publishDesiredDeploymentState` thành công; **Manifest Generated** chỉ là progress marker của work đã hoàn tất trước transition đó, không phải một giá trị trung gian của `deployment.status`.

Năm progress marker UC-04 — **Infrastructure Ready**, **Configuration Resolved**, **Manifest Generated**, **CD Synced**, **Application Ready** — là năm row `deployment_step` riêng biệt, mỗi row có `step_name`, `status`, timestamps và failure detail riêng. Vì vậy cả năm dấu kiểm có thể xuất hiện đồng thời cho cùng một Deployment; chúng không phải năm giá trị loại trừ nhau của `deployment.status`. UC-04 chỉ đọc và tổng hợp Deployment Repository, CD Status Provider và Workload Status Provider; nó không chạy Orchestrator, reconcile infrastructure, trigger deployment hoặc ghi transition vào aggregate. Không có operation contract sau `saveDeploymentRecord`, nên CD sync/health observation không được suy diễn thành literal mới của `deployment.status`; aggregate vẫn `SUBMITTED` (hoặc `FAILED` trên nhánh A2 đã contract hóa).

## Resource Instance state mapping

| Diagram state | ERD enum value | Operation/event đưa object vào state |
|---|---|---|
| Planned | `PLANNED` | `planInfrastructureChanges` quyết định create hoặc update dựa trên graph, resolved definition và current instances. Với create, phase này có thể chỉ nằm trong execution plan trước khi durable Resource Instance được tạo. |
| Provisioning | `PROVISIONING` | `reconcileInfrastructure` bắt đầu provisioner apply cho create/update item. |
| Ready | `READY` | `provisioner apply success` và các infrastructure/provider-state references đã được persist; hoặc reconcile quyết định reuse và xác nhận instance hiện hữu vẫn ready. `collectResourceOutputs` chỉ chạy sau state này. |
| Failed | `FAILED` | `provisioner apply failure`; repository phản ánh provider state/reference thực tế và không báo ready giả. Failure này đồng thời làm Deployment đi tới `FAILED` theo UC-03 A2. |

Create, update và reuse là decision của `planInfrastructureChanges`, không phải ba status cạnh tranh. Create là entry path mới `PLANNED` → `PROVISIONING` → `READY`; update giữ nguyên `resourceInstanceId` rồi đi từ instance hiện hữu `READY` qua `PLANNED` → `PROVISIONING` → `READY`. Reuse không phải entry point mới: contract 6 yêu cầu target `resource_instance_id` đã tồn tại, nên nó chỉ là self-transition ở `READY` để refresh/xác nhận provider state. `collectResourceOutputs` cũng là self-transition ở `READY` vì `Resource Output` là runtime view `TRANSIENT`, không có table và không làm đổi lifecycle status của Resource Instance.

## Traceability

- Thứ tự Deployment bám main flow của `sequence_digrams/uc_03_deploy_application.puml`: create/validate/plan → confirm → reconcile infrastructure → collect outputs/resolve configuration → generate/adapt/materialize manifest → publish CD state → lưu record.
- Các `deployment_step` progress marker bám UC-04: **Infrastructure Ready**, **Configuration Resolved**, **Manifest Generated**, **CD Synced**, **Application Ready**; chúng là các row độc lập có thể đồng thời hoàn tất, không phải chuỗi `deployment.status`.
- Failure semantics bám UC-03 A1/A2 và postconditions của `createDeployment`, `confirmDeployment`, `reconcileInfrastructure`, `resolveEnvironmentConfiguration`, `publishDesiredDeploymentState`, `saveDeploymentRecord`.
- Persistent status nằm ở `deployment.status`, `deployment_record.status`, `deployment_step.status` và `resource_instance.status`; resolved outputs/config/specification vẫn là transient như Domain Model và ERD đã quy định.
