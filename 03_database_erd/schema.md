# Step 3: Database / ERD

Schema này hiện thực persistence classification đã được duyệt ở Step 2. Tên table/column dùng `snake_case`; `UUID` dùng cho identity và foreign key; `JSONB` chỉ dùng cho cấu trúc linh hoạt vốn đã là `Map`/`List` trong domain model. Các `TRANSIENT` execution object không được tạo table.

## Application Repository

### `application_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_id` | UUID | PK, NOT NULL | Identity của Application Definition. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE | Tên application. |
| `description` | TEXT | NULL | Mô tả application. |
| `retired_at` | TIMESTAMP | NULL | Soft-delete marker; referenced application definition không bị hard-delete. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `workload`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `workload_id` | UUID | PK, NOT NULL | Identity của Workload. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu workload. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_id`, `name`) | Tên workload, duy nhất trong application. |
| `type` | VARCHAR(100) | NOT NULL | Loại workload. |
| `image_repository` | VARCHAR(1024) | NOT NULL | Image repository; không chứa deployment-time image version. |
| `port` | INT | NULL, CHECK (`port` BETWEEN 1 AND 65535) | Application port nếu có. |
| `exposed_outputs` | JSONB | NOT NULL | Output definitions gồm name, availability (`PLAN_TIME`/`RUNTIME`) và resolution metadata. Same-deployment binding chỉ nhận `PLAN_TIME`. |
| `retired_at` | TIMESTAMP | NULL | Soft-delete marker; active query lọc `NULL`, history vẫn resolve được FK. |

### `resource_requirement`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `resource_requirement_id` | UUID | PK, NOT NULL | Identity của logical Resource Requirement. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu requirement. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_id`, `name`) | Tên resource logic trong application. |
| `resource_type` | VARCHAR(100) | NOT NULL | Loại resource, ví dụ PostgreSQL hoặc Redis. |
| `retired_at` | TIMESTAMP | NULL | Retire marker khi requirement đã từng được tham chiếu. |

### `environment_variable_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `variable_definition_id` | UUID | PK, NOT NULL | Identity của Environment Variable Definition. |
| `workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Workload cần variable này. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`workload_id`, `name`) | Tên variable trong workload. |
| `required` | BOOLEAN | NOT NULL | Variable có bắt buộc được configure hay không. |
| `retired_at` | TIMESTAMP | NULL | Retire marker; không phá binding/history hiện có. |

### `secret_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `secret_definition_id` | UUID | PK, NOT NULL | Identity của Secret Definition. |
| `workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Workload cần secret này. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`workload_id`, `name`) | Tên secret trong workload. |
| `required` | BOOLEAN | NOT NULL | Secret có bắt buộc được configure hay không. |
| `retired_at` | TIMESTAMP | NULL | Retire marker; không phá binding/history hiện có. |

Các FK lịch sử/configuration từ `deployment`, `workload_deployment`, `environment_variable`, `secret`, `configuration_value`, `dependency`, `resource_instance` và `resource_instance_binding` tới Application/Workload/definition/requirement dùng `ON DELETE RESTRICT`; FK `resource_instance.resource_definition_id` cũng `RESTRICT`. Item đã từng được tham chiếu phải set `retired_at`; hard delete chỉ được phép cho item chưa từng có reference và vẫn phải qua kiểm tra FK trong transaction. Query tạo deployment/specification mới lọc `retired_at IS NULL`, còn history query không lọc mất target đã retire.

### `dependency`

`dependency` chính là persistent association cho quan hệ source component → target component; không cần thêm domain entity khác cho many-to-many topology.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `dependency_id` | UUID | PK, NOT NULL | Identity của Dependency. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu dependency. |
| `source_workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Workload ở đầu source của quan hệ `depends on`. |
| `target_type` | ENUM (`WORKLOAD`, `RESOURCE`) | NOT NULL | Discriminator cho loại target. |
| `target_workload_id` | UUID | FK → `workload.workload_id`, NULL | Target khi `target_type = WORKLOAD`. |
| `target_resource_requirement_id` | UUID | FK → `resource_requirement.resource_requirement_id`, NULL | Target khi `target_type = RESOURCE`. |

