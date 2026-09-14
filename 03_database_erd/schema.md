# Step 3: Database / ERD

Schema này hiện thực persistence classification đã được duyệt ở Step 2. Tên table/column dùng `snake_case`; `UUID` dùng cho identity và foreign key; `JSONB` chỉ dùng cho cấu trúc linh hoạt vốn đã là `Map`/`List` trong domain model. Các `TRANSIENT` execution object không được tạo table.

## Application Repository

### `application_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_id` | UUID | PK, NOT NULL | Identity của Application Definition. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE | Tên application. |
| `description` | TEXT | NULL | Mô tả application. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `application_definition_version`

Mỗi lần Save ở UC-01 tạo một dòng mới. Dòng đã tạo không bị `UPDATE` hay `DELETE`; mọi FK trỏ tới bảng này dùng `ON DELETE RESTRICT`.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_definition_version_id` | UUID | PK, NOT NULL | Identity của một phiên bản. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu phiên bản. |
| `version_number` | INT | NOT NULL, UNIQUE (`application_id`, `version_number`) | Số thứ tự phiên bản trong application, tăng dần. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo phiên bản. |

### `application_component`

Identity table hiện thực **ID cố định qua các phiên bản** của Workload, Resource Requirement, Environment Variable Definition và Secret Definition. Đây không phải domain object mới: mỗi dòng chỉ giữ ID logic và loại thành phần; thuộc tính theo phiên bản nằm ở các bảng bên dưới. Các bảng nằm ngoài phiên bản (configuration, Resource/Workload Instance, workload deployment) tham chiếu ID logic qua bảng này nên vẫn đúng khi thành phần đổi tên hoặc không còn trong phiên bản mới.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `component_id` | UUID | PK, NOT NULL | ID logic cố định qua các phiên bản. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu thành phần. |
| `component_type` | ENUM (`WORKLOAD`, `RESOURCE_REQUIREMENT`, `ENVIRONMENT_VARIABLE_DEFINITION`, `SECRET_DEFINITION`) | NOT NULL, UNIQUE (`component_id`, `component_type`) | Loại thành phần; cặp (`component_id`, `component_type`) cho phép composite FK kiểm tra đúng loại. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm thành phần xuất hiện lần đầu. |

### `workload`

Mỗi dòng là một Workload trong một phiên bản. Khóa chính: (`application_definition_version_id`, `workload_id`).

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_definition_version_id` | UUID | PK, FK → `application_definition_version.application_definition_version_id`, NOT NULL | Phiên bản chứa workload. |
| `workload_id` | UUID | PK, FK → `application_component.component_id` (loại `WORKLOAD`), NOT NULL | ID logic cố định của workload qua các phiên bản. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_definition_version_id`, `name`) | Tên workload, duy nhất trong phiên bản. |
| `type` | VARCHAR(100) | NOT NULL | Loại workload. |
| `image_repository` | VARCHAR(1024) | NOT NULL | Image repository; không chứa deployment-time image version. |
| `port` | INT | NULL, CHECK (`port` BETWEEN 1 AND 65535) | Application port nếu có. |
| `exposed_outputs` | JSONB | NOT NULL | Danh sách logical outputs workload công bố, ví dụ `endpoint`. |

### `resource_requirement`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_definition_version_id` | UUID | PK, FK → `application_definition_version.application_definition_version_id`, NOT NULL | Phiên bản chứa requirement. |
| `resource_requirement_id` | UUID | PK, FK → `application_component.component_id` (loại `RESOURCE_REQUIREMENT`), NOT NULL | ID logic cố định; là một phần của khóa chủ sở hữu Resource Instance. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_definition_version_id`, `name`) | Tên resource logic trong phiên bản. |
| `resource_type` | VARCHAR(100) | NOT NULL | Loại resource, ví dụ PostgreSQL hoặc Redis. |

Khóa chính: (`application_definition_version_id`, `resource_requirement_id`).

### `environment_variable_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_definition_version_id` | UUID | PK, NOT NULL | Phiên bản chứa definition. |
| `variable_definition_id` | UUID | PK, FK → `application_component.component_id` (loại `ENVIRONMENT_VARIABLE_DEFINITION`), NOT NULL | ID logic cố định của Environment Variable Definition. |
| `workload_id` | UUID | NOT NULL; FK (`application_definition_version_id`, `workload_id`) → `workload` | Workload cần variable này trong cùng phiên bản. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_definition_version_id`, `workload_id`, `name`) | Tên variable trong workload. |
| `required` | BOOLEAN | NOT NULL | Variable có bắt buộc được configure hay không. |

