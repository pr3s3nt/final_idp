---
id: STATE-MACHINE-INDEX
artifact: state-machine-index
status: current
last_reviewed: 2026-09-18
---

# Step 5: State Machine

Step này mô tả lifecycle có trạng thái và transition không tầm thường của **Auth Session**, **Deployment**, **Resource Instance** và **Workload Instance**. Tên state dễ đọc bám theo tiến trình UC-03/UC-04/UC-06; literal viết hoa trong ngoặc là giá trị của cột `status`, còn trạng thái Auth Session được suy ra từ timestamp và trạng thái account.

Diagram sources: [Auth Session](auth-session.puml), [Deployment](deployment.puml), [Resource Instance](resource-instance.puml), and [Workload Instance](workload-instance.puml).

Mọi literal dùng đúng mục **Danh mục ENUM** trong `docs/architecture/database/schema.md`: `user_account.status` gồm `ACTIVE`, `DISABLED`; `deployment.status` gồm `AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED`, `FAILED`; `resource_instance.status` gồm `PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `DESTROYED`, `UNLINKED`; `workload_instance.status` gồm `DEPLOYING`, `HEALTHY`, `FAILED`, `REMOVED`. Auth Session không có status enum; trạng thái của nó được suy ra từ `revoked_at`, `expires_at`, `last_seen_at` và account status.

## Vì sao có bốn state machine

**Auth Session** có lifecycle độc lập qua nhiều request: active, idle/absolute expiry hoặc bị revoke bởi logout/reset/disable; trạng thái được suy ra thay vì lưu enum. **Deployment** cần state machine vì aggregate đi qua validation/confirmation, tạo job, thực thi nền theo tầng và kết thúc thành công hoặc thất bại. **Resource Instance** có lifecycle độc lập vì infrastructure có thể được create, update, reuse, liên kết, hủy hoặc gỡ liên kết. **Workload Instance** có lifecycle độc lập vì trạng thái workload theo environment + target được dùng xuyên suốt nhiều deployment.

Các domain object còn lại không cần state machine riêng. **Local User Account** chỉ có status `ACTIVE`/`DISABLED` do Operator đặt trực tiếp; không có workflow nhiều bước. **Application Definition** và **Application Definition Version** không có `status`: mỗi lần lưu chỉ tạo một phiên bản bất biến. **Environment Configuration**, **Workload**, **Resource Requirement**, **Resource Definition**, **Workload Deployment**, **Deployment Context** và các configuration definition/binding cũng không có lifecycle enum hay transition nhiều bước. **Deployment Record** và **Deployment Step** là dữ liệu ghi nhận/snapshot của lifecycle Deployment, không phải lifecycle aggregate độc lập. **Deployment Execution Job** có trạng thái nhưng tập giá trị và cơ chế phục hồi chưa chốt (D6), nên chưa mô hình hóa state machine. Các object execution-scoped/transient không tạo state machine persistent riêng.

## Auth Session state mapping

| Diagram state | Persistence condition | Operation/event đưa object vào state |
|---|---|---|
| Active | `revoked_at IS NULL`, chưa quá `expires_at`, idle dưới 30 phút và account `ACTIVE` | `signIn()` tạo session; `authenticateRequest()` hợp lệ cập nhật `last_seen_at`. |
| Idle expired | `last_seen_at <= now() - 30 minutes` | `authenticateRequest()` từ chối session. |
| Absolute expired | `expires_at <= now()` | `authenticateRequest()` từ chối session bất kể activity. |
| Revoked | `revoked_at IS NOT NULL` | `signOut()`, `resetLocalPassword()` hoặc `setLocalUserStatus(DISABLED)` đặt timestamp. |

Enable account không phục hồi session cũ. Session expired/revoked không quay lại
Active; Developer phải đăng nhập để tạo session mới.

## Deployment state mapping

| Diagram state | ERD enum value | Operation/event đưa object vào state |
|---|---|---|
| Created / Validating (execution only) | N/A — chưa có row `deployment` | `createDeployment` khởi tạo attempt; `validateDeploymentInput` chạy bên trong orchestration. |
| Rejected / Validation Failed (execution only) | N/A — transaction rollback, không có row `deployment` | A1 của `createDeployment` (gồm cấu hình không khớp phiên bản, deploy một phần với phiên bản khác phiên bản đang chạy, depends on tạo vòng, workload phụ thuộc ngoài phạm vi chưa healthy): trả validation error và kết thúc attempt trước persistence. Đây không phải `FAILED`. |
| Validated / Awaiting Confirmation | `AWAITING_CONFIRMATION` | `createDeployment` hoàn tất validation, dựng graph, chia tầng, resolve Resource Definition, lập infrastructure plan và lưu fingerprint. |
| Awaiting Confirmation (`PLAN_CHANGED` self-transition) | `AWAITING_CONFIRMATION` | `confirmDeployment` rebuild plan và phát hiện fingerprint mismatch: refresh fingerprint, trả `PLAN_CHANGED`, re-present rebuilt plan; không tạo job; status không đổi. |
| Confirmed, job created | `CONFIRMED` | `confirmDeployment` với override hợp lệ: atomic CAS status và tạo `deployment_execution_job` (kèm override) trong cùng một transaction; request trả lời ngay. |
| Deploying wave by wave | `DEPLOYING` | Deployment Worker lấy job (`claimNextExecutionJob`) và đổi status. Trong status này, với từng tầng: `reconcileInfrastructure`, `collectResourceOutputs`, `resolveEnvironmentConfiguration`, sinh/adapt/materialize manifest, `publishDesiredDeploymentState`, `waitForWorkloadsHealthy`, `collectWorkloadOutputs`, `propagateOutputChanges`; sau các tầng, gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản. |
| Succeeded | `SUCCEEDED` | `saveDeploymentRecord` sau khi mọi workload trong phạm vi (kể cả `CASCADED`) healthy và việc gỡ đã xong; Worker đổi status `DEPLOYING` → `SUCCEEDED`. |
| Failed | `FAILED` | A2 tại bất kỳ tầng nào (provisioning, resolve configuration, manifest, CD delivery, workload không healthy, đọc output) hoặc ở bước gỡ; Worker dừng phần còn lại và `saveDeploymentRecord` lưu tầng, thành phần liên quan và error summary. |

Nhánh A1 cần được đọc đúng theo contract: nếu `validateDeploymentInput` thất bại trước khi `createDeployment` hoàn tất, diagram đi tới outcome execution-only **Rejected / Validation Failed**; transaction rollback nên không có `deployment`, `deployment_execution_job`, `deployment_record` hoặc `deployment_step`. Nếu override khi confirm không hợp lệ, Deployment vẫn ở `AWAITING_CONFIRMATION` và không có job. Node `FAILED` chỉ dành cho nhánh A2 sau khi Deployment đã tồn tại.

Tại confirm, fingerprint mismatch tạo self-transition `PLAN_CHANGED` trên `AWAITING_CONFIRMATION`. Nếu fingerprint khớp nhưng request thua atomic CAS `AWAITING_CONFIRMATION` → `CONFIRMED`, hệ thống trả `DEPLOYMENT_ALREADY_CONFIRMED`; transaction của losing request rollback nên không có job thứ hai.

`CONFIRMED` là trạng thái chờ: deployment đã được xác nhận và job đã tồn tại, nhưng Deployment Worker chưa lấy job. Việc worker chết khi đang ở `DEPLOYING` (lease, thử lại, làm tiếp) chưa được mô hình hóa; xem D6.

Trong `DEPLOYING`, các bước của từng tầng và từng thành phần là progress (các row `deployment_step` có `wave_number`, `related_component_reference`), không phải giá trị trung gian của `deployment.status`. Ai ghi các row này và ghi lúc nào để lại cho D4. `deployment.status` do IDP tự quyết dựa trên việc chờ workload healthy, không bao giờ nhận giá trị từ CD system (D5). UC-04 chỉ đọc và tổng hợp Deployment Repository, Resource/Workload Instance Repository, CD Status Provider và Workload Status Provider; nó không ghi transition vào aggregate.

## Resource Instance state mapping

| Diagram state | ERD enum value | Operation/event đưa object vào state |
|---|---|---|
| Planned | `PLANNED` | `planInfrastructureChanges` quyết định create hoặc update (Resource Definition `MANAGED`, kể cả cụm Kubernetes và network trên cloud) dựa trên graph, resolved definition trong phiên bản catalog và instance tìm theo khóa chủ sở hữu; update khi đầu vào hiện tại khác `applied_input_fingerprint`. `propagateOutputChanges` cũng đưa instance `READY` về phase này khi output của Platform Requirement mà nó `requires` thay đổi. Với create, phase này có thể chỉ nằm trong execution plan trước khi durable Resource Instance được tạo. |
| Provisioning | `PROVISIONING` | `reconcileInfrastructure` bắt đầu provisioner apply cho create/update item `MANAGED`. |
| Ready | `READY` | `provisioner apply success` và các infrastructure/provider-state references đã được persist; hoặc reconcile quyết định reuse; hoặc reconcile liên kết instance của chủ sở hữu tới thứ có sẵn `EXISTING` — resource dùng chung hoặc cụm Kubernetes nội bộ (không provisioning). `collectResourceOutputs` và `propagateOutputChanges` (cập nhật `output_fingerprint`) chỉ chạy ở state này. |
| Failed | `FAILED` | `provisioner apply failure`; repository phản ánh provider state/reference thực tế và không báo ready giả. Failure này đồng thời làm Deployment đi tới `FAILED` theo UC-03 A2. |
| Destroyed | `DESTROYED` | `reconcileInfrastructure` hủy resource `MANAGED` không còn trong phiên bản được deploy (provisioner destroy). Trạng thái kết thúc; dòng được giữ lại. |
| Unlinked | `UNLINKED` | `reconcileInfrastructure` gỡ liên kết resource `EXISTING` không còn trong phiên bản được deploy; hạ tầng thật không bị đụng tới. Trạng thái kết thúc; dòng được giữ lại. |

Create, update, reuse và liên kết là decision của `planInfrastructureChanges`, không phải các status cạnh tranh. Create là entry path mới `PLANNED` → `PROVISIONING` → `READY`; update giữ nguyên `resourceInstanceId` rồi đi từ instance hiện hữu `READY` qua `PLANNED` → `PROVISIONING` → `READY`. Reuse là self-transition ở `READY`. Liên kết resource `EXISTING` là entry path trực tiếp vào `READY` và không bao giờ gọi provisioner. `DESTROYED`/`UNLINKED` là trạng thái kết thúc; nếu chủ sở hữu cần resource lại ở phiên bản sau, một Resource Instance mới được tạo. Mỗi khóa chủ sở hữu có tối đa một instance chưa ở trạng thái kết thúc.

## Workload Instance state mapping

| Diagram state | ERD enum value | Operation/event đưa object vào state |
|---|---|---|
| Deploying | `DEPLOYING` | `publishDesiredDeploymentState` publish workload của một tầng (được chọn hoặc `CASCADED`); `current_workload_deployment_id` trỏ tới workload deployment vừa publish. |
| Healthy | `HEALTHY` | `waitForWorkloadsHealthy` xác nhận mọi pod healthy, rồi `collectWorkloadOutputs` và `propagateOutputChanges` cập nhật `output_fingerprint`. |
| Failed | `FAILED` | Workload không healthy (A2); Deployment Worker dừng các tầng sau. |
| Removed | `REMOVED` | Lần publish cuối không còn workload vì phiên bản được deploy không còn chứa nó. Trạng thái kết thúc; dòng được giữ lại. |

Mỗi workload (ID cố định) + environment + deployment target có đúng một Workload Instance. `HEALTHY` được dùng khi deploy một phần (đọc output của workload phụ thuộc đang chạy ngoài phạm vi) và khi lan truyền (lấy image đang chạy của thành phần phụ thuộc).

## Traceability

- Thứ tự Deployment bám main flow của `docs/use-cases/UC-03/sequence.puml`: create/validate/chia tầng/plan → confirm (tạo job, trả lời ngay) → Deployment Worker lấy job → với mỗi tầng: reconcile infrastructure → collect Resource Output → resolve configuration → generate/adapt/materialize manifest → publish CD state → chờ healthy → collect Workload Output → lan truyền thay đổi output → gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản → lưu record.
- Các `deployment_step` bám UC-04 theo tầng và thành phần; chúng không phải chuỗi `deployment.status`. Tập bước và writer thuộc D4.
- Failure semantics bám UC-03 A1/A2 và postconditions của `createDeployment`, `confirmDeployment`, `reconcileInfrastructure`, `resolveEnvironmentConfiguration`, `publishDesiredDeploymentState`, `collectWorkloadOutputs`, `propagateOutputChanges`, `saveDeploymentRecord`.
- Authentication lifecycle dùng `user_account.status` cùng `auth_session.revoked_at`/`expires_at`/`last_seen_at`; deployment lifecycle dùng `deployment.status`, `resource_instance.status`, `workload_instance.status`, cùng `deployment_record.status` (D5), `deployment_step.status` (D4) và `deployment_execution_job.status` (D6). Resolved outputs/config/specification vẫn là transient như Domain Model và ERD đã quy định.
