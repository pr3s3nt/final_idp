# Step 5: State Machines

Hai aggregate có lifecycle không tầm thường là **Deployment** và **Resource Instance**. Mọi physical enum đã được chốt và đồng bộ với Step 2/3/4.

## Physical enum registry

| Field | Literals |
|---|---|
| `dependency.target_type` | `WORKLOAD`, `RESOURCE` |
| `configuration_value.value_source` | `DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT` |
| `secret.value_source` | `SECRET_REF`, `RESOURCE_OUTPUT` |
| `deployment.status` | `AWAITING_CONFIRMATION`, `QUEUED`, `RUNNING`, `SUBMITTED`, `FAILED` |
| `deployment_record.status` | Dùng chung deployment lifecycle enum: `AWAITING_CONFIRMATION`, `QUEUED`, `RUNNING`, `SUBMITTED`, `FAILED` |
| `deployment_step.step_name` | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED` |
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` |
| `resource_instance.status` | `PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `RETIRED` |
| `resource_instance.sharing_scope` | `APPLICATION_ENVIRONMENT`, `EXPLICIT_SHARED` |
| `resource_instance_binding.binding_role` | `OWNER`, `SHARED_CONSUMER` |
| `deployment_record.delivery_status` | `NOT_PUBLISHED`, `ACCEPTED`, `SYNCING`, `SYNCED`, `OUT_OF_SYNC`, `DEGRADED`, `FAILED`, `UNKNOWN` |
| `deployment_execution_job.status` | `QUEUED`, `CLAIMED`, `SUCCEEDED`, `FAILED` |

## Deployment lifecycle

`createDeployment()` persist `AWAITING_CONFIRMATION`. `confirmDeployment()` rebuild/check fingerprint và validate override; transaction thành công chuyển lifecycle sang `QUEUED`, tạo/update Deployment Record, tạo đúng ba internal step row `PENDING` và insert durable job/outbox. HTTP trả accepted, không chạy provisioner/CD.

Worker claim job theo lease, chuyển lifecycle sang `RUNNING`, rồi thực thi dài hạn. Nó persist ba progress row nội bộ khi bắt đầu/kết thúc từng phase:

1. `INFRASTRUCTURE_READY`
2. `CONFIGURATION_RESOLVED`
3. `MANIFEST_GENERATED`

Publish được CD chấp nhận chuyển lifecycle sang `SUBMITTED`; lỗi terminal sau khi row deployment tồn tại chuyển sang `FAILED`. `deployment_record.status` phản chiếu platform lifecycle bằng cùng enum. Trạng thái external CD nằm độc lập ở `deployment_record.delivery_status`; các literal `SYNCING`, `DEGRADED`, v.v. không bao giờ được ghi vào Deployment lifecycle.

Failure thuộc infrastructure/configuration/manifest đặt internal step tương ứng `FAILED`. CD publish failure đặt `delivery_status = FAILED` và `deployment_record.error_summary`; không tạo step thứ tư. Vì worker đã atomically claim và ghi lifecycle `RUNNING` trước khi rebuild/verify, `PLAN_STALE_AFTER_ACCEPT` là transition `RUNNING → FAILED`; nó gắn failure vào `INFRASTRUCTURE_READY` trước mọi external side effect. Không có `QUEUED → FAILED` trong flow hiện tại vì chưa đặc tả failure bền vững nào xảy ra trước `markExecutionRunning`.

`CD_SYNCED` và `APPLICATION_READY` là hai marker view-only do UC-04 suy ra live từ CD/Kubernetes khi prerequisite/reference tồn tại. `SYNCED`/all-Healthy thành `SUCCEEDED`; trạng thái đang hội tụ thành `PENDING`; adverse terminal observation thành `FAILED`. Missing future prerequisite trả `PENDING`, terminal failure/non-applicable trả `NOT_AVAILABLE`, không gọi provider. Hai marker không phải `deployment_step` row, không đổi aggregate và UC-04 luôn read-only.

Fingerprint mismatch hoặc invalid override giữ `AWAITING_CONFIRMATION`; không tạo job. Retry cùng idempotency key sau enqueue trả cùng tracking id. Job lease hết hạn có thể claim lại; stable idempotency keys ở provisioner/CD ngăn duplicate side effect.

## Resource Instance lifecycle

Create/update đi qua `PLANNED → PROVISIONING → READY` hoặc `FAILED`; reuse hợp lệ là self-transition ở `READY`; `RETIRED` là terminal cho instance không còn được chọn cho deployment mới. Mọi reuse lookup bắt đầu từ exact active Resource Instance Binding tuple. `EXPLICIT_SHARED` yêu cầu một `SHARED_CONSUMER` binding đã được platform sharing administration pre-authorize, cùng matching sharing key và compatible definition/target; owner columns không phải lookup fallback.

Resource Output chỉ collect sau `READY`. Workload Output không thuộc Resource Instance: Workload Output Resolver tạo giá trị plan-time từ graph/workload metadata/context trước configuration resolution, và từ chối runtime-only/circular dependency.