Khóa chính: (`application_definition_version_id`, `variable_definition_id`).

### `secret_definition`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `application_definition_version_id` | UUID | PK, NOT NULL | Phiên bản chứa definition. |
| `secret_definition_id` | UUID | PK, FK → `application_component.component_id` (loại `SECRET_DEFINITION`), NOT NULL | ID logic cố định của Secret Definition. |
| `workload_id` | UUID | NOT NULL; FK (`application_definition_version_id`, `workload_id`) → `workload` | Workload cần secret này trong cùng phiên bản. |
| `name` | VARCHAR(255) | NOT NULL, UNIQUE (`application_definition_version_id`, `workload_id`, `name`) | Tên secret trong workload. |
| `required` | BOOLEAN | NOT NULL | Secret có bắt buộc được configure hay không. |

Khóa chính: (`application_definition_version_id`, `secret_definition_id`).

### `dependency`

`dependency` chính là persistent association cho quan hệ source component → target component trong một phiên bản; không cần thêm domain entity khác cho many-to-many topology.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `dependency_id` | UUID | PK, NOT NULL | Identity của Dependency. |
| `application_definition_version_id` | UUID | FK → `application_definition_version.application_definition_version_id`, NOT NULL | Phiên bản chứa dependency. |
| `source_workload_id` | UUID | NOT NULL; FK (`application_definition_version_id`, `source_workload_id`) → `workload` | Workload ở đầu source của quan hệ `depends on`. |
| `target_type` | ENUM (`WORKLOAD`, `RESOURCE`) | NOT NULL | Discriminator cho loại target. |
| `target_workload_id` | UUID | NULL; FK (`application_definition_version_id`, `target_workload_id`) → `workload` | Target khi `target_type = WORKLOAD`. |
| `target_resource_requirement_id` | UUID | NULL; FK (`application_definition_version_id`, `target_resource_requirement_id`) → `resource_requirement` | Target khi `target_type = RESOURCE`. |

Constraint bổ sung: `CHECK` bắt buộc đúng một target FK được đặt theo `target_type`; source và target thuộc cùng phiên bản (bảo đảm bằng composite FK). Nên đặt `UNIQUE` theo phiên bản + source + target để không lưu dependency trùng. Việc các dependency của một phiên bản không tạo thành vòng được kiểm tra ở tầng service khi lưu phiên bản (không biểu diễn được bằng constraint quan hệ).

Các bảng trong Application Repository chỉ được `INSERT` khi tạo phiên bản mới; không `UPDATE` hay `DELETE` dòng của phiên bản đã lưu.

## Specification Repository / Config Repo Service

### `application_specification`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `specification_id` | UUID | PK, NOT NULL | Identity của generated Application Specification. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application nguồn. |
| `application_definition_version_id` | UUID | FK → `application_definition_version.application_definition_version_id`, NOT NULL, UNIQUE | Phiên bản được dùng để sinh specification; mỗi phiên bản có tối đa một specification. |
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
| `environment` | ENUM (`STAGING`, `PRODUCTION`) | NOT NULL, UNIQUE (`application_id`, `environment`) | Environment; mỗi application có tối đa một configuration cho mỗi environment. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `environment_variable`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `environment_variable_id` | UUID | PK, NOT NULL | Identity của configured Environment Variable. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Configuration theo environment sở hữu binding. |
| `workload_id` | UUID | FK → `application_component.component_id` (loại `WORKLOAD`), NOT NULL | ID cố định của workload nhận variable. |
| `variable_definition_id` | UUID | FK → `application_component.component_id` (loại `ENVIRONMENT_VARIABLE_DEFINITION`), NOT NULL | ID cố định của requirement được binding. |
| `variable_name` | VARCHAR(255) | NOT NULL | Snapshot logical name của variable. |