Constraint bổ sung: `CHECK` bắt buộc đúng một target FK được đặt theo `target_type`; source và target phải thuộc cùng `application_id`. Nên đặt `UNIQUE` theo source + target logic để không lưu dependency trùng.

## Specification Repository / Config Repo Service

### `application_specification`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `specification_id` | UUID | PK, NOT NULL | Identity của generated Application Specification. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL, UNIQUE | Application nguồn; bảo toàn cardinality tối đa một current specification/application. |
| `format` | VARCHAR(50) | NOT NULL | Format artifact, ví dụ `score-yaml`. |
| `content` | TEXT | NOT NULL | Nội dung specification chưa resolve. |
| `version` | VARCHAR(100) | NOT NULL | Version của artifact. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm sinh/cập nhật gần nhất. |

## Environment Configuration Repository

### `environment_configuration`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `environment_configuration_id` | UUID | PK, NOT NULL | Identity của Environment Configuration. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application được configure. |
| `environment` | VARCHAR(100) | NOT NULL, UNIQUE (`application_id`, `environment`) | Environment; mỗi application có tối đa một configuration cho một environment. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `environment_variable`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `environment_variable_id` | UUID | PK, NOT NULL | Identity của configured Environment Variable. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Configuration theo environment sở hữu binding. |
| `workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Workload nhận variable. |
| `variable_definition_id` | UUID | FK → `environment_variable_definition.variable_definition_id`, NOT NULL | Requirement được binding. |
| `variable_name` | VARCHAR(255) | NOT NULL | Snapshot logical name của variable. |

Constraint bổ sung: `UNIQUE (environment_configuration_id, variable_definition_id)`; `workload_id` phải trùng workload sở hữu `variable_definition_id` và thuộc application của configuration.

### `configuration_value`

Table này lưu `Configuration Value` và payload của ba persistent subtype value object bằng chiến lược embedded/table-per-hierarchy: `Direct Configuration Value`, `Resource Output Reference`, `Workload Output Reference`.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `configuration_value_id` | UUID | PK, NOT NULL | Identity persistence của Configuration Value. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Configuration sở hữu value source; biểu diễn trực tiếp quan hệ 1:N. |
| `environment_variable_id` | UUID | FK → `environment_variable.environment_variable_id`, NOT NULL, UNIQUE | Mỗi Environment Variable có đúng một Configuration Value. |
| `value_source` | ENUM (`DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT`) | NOT NULL | Discriminator của value object subtype. |
| `direct_value` | TEXT | NULL | Payload của Direct Configuration Value; chỉ dành cho non-secret Environment Variable. |
| `resource_requirement_id` | UUID | FK → `resource_requirement.resource_requirement_id`, NULL | Logical resource được tham chiếu; tên resource lấy qua FK, không phải runtime instance. |
| `resource_output_name` | VARCHAR(255) | NULL | Tên output logic của resource, ví dụ `host`. |
| `workload_id` | UUID | FK → `workload.workload_id`, NULL | Logical workload được tham chiếu. |
| `workload_output_name` | VARCHAR(255) | NULL | Tên output logic của workload, ví dụ `endpoint`. |

`CHECK` theo `value_source`:

- `DIRECT`: chỉ `direct_value` được phép có giá trị.
- `RESOURCE_OUTPUT`: bắt buộc `resource_requirement_id` + `resource_output_name`; các payload khác phải `NULL`.
- `WORKLOAD_OUTPUT`: bắt buộc `workload_id` + `workload_output_name`; các payload khác phải `NULL`.

`environment_configuration_id` phải trùng configuration sở hữu `environment_variable_id`; constraint này được enforce bằng composite FK hoặc aggregate transaction.

### `secret`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `secret_id` | UUID | PK, NOT NULL | Identity của configured Secret. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Configuration theo environment sở hữu secret binding. |
| `workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Workload nhận secret. |
| `secret_definition_id` | UUID | FK → `secret_definition.secret_definition_id`, NOT NULL | Secret requirement được binding. |
| `secret_name` | VARCHAR(255) | NOT NULL | Snapshot logical name của Secret. |
| `value_source` | ENUM (`SECRET_REF`, `RESOURCE_OUTPUT`) | NOT NULL | Secret đến từ Secret Store reference hoặc sensitive Resource Output Reference. |
| `secret_ref` | VARCHAR(2048) | NULL | Opaque reference/URI do Secret Store trả về; không phải secret value. |
| `resource_requirement_id` | UUID | FK → `resource_requirement.resource_requirement_id`, NULL | Logical resource khi secret bind tới sensitive output. |
| `resource_output_name` | VARCHAR(255) | NULL | Tên sensitive output logic. |