Constraint bổ sung: `UNIQUE (environment_configuration_id, variable_definition_id)`; `workload_id` và `variable_definition_id` thuộc application của configuration. Configuration không có phiên bản: việc `workload_id` sở hữu `variable_definition_id` được kiểm tra ở tầng service theo phiên bản mới nhất khi lưu và theo phiên bản được deploy khi deploy.

### `configuration_value`

Table này lưu `Configuration Value` và payload của ba persistent subtype value object bằng chiến lược embedded/table-per-hierarchy: `Direct Configuration Value`, `Resource Output Reference`, `Workload Output Reference`.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `configuration_value_id` | UUID | PK, NOT NULL | Identity persistence của Configuration Value. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Configuration sở hữu value source; biểu diễn trực tiếp quan hệ 1:N. |
| `environment_variable_id` | UUID | FK → `environment_variable.environment_variable_id`, NOT NULL, UNIQUE | Mỗi Environment Variable có đúng một Configuration Value. |
| `value_source` | ENUM (`DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT`) | NOT NULL | Discriminator của value object subtype. |
| `direct_value` | TEXT | NULL | Payload của Direct Configuration Value; chỉ dành cho non-secret Environment Variable. |
| `resource_requirement_id` | UUID | FK → `application_component.component_id` (loại `RESOURCE_REQUIREMENT`), NULL | ID cố định của logical resource được tham chiếu; không phải runtime instance. |
| `resource_output_name` | VARCHAR(255) | NULL | Tên output logic của resource, ví dụ `host`. |
| `workload_id` | UUID | FK → `application_component.component_id` (loại `WORKLOAD`), NULL | ID cố định của logical workload được tham chiếu. |
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
| `workload_id` | UUID | FK → `application_component.component_id` (loại `WORKLOAD`), NOT NULL | ID cố định của workload nhận secret. |
| `secret_definition_id` | UUID | FK → `application_component.component_id` (loại `SECRET_DEFINITION`), NOT NULL | ID cố định của Secret requirement được binding. |
| `secret_name` | VARCHAR(255) | NOT NULL | Snapshot logical name của Secret. |
| `value_source` | ENUM (`SECRET_REF`, `RESOURCE_OUTPUT`) | NOT NULL | Secret đến từ Secret Store reference hoặc sensitive Resource Output Reference. |
| `secret_ref` | VARCHAR(2048) | NULL | Opaque reference/URI do Secret Store trả về; không phải secret value. |
| `resource_requirement_id` | UUID | FK → `application_component.component_id` (loại `RESOURCE_REQUIREMENT`), NULL | ID cố định của logical resource khi secret bind tới sensitive output. |
| `resource_output_name` | VARCHAR(255) | NULL | Tên sensitive output logic. |

Constraint bổ sung: `UNIQUE (environment_configuration_id, secret_definition_id)`. `CHECK` bắt buộc đúng một source: `SECRET_REF` chỉ có `secret_ref`; `RESOURCE_OUTPUT` chỉ có `resource_requirement_id` + `resource_output_name`. Việc `workload_id` sở hữu `secret_definition_id`, và resource được tham chiếu là thành phần mà workload depends on, được kiểm tra ở tầng service theo phiên bản (mới nhất khi lưu, được deploy khi deploy).

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
| `management_mode` | ENUM (`MANAGED`, `EXISTING`) | NOT NULL | `MANAGED`: IDP tạo/sửa/hủy hạ tầng qua provisioner. `EXISTING`: definition trỏ tới resource dùng chung có sẵn; IDP chỉ đọc output, không đụng hạ tầng thật. |
| `applicability_conditions` | JSONB | NULL | Điều kiện application/environment được áp dụng definition, dùng để khai báo tường minh việc dùng chung (ví dụ mọi application ở `STAGING`). |
| `existing_resource_reference` | VARCHAR(2048) | NULL | Reference tới resource có sẵn khi `management_mode = EXISTING`. |

Constraint bổ sung: `CHECK` khi `management_mode = EXISTING` thì `existing_resource_reference` khác `NULL`.

## Resource Instance Repository