Constraint bổ sung: `UNIQUE (environment_configuration_id, secret_definition_id)`. `CHECK` bắt buộc đúng một source: `SECRET_REF` chỉ có `secret_ref`; `RESOURCE_OUTPUT` chỉ có `resource_requirement_id` + `resource_output_name`. `workload_id` phải trùng workload sở hữu `secret_definition_id`.

## Platform Resource Definition Catalog

`Resource Definition` là persistent reference data theo Step 2 nhưng được platform quản lý, nằm ngoài năm application-lifecycle repository. Table vẫn phải có trong ERD để không làm mất persistent domain object.

### `resource_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `resource_definition_id` | UUID | PK, NOT NULL | Identity của Resource Definition. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE | Tên catalog definition. |
| `resource_type` | VARCHAR(100) | NOT NULL | Loại logical resource được definition hỗ trợ. |
| `provisioner_reference` | VARCHAR(2048) | NOT NULL | Reference tới provisioner/module; không phải execution result. |
| `supported_contexts` | JSONB | NOT NULL | Các deployment context được hỗ trợ. |
| `default_parameters` | JSONB | NOT NULL | Giá trị mặc định và schema của các provisioning parameter do definition cung cấp, ví dụ default cho `instanceClass` và `storageGb`; là nguồn bền vững để plan resolve parameter theo deployment context. |
| `allowed_overrides` | JSONB | NOT NULL | Policy override do definition cung cấp: các parameter key Developer được phép override cùng tập giá trị, khoảng min-max hoặc enum hợp lệ. |
| `exposed_outputs` | JSONB | NOT NULL | Danh sách normal outputs hợp lệ. |
| `sensitive_outputs` | JSONB | NOT NULL | Danh sách sensitive outputs hợp lệ. |
| `retired_at` | TIMESTAMP | NULL | Catalog definition đã có Resource Instance/history chỉ được retire. |

## Resource Instance Repository

### `resource_instance`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `resource_instance_id` | UUID | PK, NOT NULL | Identity bền vững của provisioned infrastructure. |
| `resource_definition_id` | UUID | FK → `resource_definition.resource_definition_id`, NOT NULL | Catalog definition đã dùng để provision. |
| `owner_application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application tạo/sở hữu gốc; dùng cho ownership/audit, không phải read path thay thế cho active binding. |
| `environment` | VARCHAR(100) | NOT NULL | Environment ownership của instance. |
| `resource_requirement_id` | UUID | FK → `resource_requirement.resource_requirement_id`, NOT NULL | Logical requirement cụ thể mà instance thực hiện. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Target nơi resource được quản lý. |
| `sharing_scope` | ENUM (`APPLICATION_ENVIRONMENT`, `EXPLICIT_SHARED`) | NOT NULL, DEFAULT `APPLICATION_ENVIRONMENT` | Policy scope; mặc định cấm reuse chéo application/environment. |
| `sharing_key` | VARCHAR(255) | NULL | Bắt buộc chỉ khi `EXPLICIT_SHARED`; phải match policy và query. |
| `infrastructure_reference` | VARCHAR(2048) | NULL, UNIQUE khi khác NULL | Durable provider identity; bắt buộc khi status `READY`, có thể chưa có khi `PLANNED/PROVISIONING/FAILED`. |
| `provider_state_reference` | VARCHAR(2048) | NULL | Reference tới provider state nếu có. |
| `status` | ENUM (`PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `RETIRED`) | NOT NULL | Lifecycle status vật lý của Resource Instance. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

`CHECK` yêu cầu `sharing_scope = APPLICATION_ENVIRONMENT` thì `sharing_key IS NULL`, còn `EXPLICIT_SHARED` thì `sharing_key IS NOT NULL`; `status = READY` bắt buộc có `infrastructure_reference`. Các owner columns trên row này chỉ ghi ownership gốc và phục vụ integrity/audit. **Mọi reusable-instance lookup canonical luôn bắt đầu từ exact active tuple trên `resource_instance_binding`; không được query owner columns như một read path thay thế.** Sau khi join binding, repository mới lọc instance `READY`, definition/target compatibility và sharing policy. Không có table `resource_output`: output được collector nạp transient sau khi instance ready.

### `resource_instance_binding`

Association bền vững buộc mọi lookup/reuse vào đúng logical consumer scope; owner columns trên `resource_instance` chỉ là original creator ownership.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `resource_instance_binding_id` | UUID | PK, NOT NULL | Identity kỹ thuật. |
| `resource_instance_id` | UUID | FK → `resource_instance.resource_instance_id`, NOT NULL | Instance được binding. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Consumer application. |
| `environment` | VARCHAR(100) | NOT NULL | Consumer environment. |
| `resource_requirement_id` | UUID | FK → `resource_requirement.resource_requirement_id`, NOT NULL | Logical requirement của consumer. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Consumer target; phải trùng instance target. |
| `binding_role` | ENUM (`OWNER`, `SHARED_CONSUMER`) | NOT NULL | Owner binding hoặc explicit shared consumer. |
| `sharing_key` | VARCHAR(255) | NULL | Chỉ có cho shared consumer và phải trùng instance key. |
| `created_at` | TIMESTAMP | NOT NULL | Audit timestamp. |
| `retired_at` | TIMESTAMP | NULL | Binding hết hiệu lực nhưng được giữ cho audit/history. |

Partial `UNIQUE (application_id, environment, resource_requirement_id, deployment_target) WHERE retired_at IS NULL` bảo đảm một logical scope không match nhiều active instance nhưng cho phép replacement sau retire. Mỗi instance có đúng một active `OWNER` binding trùng owner tuple. `SHARED_CONSUMER` chỉ được platform sharing administration (ngoài bốn Developer UC) pre-authorize/insert khi instance `EXPLICIT_SHARED`, `READY`, sharing key, authorization và definition compatibility đều khớp. Repository reuse query luôn bắt đầu bằng exact active binding tuple và không tự mở rộng sharing scope trong deployment flow.

## Deployment Repository