### `resource_instance`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `resource_instance_id` | UUID | PK, NOT NULL | Identity bền vững của Resource Instance. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application chủ sở hữu. |
| `environment` | ENUM (`STAGING`, `PRODUCTION`) | NOT NULL | Environment chủ sở hữu. |
| `resource_requirement_id` | UUID | FK → `application_component.component_id` (loại `RESOURCE_REQUIREMENT`), NOT NULL | ID cố định của Resource Requirement chủ sở hữu. |
| `resource_definition_id` | UUID | FK → `resource_definition.resource_definition_id`, NOT NULL | Catalog definition đã dùng để provision hoặc liên kết. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Target nơi resource được quản lý. |
| `infrastructure_reference` | VARCHAR(2048) | NOT NULL | Durable provider/infrastructure identity. Không `UNIQUE`: nhiều instance loại `EXISTING` của các chủ sở hữu khác nhau có thể trỏ cùng một resource dùng chung. |
| `provider_state_reference` | VARCHAR(2048) | NULL | Reference tới provider state nếu có. |
| `status` | ENUM (`PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `DESTROYED`, `UNLINKED`) | NOT NULL | Lifecycle status của Resource Instance. `DESTROYED`/`UNLINKED` là trạng thái kết thúc; dòng được giữ lại cho lịch sử. |
| `output_fingerprint` | CHAR(64) | NULL | SHA-256 hex của output lần gần nhất, dùng để phát hiện output thay đổi; không lưu giá trị output. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

Constraint bổ sung: partial `UNIQUE (application_id, environment, resource_requirement_id, deployment_target) WHERE status NOT IN ('DESTROYED', 'UNLINKED')` — mỗi chủ sở hữu có tối đa một Resource Instance đang dùng. Tìm để dùng lại luôn theo đủ bộ khóa này.

Không có table `resource_output`: `Resource Output` là runtime view `TRANSIENT`; output được collector nạp in-memory sau khi instance ready.

## Workload Instance Repository

### `workload_instance`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `workload_instance_id` | UUID | PK, NOT NULL | Identity của Workload Instance. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application sở hữu workload. |
| `workload_id` | UUID | FK → `application_component.component_id` (loại `WORKLOAD`), NOT NULL | ID cố định của workload. |
| `environment` | ENUM (`STAGING`, `PRODUCTION`) | NOT NULL | Environment nơi workload chạy. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Target nơi workload chạy. |
| `current_workload_deployment_id` | UUID | FK → `workload_deployment.workload_deployment_id`, NOT NULL | Workload deployment (image, phiên bản) đang chạy. |
| `status` | ENUM (`DEPLOYING`, `HEALTHY`, `FAILED`, `REMOVED`) | NOT NULL | Trạng thái hiện hành; `REMOVED` là trạng thái kết thúc, dòng được giữ lại. |
| `output_fingerprint` | CHAR(64) | NULL | SHA-256 hex của Workload Output lần gần nhất; không lưu giá trị output. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

Constraint bổ sung: `UNIQUE (workload_id, environment, deployment_target)`.

Không có table `workload_output`: `Workload Output` là runtime view `TRANSIENT` do Workload Output Collector đọc trong execution.

## Deployment Repository

### `deployment`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_id` | UUID | PK, NOT NULL | Identity của một lần deployment. |
| `application_id` | UUID | FK → `application_definition.application_id`, NOT NULL | Application được deploy. |
| `application_definition_version_id` | UUID | FK → `application_definition_version.application_definition_version_id` (`ON DELETE RESTRICT`), NOT NULL | Phiên bản Application Definition được deploy. |
| `environment_configuration_id` | UUID | FK → `environment_configuration.environment_configuration_id`, NOT NULL | Environment Configuration được dùng làm source references. |
| `environment` | ENUM (`STAGING`, `PRODUCTION`) | NOT NULL | Environment snapshot của deployment. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Kubernetes/deployment target đã chọn. |
| `plan_fingerprint` | CHAR(64) | NOT NULL | SHA-256 fingerprint dạng hex của Infrastructure Plan đã canonicalize để phát hiện plan thay đổi khi confirm; không chứa chính plan. |
| `plan_fingerprint_algo` | VARCHAR(32) | NOT NULL | Phiên bản thuật toán canonicalization + hashing của fingerprint, ví dụ `sha256-v1`, dùng lại khi rebuild plan. |
| `status` | ENUM (`AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED`, `FAILED`) | NOT NULL | Lifecycle status của Deployment, do IDP tự quyết; không nhận giá trị từ CD system. |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo deployment. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

### `workload_deployment`

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `workload_deployment_id` | UUID | PK, NOT NULL | Identity của workload snapshot trong deployment. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL | Deployment sở hữu snapshot. |
| `workload_id` | UUID | FK → `application_component.component_id` (loại `WORKLOAD`), NOT NULL | ID cố định của logical workload được deploy. |
| `image_repository` | VARCHAR(1024) | NOT NULL | Image repository thực tế. |
| `image_version` | VARCHAR(255) | NOT NULL | Image tag/version thực tế do Developer hoặc CI cung cấp; với workload được tự động deploy lại là image đang chạy. |
| `inclusion_reason` | ENUM (`SELECTED`, `CASCADED`) | NOT NULL | `SELECTED`: Developer chọn; `CASCADED`: được thêm tự động khi output của thành phần nó depends on thay đổi. |
| `wave_number` | INT | NOT NULL, CHECK (`wave_number` >= 0) | Tầng triển khai của workload trong deployment. |

Constraint bổ sung: `UNIQUE (deployment_id, workload_id)`; một Deployment phải có ít nhất một Workload Deployment (aggregate invariant, enforce ở transaction/service hoặc deferred database constraint). Workload được thêm do lan truyền output (`CASCADED`) được insert trong lúc Deployment Worker thực thi.

### `deployment_execution_job`

Job triển khai được tạo trong **cùng một transaction** với atomic compare-and-swap `deployment.status` từ `AWAITING_CONFIRMATION` sang `CONFIRMED`; không bao giờ có deployment `CONFIRMED` mà không có job, hay job mà deployment chưa được xác nhận. Deployment Worker lấy job để chạy nền.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `job_id` | UUID | PK, NOT NULL | Identity của job. |
| `deployment_id` | UUID | FK → `deployment.deployment_id`, NOT NULL, UNIQUE | Deployment cần thực thi; mỗi deployment có tối đa một job. |
| `selected_overrides` | JSONB | NOT NULL | Giá trị override Developer đã chọn khi xác nhận (không phải plan payload). |
| `status` | ENUM | NOT NULL | Trạng thái job; tập giá trị dự kiến xem danh mục ENUM (chốt ở D6). |
| `created_at` | TIMESTAMP | NOT NULL | Thời điểm tạo job. |
| `updated_at` | TIMESTAMP | NOT NULL | Thời điểm cập nhật gần nhất. |

Cơ chế phục hồi khi worker chết giữa chừng (lease, heartbeat, thử lại) chưa được mô hình hóa; xem D6.

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
| `environment` | ENUM (`STAGING`, `PRODUCTION`) | NOT NULL | Environment được ghi nhận cho history. |
| `deployment_target` | VARCHAR(255) | NOT NULL | Target thực tế được ghi nhận. |
| `delivery_reference` | VARCHAR(2048) | NULL | Reference tới desired state/CD delivery nếu có. |
| `removed_components` | JSONB | NOT NULL | Danh sách thành phần (ID cố định) đã được gỡ, hủy hoặc gỡ liên kết trong deployment; mảng rỗng nếu không có. |
| `status` | ENUM | NOT NULL | Final/current status được lưu bền vững. Có thể bỏ cột và dùng `deployment.status` — chốt ở D5; không được nhận giá trị từ CD system. |
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
| `wave_number` | INT | NOT NULL, CHECK (`wave_number` >= 0) | Tầng triển khai mà step thuộc về. |
| `step_name` | VARCHAR(255) | NOT NULL | Tên step; dự kiến đổi sang ENUM khi chốt D4. |
| `status` | ENUM | NOT NULL | Trạng thái step; tập giá trị dự kiến xem danh mục ENUM (chốt ở D4). |
| `related_component_reference` | VARCHAR(2048) | NULL | Workload/resource reference (ID cố định) mà step thuộc về. |
| `error_summary` | TEXT | NULL | Chi tiết lỗi của step nếu có. |
| `started_at` | TIMESTAMP | NULL | Thời điểm bắt đầu. |
| `completed_at` | TIMESTAMP | NULL | Thời điểm hoàn tất. |

`deployment_id` phải trùng deployment của `deployment_record_id`; constraint này được enforce bằng composite FK. Một record đã được tạo phải có ít nhất một step theo aggregate invariant.

### `deployment_record_resource_instance`

Đây là relational association table cho quan hệ persistent many-to-many `Deployment Record` ↔ `Resource Instance`; nó không phải một domain object mới.

| Column | Type | Constraints | Mô tả |
|---|---|---|---|
| `deployment_record_resource_instance_id` | UUID | PK, NOT NULL | Identity kỹ thuật của association row. |
| `deployment_record_id` | UUID | FK → `deployment_record.deployment_record_id`, NOT NULL | Deployment Record tham chiếu infrastructure. |
| `resource_instance_id` | UUID | FK → `resource_instance.resource_instance_id`, NOT NULL | Durable Resource Instance được create/update/reuse. |

Constraint bổ sung: `UNIQUE (deployment_record_id, resource_instance_id)`.

## Enforcement của hai persistence constraints chính

### Reference, không phải resolved runtime value

- `configuration_value.value_source` quyết định payload hợp lệ bằng `CHECK`; output-based rows chỉ lưu logical FK + output name.
- Resource binding lưu `resource_requirement_id` + `resource_output_name`; workload binding lưu `workload_id` + `workload_output_name`. Tên component được lấy từ table target, tránh denormalized name không có referential integrity.
- Schema không có `resolved_value`, resolved host/port/endpoint/credential trong `configuration_value`, `secret`, `deployment` hoặc `deployment_record`.
- Không tạo table cho `resource_output`, `resolved_configuration` hay `resolved_specification`; các giá trị này chỉ tồn tại trong deployment execution.
- Infrastructure Plan cũng giữ nguyên là `TRANSIENT` và được rebuild khi confirm; `deployment` chỉ persist `plan_fingerprint` cùng `plan_fingerprint_algo`, không persist plan hoặc resolved plan payload.

### Secret reference, không phải plaintext

- `secret` hoàn toàn không có column plaintext/value. Direct secret đi qua Secret Store và DB chỉ nhận `secret_ref` opaque.
- `CHECK` trên `secret.value_source` bảo đảm `SECRET_REF` và `RESOURCE_OUTPUT` loại trừ lẫn nhau.
- Với sensitive resource output, DB chỉ lưu `resource_requirement_id` + `resource_output_name`; resolved secret value không được persist.
- `configuration_value.direct_value` chỉ thuộc Environment Variable path vì table liên kết bắt buộc tới `environment_variable`, không thể liên kết tới `secret`.

### Phiên bản bất biến và ID cố định

- Mỗi lần Save ở UC-01 insert một dòng `application_definition_version` cùng toàn bộ dòng `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` của phiên bản đó; không `UPDATE`/`DELETE` dòng của phiên bản đã lưu. FK tới phiên bản dùng `ON DELETE RESTRICT`.
- ID logic của thành phần nằm ở `application_component` và giữ nguyên qua các phiên bản. Bảng nằm ngoài phiên bản (`environment_variable`, `configuration_value`, `secret`, `resource_instance`, `workload_instance`, `workload_deployment`) tham chiếu ID logic, nên lịch sử và cấu hình không bị hỏng khi thành phần đổi tên hoặc không còn trong phiên bản mới.
- `deployment.application_definition_version_id` ghi đúng phiên bản đã deploy. Thành phần không còn trong phiên bản chỉ bị gỡ ở hạ tầng/cluster; dòng `resource_instance`/`workload_instance` được giữ với status kết thúc.
- Chỉ lưu dấu vân tay output (`output_fingerprint`) trên `resource_instance` và `workload_instance`; không có cột lưu giá trị output.

## Danh mục ENUM

Bảng này là nguồn chuẩn duy nhất cho literal ENUM; domain model, operation contracts và state machine dùng đúng các giá trị dưới đây. Quy ước tên `UPPER_SNAKE_CASE`.

### Đã chốt

| Cột | Giá trị |
|---|---|
| `application_component.component_type` | `WORKLOAD`, `RESOURCE_REQUIREMENT`, `ENVIRONMENT_VARIABLE_DEFINITION`, `SECRET_DEFINITION` |
| `dependency.target_type` | `WORKLOAD`, `RESOURCE` |
| `configuration_value.value_source` | `DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT` |
| `secret.value_source` | `SECRET_REF`, `RESOURCE_OUTPUT` |
| `environment_configuration.environment`, `resource_instance.environment`, `workload_instance.environment`, `deployment.environment`, `deployment_record.environment` | `STAGING`, `PRODUCTION` |
| `resource_definition.management_mode` | `MANAGED`, `EXISTING` |
| `resource_instance.status` | `PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `DESTROYED`, `UNLINKED` |
| `workload_instance.status` | `DEPLOYING`, `HEALTHY`, `FAILED`, `REMOVED` |
| `deployment.status` | `AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED`, `FAILED` |
| `workload_deployment.inclusion_reason` | `SELECTED`, `CASCADED` |

### Dự kiến, chưa chốt

| Cột / khái niệm | Giá trị dự kiến | Chốt khi giải quyết |
|---|---|---|
| Loại thành phần trong plan | `RESOURCE`, `WORKLOAD` | D3 |
| Action cho resource trong plan | `CREATE`, `UPDATE`, `REUSE`, `DESTROY`, `UNLINK` | D3 |
| Action cho workload trong plan | `DEPLOY`, `REMOVE` | D3 |
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` | D4 |
| `deployment_step.step_name` (đổi từ VARCHAR sang ENUM) | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, `CD_SYNCED`, `APPLICATION_READY`, `REMOVED`, `DESTROYED`, `UNLINKED` | D4 |
| `deployment_record.status` | Bỏ cột, dùng `deployment.status` | D5 |
| `delivery_status` (trường riêng, nếu lưu) | `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED` | D5 |
| `deployment_execution_job.status` | `QUEUED`, `RUNNING`, `COMPLETED`, `FAILED` | D6 |

## Mapping domain → table

### PERSISTENT objects

| Domain object | Table / persistence mapping |
|---|---|
| Application Definition | `application_definition` |
| Application Definition Version | `application_definition_version` |
| Workload | `workload` (theo phiên bản) + ID cố định ở `application_component` |
| Resource Requirement | `resource_requirement` (theo phiên bản) + ID cố định ở `application_component` |
| Environment Variable Definition | `environment_variable_definition` (theo phiên bản) + ID cố định ở `application_component` |
| Secret Definition | `secret_definition` (theo phiên bản) + ID cố định ở `application_component` |
| Dependency | `dependency` (theo phiên bản) |
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
| Deployment Execution Job | `deployment_execution_job` |
| Deployment Context | `deployment_context` |
| Resource Instance | `resource_instance` |
| Workload Instance | `workload_instance` |
| Deployment Record | `deployment_record` |
| Deployment Step | `deployment_step` |

`deployment_record_resource_instance` chỉ hiện thực quan hệ many-to-many đã có trong domain model; không giới thiệu domain entity mới. `application_component` là identity table hiện thực ID cố định qua phiên bản của Workload, Resource Requirement, Environment Variable Definition và Secret Definition; cũng không phải domain entity mới.

### TRANSIENT objects intentionally excluded

| Domain object | Lý do không tạo table |
|---|---|
| Deployment Graph | Dựng lại cho từng execution từ persistent sources. |
| Resource Resolution | Quyết định trung gian; durable outcome là Resource Instance/reference. |
| Infrastructure Plan | Plan được rebuild khi confirm; chỉ SHA-256 fingerprint và algorithm version của canonical plan được persist trên `deployment`. |
| Resource Output | Runtime output được collector nạp in-memory sau khi resource ready; chỉ `resource_instance.output_fingerprint` được lưu. |
| Workload Output | Runtime output được Workload Output Collector đọc từ workload đã healthy hoặc đang chạy; chỉ `workload_instance.output_fingerprint` được lưu. |
| Resolved Configuration | In-memory snapshot của values đã resolve. |
| Resolved Specification | Execution artifact dùng làm input cho `score-k8s`, không phải versioned source specification. |