### `deployment`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_id` | UUID | PK, NOT NULL | Identity của một lần deployment. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application được deploy. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Environment Configuration được dùng làm source references. |
| `environment` | VARCHAR(100) | NOT NULL | Environment snapshot của deployment. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Kubernetes/deployment target đã chọn. |
| `plan_fingerprint` | CHAR(64) | NOT NULL | SHA-256 fingerprint dạng hex của Infrastructure Plan đã canonicalize để phát hiện plan thay đổi khi confirm; không chứa chính plan. |
| `plan_fingerprint_algo` | VARCHAR(32) | NOT NULL | Phiên bản thuật toán canonicalization + hashing của fingerprint, ví dụ `sha256-v1`, dùng lại khi rebuild plan. |
| `status` | ENUM (`AWAITING_CONFIRMATION`, `QUEUED`, `RUNNING`, `SUBMITTED`, `FAILED`) | NOT NULL | Platform lifecycle; không chứa external CD status. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo deployment. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `workload_deployment`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `workload_deployment_id` | UUID | PK, NOT NULL | Identity của workload snapshot trong deployment. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL | Deployment sở hữu snapshot. |
| `workload_id` | UUID | FK → `workload.workload_id`, NOT NULL | Logical workload được deploy. |
| `image_repository` | VARCHAR(1024) | NOT NULL | Image repository thực tế. |
| `image_version` | VARCHAR(255) | NOT NULL | Image tag/version thực tế do Developer hoặc CI cung cấp. |

Constraint bổ sung: `UNIQUE (deployment_id, workload_id)`; một Deployment phải có ít nhất một Workload Deployment (aggregate invariant, enforce ở transaction/service hoặc deferred database constraint).

### `deployment_context`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_context_id` | UUID | PK, NOT NULL | Persistence identity của Deployment Context value object. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL, UNIQUE | Bảo đảm đúng một context cho mỗi deployment. |
| `cloud_provider` | VARCHAR(100) | NULL | Cloud provider nếu target yêu cầu. |
| `region` | VARCHAR(100) | NULL | Region nếu target yêu cầu. |
| `target_specific_input` | JSONB | NOT NULL | Target-specific input có cấu trúc linh hoạt. |

### `deployment_record`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_record_id` | UUID | PK, NOT NULL | Identity của durable Deployment Record. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL, UNIQUE | Một deployment sinh tối đa một record. |
| `environment` | VARCHAR(100) | NOT NULL | Environment được ghi nhận cho history. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Target thực tế được ghi nhận. |
| `delivery_reference` | VARCHAR(2048) | NULL | Reference tới desired state/CD delivery nếu có. |
| `status` | ENUM (`AWAITING_CONFIRMATION`, `QUEUED`, `RUNNING`, `SUBMITTED`, `FAILED`) | NOT NULL | Snapshot platform lifecycle, dùng cùng physical enum với `deployment.status`. |
| `delivery_status` | ENUM (`NOT_PUBLISHED`, `ACCEPTED`, `SYNCING`, `SYNCED`, `OUT_OF_SYNC`, `DEGRADED`, `FAILED`, `UNKNOWN`) | NOT NULL, DEFAULT `NOT_PUBLISHED` | External CD delivery status riêng; không được copy vào lifecycle. |
| `error_summary` | TEXT | NULL | Lỗi tổng quát nếu deployment thất bại. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo record. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật record. |

`infrastructureReferences` trong domain được chuẩn hóa qua association table `deployment_record_resource_instance`, thay vì lưu resolved output hay blob không có referential integrity.

### `deployment_step`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_step_id` | UUID | PK, NOT NULL | Identity của Deployment Step. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL | Deployment sở hữu step; biểu diễn trực tiếp cardinality 1:N. |
| `deployment_record_id` | UUID | FK → `deployment_record.deployment_record_id`, NOT NULL | Deployment Record sở hữu step. Qua quan hệ 1:1 record–deployment, một Deployment có nhiều step. |
| `sequence_number` | INT | NOT NULL, UNIQUE (`deployment_record_id`, `sequence_number`) | Thứ tự step trong deployment. |
| `step_name` | ENUM (`INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`) | NOT NULL | Chỉ ba progress step nội bộ được persist. |
| `status` | ENUM (`PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED`) | NOT NULL | Physical step status. |
| `related_component_reference` | VARCHAR(2048) | NULL | Workload/resource reference liên quan. |
| `error_summary` | TEXT | NULL | Chi tiết lỗi của step nếu có. |
| `started_at` | TIMESTAMP | NULL | Thời điểm bắt đầu. |
| `completed_at` | TIMESTAMP | NULL | Thời điểm hoàn tất. |

`deployment_id` phải trùng deployment của `deployment_record_id`; constraint này được enforce bằng composite FK. `UNIQUE (deployment_id, step_name)` bảo đảm mỗi internal step có một row để worker upsert. `CD_SYNCED` và `APPLICATION_READY` không thuộc enum này; UC-04 suy ra chúng live từ CD/Kubernetes.

### `deployment_record_resource_instance`

Đây là relational association table cho quan hệ persistent many-to-many `Deployment Record` ↔ `Resource Instance`; nó không phải một domain object mới.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_record_resource_instance_id` | UUID | PK, NOT NULL | Identity kỹ thuật của association row. |
| `deployment_record_id` | UUID | FK → `deployment_record.deployment_record_id`, NOT NULL | Deployment Record tham chiếu infrastructure. |
| `resource_instance_id` | UUID | FK → `resource_instance.resource_instance_id`, NOT NULL | Durable Resource Instance được create/update/reuse. |

Constraint bổ sung: `UNIQUE (deployment_record_id, resource_instance_id)`.

Worker insert association rows ngay khi `INFRASTRUCTURE_READY` thành công (cùng transaction cập nhật step), không đợi CD publish; nhờ đó UC-04 có thể đọc infrastructure status trong khi lifecycle còn `RUNNING`.

### `deployment_execution_job`

Durable execution job đồng thời là transactional outbox; không chứa Infrastructure Plan payload.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `job_id` | UUID | PK, NOT NULL | Tracking identity trả về bởi confirm. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL, UNIQUE | Tối đa một execution job cho deployment. |
| `idempotency_key` | VARCHAR(255) | NOT NULL, UNIQUE (`deployment_id`, `idempotency_key`) | Cùng deployment/key trả cùng tracking result, không xung đột key của deployment khác. |
| `override_values` | JSONB | NOT NULL | Selected override values tối thiểu cho worker; không phải plan/item/definition payload. |
| `accepted_plan_fingerprint` | CHAR(64) | NOT NULL | Fingerprint worker phải verify sau khi rebuild. |
| `status` | ENUM (`QUEUED`, `CLAIMED`, `SUCCEEDED`, `FAILED`) | NOT NULL | Job/outbox status vật lý. |
| `attempts` | INT | NOT NULL, DEFAULT 0, CHECK (`attempts >= 0`) | Số lần claim. |
| `max_attempts` | INT | NOT NULL, CHECK (`max_attempts > 0`) | Retry bound. |
| `available_at` | TIMESTAMP | NOT NULL | Backoff scheduling. |
| `lease_until` | TIMESTAMP | NULL | Lease reclaim nếu worker chết. |
| `last_error` | TEXT | NULL | Lỗi retry/terminal, đã redact secret. |
| `created_at`, `updated_at` | TIMESTAMP | NOT NULL | Audit timestamps. |

`confirmDeployment()` thực hiện CAS `AWAITING_CONFIRMATION → QUEUED`, tạo/cập nhật `deployment_record.status = QUEUED`, tạo đúng ba step row `PENDING`, và insert job trong **một DB transaction**. Worker claim bằng row lock/`SKIP LOCKED` hoặc CAS lease; provider/CD calls dùng stable idempotency key theo deployment + phase/item. Khi hết retry, worker persist job/deployment/record `FAILED` và failed internal step atomically trong DB.

## Enforcement của persistence constraints chính

### Reference, không phải resolved runtime value

- `configuration_value.value_source` quyết định payload hợp lệ bằng `CHECK`; output-based rows chỉ lưu logical FK + output name.
- Resource binding lưu `resource_requirement_id` + `resource_output_name`; workload binding lưu `workload_id` + `workload_output_name`. Tên component được lấy từ table target, tránh denormalized name không có referential integrity.
- Schema không có `resolved_value`, resolved host/port/endpoint/credential trong `configuration_value`, `secret`, `deployment` hoặc `deployment_record`.
- Không tạo table cho `resource_output`, `resolved_configuration` hay `resolved_specification`; các giá trị này chỉ tồn tại trong deployment execution.
- Infrastructure Plan cũng giữ nguyên là `TRANSIENT` và được rebuild khi confirm; `deployment` chỉ persist `plan_fingerprint` cùng `plan_fingerprint_algo`, không persist plan hoặc resolved plan payload.

### Secret reference, không phải plaintext

- `secret` hoàn toàn không có column plaintext/value. Direct secret đi qua Secret Store và DB chỉ nhận `secret_ref` opaque.
- Staging và promotion giữ cùng stable opaque reference; `stageSecret` dùng idempotency key + TTL. DB save failure chỉ revoke newly staged refs, còn DB success phải nhận promote acknowledgement (hoặc idempotent retry/reconciliation) trước khi báo thành công.
- `CHECK` trên `secret.value_source` bảo đảm `SECRET_REF` và `RESOURCE_OUTPUT` loại trừ lẫn nhau.
- Với sensitive resource output, DB chỉ lưu `resource_requirement_id` + `resource_output_name`; resolved secret value không được persist.
- `configuration_value.direct_value` chỉ thuộc Environment Variable path vì table liên kết bắt buộc tới `environment_variable`, không thể liên kết tới `secret`.

## Mapping domain → table

### PERSISTENT objects

| Domain object | Table / persistence mapping |
|---|---|
| Application Definition | `application_definition` |
| Workload | `workload` |
| Workload Output Definition | Embedded JSONB items in `workload.exposed_outputs` |
| Resource Requirement | `resource_requirement` |
| Environment Variable Definition | `environment_variable_definition` |
| Secret Definition | `secret_definition` |
| Dependency | `dependency` |
| Application Specification | `application_specification` |
| Environment Configuration | `environment_configuration` |
| Environment Variable | `environment_variable` |
| Secret | `secret` |
| Configuration Value | `configuration_value` |
| Direct Configuration Value | Embedded subtype payload `configuration_value.direct_value` when `value_source = DIRECT` |
| Resource Output Reference | Embedded reference fields in `configuration_value`, or in `secret` for sensitive output binding |
| Workload Output Reference | Embedded fields `configuration_value.workload_id` + `workload_output_name` |
| Resource Definition | `resource_definition` (platform-managed catalog) |
| Deployment | `deployment` |
| Workload Deployment | `workload_deployment` |
| Deployment Context | `deployment_context` |
| Resource Instance | `resource_instance` |
| Resource Instance Binding | `resource_instance_binding` |
| Deployment Record | `deployment_record` |
| Deployment Step | `deployment_step` |
| Deployment Execution Job | `deployment_execution_job` |

`deployment_record_resource_instance` chỉ hiện thực quan hệ many-to-many đã có trong domain model; không giới thiệu domain entity mới.

### TRANSIENT objects intentionally excluded

| Domain object | Lý do không tạo table |
|---|---|
| Deployment Graph | Dựng lại cho từng execution từ persistent sources. |
| Resource Resolution | Quyết định trung gian; durable outcome là Resource Instance/reference. |
| Infrastructure Plan / Item / Override Definition | Typed plan được rebuild khi confirm và worker execute; chỉ SHA-256 fingerprint + algorithm persist trên `deployment`. Selected override values riêng nằm trong job, không phải plan payload. |
| Resource Output | Runtime output được collector nạp in-memory sau khi resource ready. |
| Workload Output | Plan-time output được Workload Output Resolver tính từ graph/metadata/context; runtime-only/circular output bị từ chối. |
| Resolved Configuration | In-memory snapshot của values đã resolve. |
| Resolved Specification | Execution artifact dùng làm input cho `score-k8s`, không phải versioned source specification. |
