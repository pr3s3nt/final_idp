# Step 4: Operation Contracts

Tài liệu này đặc tả các system operation quan trọng của UC-01 đến UC-03 và UC-05 theo kiểu Larman. Tên domain object dùng đúng Step 2; tên table/column `snake_case` dùng đúng Step 3. Các nhãn trạng thái như `AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED` và `FAILED` dùng đúng literal trong mục **Danh mục ENUM** của `03_database_erd/schema.md`; literal thuộc các mục hoãn (D3–D6) chỉ là giá trị dự kiến.

Các execution-scoped object `Deployment Graph`, `Resource Resolution`, Infrastructure Plan, `Resource Output`, `Workload Output`, `Resolved Configuration` và `Resolved Specification` là `TRANSIENT`; postcondition có thể tạo chúng trong execution hiện tại nhưng không tạo table/row tương ứng. Mọi postcondition bên dưới mô tả state sau khi operation hoàn tất, không mô tả trình tự gọi component. Từ contract 6 trở đi, operation do **Deployment Worker** chạy nền gọi, sau khi job triển khai đã được tạo ở contract 5. Cấu trúc chi tiết của plan (D3), việc ghi `deployment_step` (D4), việc tách trạng thái CD (D5) và phục hồi worker (D6) chưa được đặc tả ở đây.

## 1. `saveApplicationDefinition()`

- **Operation**: `saveApplicationDefinition(applicationDefinition)`
- **Cross References**: UC-01 – Create / Configure Application, luồng chính và A1 – Dữ liệu không hợp lệ.
- **Preconditions**:
  - Developer đã đăng nhập và có quyền tạo hoặc chỉnh sửa Application Definition được truyền vào.
  - Nếu là update, một instance `Application Definition` với `applicationId` tương ứng đã tồn tại trong `application_definition` và có ít nhất một `Application Definition Version`; nếu là create, `applicationId` chưa định danh một row khác và `name` chưa được dùng bởi Application Definition khác.
  - `applicationDefinition` có ít nhất một `Workload`. Trong nội dung submit, `Workload.name` và `Resource Requirement.name` không trùng; mỗi `Workload.imageRepository` hợp lệ; mỗi `Workload.port`, nếu có, nằm trong khoảng `1..65535`.
  - Thành phần đã có ở phiên bản trước mang đúng ID cố định của nó (kể cả khi đổi tên); thành phần mới chưa có ID hoặc mang ID chưa được dùng trong application.
  - Mỗi `Environment Variable Definition` và `Secret Definition` thuộc đúng một Workload trong nội dung submit; tên definition là duy nhất trong Workload tương ứng.
  - Mỗi `Dependency` có một source là Workload trong nội dung submit và đúng một target là `Workload` hoặc `Resource Requirement` trong nội dung submit; dependency logic không bị trùng và toàn bộ quan hệ `depends on` không tạo thành vòng.
  - Definition không chứa image tag/version, Environment Configuration hoặc deployment target; các dữ liệu này không thuộc UC-01.
- **Postconditions**:
  - Khi create, một instance `Application Definition` được tạo; một row `application_definition` tồn tại với `application_id`, `name`, `description`, `created_at` và `updated_at` tương ứng. Khi update, `name`, `description`, `updatedAt`/`updated_at` được sửa, còn `applicationId`/`application_id` không đổi.
  - Một instance `Application Definition Version` mới được tạo; một row `application_definition_version` được insert với `application_definition_version_id`, `application_id`, `version_number` lớn hơn mọi phiên bản trước của application và `created_at`.
  - Với mỗi thành phần mới (Workload, Resource Requirement, Environment Variable Definition, Secret Definition), một row `application_component` được insert với `component_id`, `application_id` và `component_type` tương ứng; thành phần đã có giữ nguyên `component_id`.
  - Các row `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition` và `dependency` của phiên bản mới được insert; mỗi row mang `application_definition_version_id` của phiên bản mới và ID cố định lấy từ `application_component`; `image_repository` được lưu nhưng không có image version.
  - Association giữa phiên bản và từng thành phần được hình thành qua `application_definition_version_id`; association Workload ↔ definition và Dependency ↔ source/target được hình thành qua composite FK trong cùng phiên bản.
  - Không row nào của các phiên bản trước bị `UPDATE` hay `DELETE`. Thành phần bị bỏ khỏi nội dung submit chỉ đơn giản không có row trong phiên bản mới; lịch sử và cấu hình vẫn tham chiếu được qua ID cố định.
  - Không instance `Deployment`, `Workload Deployment`, `Environment Configuration`, `Resource Instance` hoặc `Workload Instance` nào được tạo, xóa hay sửa; deployment hiện tại và các environment không tự động thay đổi.
  - Việc tạo/cập nhật `Application Specification` không thuộc state change trực tiếp của contract này; đó là postcondition của `generateApplicationSpecification()` kế tiếp trong luồng UC-01.
- **Exceptions / Guarantees**:
  - A1: nếu validation thất bại do tên trùng, image repository/port không hợp lệ, dependency tham chiếu sai hoặc tạo thành vòng, không row `application_definition_version`, `application_component` hay row thành phần nào được insert, và `application_definition` giữ nguyên.
  - Việc tạo phiên bản là atomic: không tồn tại phiên bản chỉ lưu một phần Workload, definition hoặc Dependency.

## 2. `generateApplicationSpecification()`

- **Operation**: `generateApplicationSpecification(applicationDefinitionVersion)`
- **Cross References**: UC-01 – Create / Configure Application, bước sinh/cập nhật application specification sau khi lưu phiên bản Application Definition mới.
- **Preconditions**:
  - `applicationDefinitionVersion` là một `Application Definition Version` hợp lệ vừa được persist trong `application_definition_version`, cùng toàn bộ Workload, Resource Requirement, configuration requirement và Dependency của phiên bản đó.
  - Phiên bản có ít nhất một Workload và thỏa các invariant của UC-01; không chứa image tag/version, Environment Configuration hoặc deployment target.
  - Chưa có row `application_specification` nào cho `application_definition_version_id` của input.
- **Postconditions**:
  - Một instance `Application Specification` được tạo cho phiên bản; một row `application_specification` được tạo với `application_id` và `application_definition_version_id` tương ứng.
  - Không tạo specification thứ hai cho cùng phiên bản, phù hợp constraint `UNIQUE (application_definition_version_id)`; specification của các phiên bản trước không bị sửa.
  - Thuộc tính `format` được gán format artifact được sinh, ví dụ `score-yaml`; `content` biểu diễn Workload, Resource Requirement, configuration requirement và Dependency hiện hành; `version` được gán version artifact mới; `updatedAt`/`updated_at` được cập nhật.
  - `content` không chứa deployment-time image version, resolved Environment Configuration, plaintext Secret hoặc Kubernetes target-specific state.
  - Không thuộc tính hoặc association nào của `Deployment`, `Workload Deployment`, `Environment Configuration` hay `Resource Instance` bị thay đổi.
- **Exceptions / Guarantees**:
  - A1: Application Definition không hợp lệ không tạo ra hoặc thay thế `Application Specification`.
  - Nếu generation hoặc persist artifact thất bại, row `application_specification` hiện hành vẫn giữ `content`, `version` và `updated_at` trước operation; không tồn tại artifact được cập nhật một phần.

## 3. `saveEnvironmentConfiguration()`

- **Operation**: `saveEnvironmentConfiguration(configuration)`
- **Cross References**: UC-02 – Configure Application Environment, luồng chính và A1 – Configuration không hợp lệ.
- **Preconditions**:
  - Một `Application Definition` với `configuration.applicationId` đã tồn tại trong `application_definition`; phiên bản mới nhất của nó có ít nhất một `Environment Variable Definition` hoặc `Secret Definition` cần cấu hình.
  - `configuration.environment` là `STAGING` hoặc `PRODUCTION`; nếu update, một `Environment Configuration` cho cặp `(applicationId, environment)` đã tồn tại, phù hợp `UNIQUE (application_id, environment)`.
  - Toàn bộ kiểm tra dưới đây dùng **phiên bản Application Definition mới nhất**; mọi tham chiếu dùng ID cố định trong `application_component`.
  - Mỗi `Environment Variable` tham chiếu một `Environment Variable Definition` và Workload sở hữu definition đó trong phiên bản mới nhất; mỗi definition bắt buộc có đúng một `Configuration Value`.
  - `Direct Configuration Value` chỉ được dùng cho Environment Variable thông thường. Mỗi `Resource Output Reference` tham chiếu một `Resource Requirement` trong phiên bản mới nhất và một output có trong `Resource Definition.exposedOutputs`; mỗi `Workload Output Reference` tham chiếu một Workload trong phiên bản mới nhất và một output có trong `Workload.exposedOutputs`.
  - Resource hoặc workload được tham chiếu (kể cả sensitive Resource Output của Secret) là target của một `Dependency` có source là workload sở hữu biến/secret đó.
  - Mỗi `Secret` tham chiếu đúng `Secret Definition` và Workload sở hữu definition đó, đồng thời có đúng một source: opaque `secretReference` do Secret Store trả về hoặc sensitive `Resource Output Reference` có output nằm trong `Resource Definition.sensitiveOutputs`.
  - Tất cả configuration bắt buộc đã có source hợp lệ; plaintext Secret, nếu Developer vừa nhập, đã được Secret Store chuyển thành `secretReference` trước khi persist aggregate.
- **Postconditions**:
  - Khi create, một instance `Environment Configuration` và một row `environment_configuration` được tạo với `environment_configuration_id`, `application_id`, `environment`, `created_at`, `updated_at`; association tới đúng `Application Definition` được hình thành qua `application_id`.
  - Khi update, instance hiện hành cho cùng `(applicationId, environment)` được sửa và `updatedAt`/`updated_at` được cập nhật; không tạo configuration thứ hai cho cùng cặp khóa logic.
  - Với mỗi variable được cấu hình, một instance `Environment Variable` tồn tại và association tới `Environment Configuration`, `Workload`, `Environment Variable Definition` được hình thành; row `environment_variable` chứa đúng `environment_configuration_id`, `workload_id`, `variable_definition_id`, `variable_name`.
  - Mỗi `Environment Variable` có đúng một instance `Configuration Value`; row `configuration_value` có đúng `value_source` và đúng một payload hợp lệ: `direct_value`, hoặc cặp `resource_requirement_id` + `resource_output_name`, hoặc cặp `workload_id` + `workload_output_name`. Các payload không thuộc source đã chọn bằng `NULL`.
  - Với mỗi secret được cấu hình, một instance `Secret` tồn tại và association tới `Environment Configuration`, `Workload`, `Secret Definition` được hình thành; row `secret` có `value_source = SECRET_REF` cùng `secret_ref`, hoặc `value_source = RESOURCE_OUTPUT` cùng `resource_requirement_id` + `resource_output_name`, và các payload còn lại bằng `NULL`.
  - Tập row `environment_variable`, `configuration_value` và `secret` bằng đúng tập binding hiện hành của configuration đã submit: binding mới được tạo, binding còn tồn tại được cập nhật, binding bị loại bỏ được xóa và các association tương ứng bị phá vỡ.
  - Chỉ logical reference được persist; không instance/row `Resource Output` hay `Resolved Configuration` được tạo và không resolved host, endpoint hoặc credential nào thay thế reference đã lưu.
  - Không `Deployment`, `Workload Deployment`, `Deployment Record` hoặc `Resource Instance` nào bị tạo, xóa hay sửa; thay đổi configuration không tự động deploy application đang chạy.
- **Exceptions / Guarantees**:
  - A1: nếu thiếu giá trị bắt buộc, resource/workload không tồn tại trong phiên bản mới nhất, output không được expose, hoặc output thuộc thành phần mà workload sở hữu biến không depends on, toàn bộ state trong `environment_configuration`, `environment_variable`, `configuration_value` và `secret` giữ nguyên như trước operation.
  - Plaintext Secret không xuất hiện trong `Environment Configuration Repository`, `Deployment Repository` hoặc log của contract; chỉ opaque `secret_ref` hoặc sensitive Resource Output Reference được persist.

## 4. `createDeployment()`

- **Operation**: `createDeployment(applicationId, version, catalogVersion, environment, target, images, context)`
- **Cross References**: UC-03 – Deploy Application, từ bước chọn phiên bản/image/context đến bước hiển thị các tầng triển khai và infrastructure plan; A1 – Deployment input hoặc dependency không hợp lệ.
- **Preconditions**:
  - `application_definition.application_id = applicationId` định danh một `Application Definition` hợp lệ; `version` định danh một `Application Definition Version` của application đó có ít nhất một Workload, và các Dependency của phiên bản không tạo thành vòng.
  - `environment` là `STAGING` hoặc `PRODUCTION`. Một row `environment_configuration` tồn tại cho đúng `(applicationId, environment)` và **khớp với phiên bản được chọn**: mọi `Environment Variable Definition.required = true`/`Secret Definition.required = true` của phiên bản đều có binding hợp lệ; không binding nào tham chiếu output của thành phần không có trong phiên bản hoặc không được depends on trong phiên bản.
  - `images` chứa đúng một image version hợp lệ cho mỗi Workload **được chọn**; repository của image lấy từ `workload.image_repository` của phiên bản.
  - `catalogVersion` định danh một row `catalog_version`.
  - Nếu chỉ một phần Workload của phiên bản được chọn, `version` và `catalogVersion` trùng phiên bản Application Definition và phiên bản catalog đang chạy trên `(environment, target)` (theo `workload_instance.current_workload_deployment_id` → `deployment.application_definition_version_id`, `deployment.catalog_version_id`). Đổi một trong hai phải chọn toàn bộ Workload.
  - Mọi Workload mà các Workload được chọn depends on nhưng không nằm trong phạm vi có `workload_instance.status = HEALTHY` trên cùng `(environment, target)`.
  - `target` được hỗ trợ: phiên bản catalog `catalogVersion` có Resource Definition loại `k8s-cluster` phù hợp với target và `context` — `MANAGED` khi target là cloud (IDP dựng cụm), `EXISTING` khi target là cụm Kubernetes nội bộ có sẵn. `context` chứa `cloudProvider`, `region` và `targetSpecificInput` bắt buộc đối với target đó (cụm nội bộ không cần `cloudProvider`, `region`).
  - Với mỗi `Resource Requirement` và `Platform Requirement` trong graph, phiên bản catalog `catalogVersion` có một `Resource Definition` phù hợp với `resourceType`, `supportedContexts` và `applicability_conditions` (nếu có), cùng `provisionerReference` hợp lệ, đồng thời cung cấp `default_parameters`, `allowed_overrides` và `requires`. Các `requires` không tạo thành vòng.
- **Postconditions**:
  - Một instance `Deployment` được tạo; một row `deployment` được tạo với `deployment_id`, `application_id`, `application_definition_version_id`, `catalog_version_id = catalogVersion`, `environment_configuration_id`, `environment`, `deployment_target`, `plan_fingerprint`, `plan_fingerprint_algo`, `created_at`, `updated_at`, và `status = AWAITING_CONFIRMATION`.
  - Association `Deployment` → `Application Definition Version`, `Deployment` → `Catalog Version` và `Deployment` → `Environment Configuration` được hình thành qua `application_definition_version_id`, `catalog_version_id` và `environment_configuration_id`; environment của configuration, snapshot `deployment.environment` và input `environment` bằng nhau.
  - Với mỗi Workload được chọn, đúng một instance `Workload Deployment` và một row `workload_deployment` được tạo với `deployment_id`, `workload_id` (ID cố định), `image_repository`, `image_version` từ `images`, `inclusion_reason = SELECTED` và `wave_number` theo tầng đã chia.
  - Đúng một instance `Deployment Context` và một row `deployment_context` được tạo cho Deployment; association 1:1 được hình thành qua `deployment_id`, và `cloud_provider`, `region`, `target_specific_input` phản ánh `context`.
  - Một instance `Deployment Graph` `TRANSIENT` được tạo cho execution hiện tại với `deploymentId`, `workloadIds`, `resourceRequirementIds`, `platformRequirementIds` (`k8s-cluster` và các loại mà Resource Definition được resolve `requires`), `dependencyIds`, `configurationReferenceIds`. Cạnh gồm Dependency của phiên bản, mỗi Workload → `k8s-cluster`, và mỗi resource → Platform Requirement mà definition của nó `requires`. Kết quả `planDeploymentWaves()`: `scopeComponentIds` (Workload được chọn, Resource Requirement mà chúng depends on trực tiếp, và Platform Requirement mà các resource đó `requires`, bắc cầu), `waves` theo thứ tự phụ thuộc; sau khi có infrastructure plan, `potentialRedeployComponentIds` (thành phần ngoài phạm vi hoặc ở tầng sau dựa trên một thành phần sẽ được create, update hoặc deploy).
  - Với mỗi Resource Requirement và Platform Requirement trong graph, một instance `Resource Resolution` `TRANSIENT` được tạo và liên kết tới đúng `Resource Definition` (`MANAGED` hoặc `EXISTING`) thuộc phiên bản catalog `catalogVersion`. Resource Instance hiện có được tìm theo đúng khóa chủ sở hữu `(application_id, environment, resource_requirement_id, deployment_target)`, trong đó `resource_requirement_id` là ID cố định của Resource Requirement hoặc Platform Requirement.
  - Execution state chứa infrastructure plan ở mức khái niệm (cấu trúc chi tiết để D3): theo từng tầng, mỗi resource (kể cả cụm Kubernetes và network) là create, update, reuse hoặc liên kết thứ có sẵn `EXISTING` (resource dùng chung, cụm nội bộ). Resource `MANAGED` đã có instance `READY` là update khi đầu vào hiện tại (definition trong phiên bản catalog, tham số đã resolve, dấu vân tay output của các Platform Requirement được `requires`) khác `resource_instance.applied_input_fingerprint`, ngược lại là reuse. Khi `version` khác phiên bản đang chạy, thêm Workload cần gỡ, resource `MANAGED` cần hủy và resource `EXISTING` cần gỡ liên kết; cùng danh sách override được phép.
  - Infrastructure plan được canonicalize bằng stable resource/key ordering và canonical JSON, normalize unit/number, đồng thời loại timestamps và auto-generated ID không mang ý nghĩa nghiệp vụ. Fingerprint domain bao phủ phiên bản catalog; action `CREATE`/`UPDATE`/`REUSE` của từng resource; identity của Resource Definition được tham chiếu và các field ảnh hưởng plan gồm `provisioner_reference`, `supported_contexts`, `default_parameters`, `allowed_overrides` và `requires`; resolved parameters của plan được suy ra một cách tất định từ `default_parameters` kết hợp deployment context, còn allowed-overrides definition (tập key được phép cùng tập giá trị, min-max hoặc enum) được suy ra một cách tất định từ `allowed_overrides`; Resource Instance identity được tham chiếu; và deployment target. Resource Definition của một phiên bản catalog không bao giờ thay đổi, nên thay đổi catalog chỉ đi vào plan khi Developer chọn phiên bản catalog khác.
  - `Deployment.planFingerprint`/`deployment.plan_fingerprint` được gán SHA-256 hex của canonical plan; `Deployment.planFingerprintAlgo`/`deployment.plan_fingerprint_algo` được gán phiên bản thuật toán tương ứng, ví dụ `sha256-v1`. Cả hai được persist atomically cùng aggregate, và infrastructure plan + allowed overrides + fingerprint + algorithm version được trả về UI.
  - Chưa có `Resource Instance`, `Workload Instance` hay row `application_component` loại `PLATFORM_REQUIREMENT` nào bị tạo, sửa hoặc gỡ bởi operation này; chưa có `Deployment Execution Job`, `Deployment Record`, `Deployment Step`, desired deployment state hay CD delivery reference nào được tạo.
- **Exceptions / Guarantees**:
  - A1: nếu phiên bản Application Definition hoặc phiên bản catalog không tồn tại, image version, required configuration, cấu hình không khớp phiên bản, deploy một phần với phiên bản Application Definition hoặc phiên bản catalog khác bản đang chạy, dependency hoặc `requires` tạo vòng hoặc không resolve được, Workload phụ thuộc ngoài phạm vi chưa chạy healthy, output reference, Resource Definition (kể cả definition `k8s-cluster` cho target) hoặc context không hợp lệ, không aggregate `Deployment` hoàn chỉnh nào tồn tại; các row `deployment`, `workload_deployment`, `deployment_context` phát sinh trong attempt được rollback và infrastructure không thay đổi.
  - Không có row độc lập cho `Deployment Graph`, `Resource Resolution` hoặc infrastructure plan; các object này chỉ tồn tại trong execution state. Chỉ `plan_fingerprint` và `plan_fingerprint_algo`, không phải plan payload, được persist trên `deployment`.

## 5. `confirmDeployment()`

- **Operation**: `confirmDeployment(deploymentId, overrides)`
- **Cross References**: UC-03 – Deploy Application, bước Developer đặt permitted overrides và chọn Deploy; A1 – Deployment input hoặc dependency không hợp lệ. UC-05 – Remove Application from Environment dùng lại chính operation này ở bước Developer xác nhận gỡ, khi đó không có override nào được gửi.
- **Preconditions**:
  - Một `Deployment` với `deploymentId` tồn tại, có `status = AWAITING_CONFIRMATION`, có đúng một `Deployment Context`, và có `planFingerprint`/`planFingerprintAlgo` đã persist. Số `Workload Deployment` phụ thuộc loại deployment: `kind = DEPLOY` có **ít nhất một**; `kind = TEARDOWN` **không có cái nào**, vì UC-05 không triển khai workload nào.
  - Các input bền vững để rebuild plan còn tồn tại và nhất quán: Application Definition Version và Catalog Version được tham chiếu (cả hai bất biến), Environment Configuration, snapshot Workload Deployment/images (rỗng khi `kind = TEARDOWN`), Deployment Context và deployment target; current Resource Instance và Workload Instance state có thể được đọc lại.
  - Không giả định execution của request `createDeployment()` hoặc infrastructure plan in-memory còn tồn tại. `overrides` là candidate values từ confirm request và chỉ được validate sau khi rebuilt-plan fingerprint khớp fingerprint đã lưu.
- **Postconditions**:
  - Infrastructure plan được rebuild từ persisted inputs và current state bằng cùng chuỗi `buildDeploymentGraph()` (gọi `resolveResourceDefinitions()` trong phiên bản catalog của Deployment) → `planDeploymentWaves()` → `findResourceInstances()` (theo khóa chủ sở hữu) → `planInfrastructureChanges()` → `findPotentialRedeploys()`. Với cùng các input đó, resolved parameters được suy ra một cách tất định từ `resource_definition.default_parameters` kết hợp Deployment Context, còn allowed-overrides definition được suy ra một cách tất định từ `resource_definition.allowed_overrides`; fingerprint được tính lại bằng đúng canonicalization/hash version trong `planFingerprintAlgo`. Với `kind = TEARDOWN`, plan được dựng lại bằng chuỗi tương ứng của UC-05 — liệt kê thành phần đã khai báo của phiên bản (chỉ tên, không resolve), đọc Resource Instance và Workload Instance hiện có, rồi lập các tầng gỡ — và không có override nào để validate.
  - Khi rebuilt fingerprint khớp `planFingerprint`, `overrides` được validate theo policy trong `resource_definition.allowed_overrides`.
  - Sau khi fingerprint khớp và overrides hợp lệ, trong **một transaction DB**: `deployment.status` được đổi bằng atomic compare-and-swap có predicate `deployment_id = deploymentId AND status = AWAITING_CONFIRMATION` sang `CONFIRMED` (cập nhật `updated_at`), **và** một instance `Deployment Execution Job` cùng một row `deployment_execution_job` được tạo với `deployment_id`, `selected_overrides` chứa đúng các override hợp lệ, `status` ở giá trị khởi tạo (dự kiến `QUEUED`, D6), `created_at`, `updated_at`. Không bao giờ có deployment `CONFIRMED` mà không có job, hay job mà deployment chưa `CONFIRMED`.
  - Request trả lời ngay sau khi transaction commit; operation **không** gọi `reconcileInfrastructure()` hay bất kỳ bước thực thi nào. Việc thực thi do Deployment Worker đảm nhận khi lấy job (`claimNextExecutionJob()`), bắt đầu bằng việc đổi `deployment.status` từ `CONFIRMED` sang `DEPLOYING`.
  - Association của Deployment với `Application Definition Version`, `Catalog Version`, `Environment Configuration`, `Workload Deployment` và `Deployment Context` không đổi; phiên bản, phiên bản catalog, image version, environment và deployment target đã snapshot không đổi.
- **Exceptions / Guarantees**:
  - `PLAN_CHANGED`: nếu rebuilt fingerprint khác `deployment.plan_fingerprint`, không apply overrides và không reconcile infrastructure; Deployment giữ `AWAITING_CONFIRMATION`. Hệ thống atomically refresh fingerprint lưu trên Deployment cho rebuilt plan khi stored fingerprint vẫn là giá trị vừa so sánh, rồi trả error cùng rebuilt plan, allowed overrides và rebuilt fingerprint để Developer review và re-confirm; nếu refresh cạnh tranh thất bại, request vẫn không reconcile và lần confirm sau sẽ rebuild/compare lại.
  - A1: nếu plan không rebuild được hoặc override không được phép/không hợp lệ sau khi fingerprint khớp, Deployment giữ `AWAITING_CONFIRMATION`, CAS không chạy, không có `deployment_execution_job` nào được tạo và infrastructure không thay đổi.
  - `DEPLOYMENT_ALREADY_CONFIRMED`: nếu status ban đầu không còn là `AWAITING_CONFIRMATION`, hoặc atomic status CAS ảnh hưởng zero rows do confirmation đồng thời đã thắng, request bị từ chối; transaction của losing request rollback nên không tạo job thứ hai (`UNIQUE (deployment_id)`), không tạo Deployment/Workload Deployment mới và không kích hoạt bước thực thi nào.
  - Nếu transaction tạo job thất bại, CAS cũng rollback; Deployment giữ `AWAITING_CONFIRMATION`.
- **Scope boundaries**:
  - Fingerprint bao phủ **allowed-overrides definition** được suy ra từ `resource_definition.allowed_overrides` (tập parameter được phép cùng tập giá trị, min-max hoặc enum), không bao phủ các override value Developer chọn; các value này chỉ đến trong confirm request và được validate sau khi fingerprint khớp.
  - Fingerprint là application-level defense-in-depth guard: nó thu hẹp nhưng không loại bỏ TOCTOU. Contention trên shared Resource Instance giữa các deployment và drift trong lúc reconcile cần cơ chế bổ sung như Resource-Instance-level version/optimistic lock hoặc per-resource reconcile lock, cùng provisioner idempotency (ví dụ Terraform refresh + plan); các cơ chế bổ sung này nằm ngoài phạm vi fingerprint.

## 6. `reconcileInfrastructure()`

- **Operation**: `reconcileInfrastructure(finalPlan, planItems, requiredResourceOutputs)`
- **Cross References**: UC-03 – Deploy Application, bước Deployment Worker triển khai resource của một tầng (kể cả cụm Kubernetes và network), và bước xử lý thành phần không còn trong phiên bản; A2 – Provisioning, delivery hoặc workload thất bại. UC-05 – Remove Application from Environment dùng lại operation này cho các tầng gỡ, khi đó **mọi** item đều là hủy hoặc gỡ liên kết.
- **Preconditions**:
  - Deployment đang ở `status = DEPLOYING` và được Deployment Worker thực thi từ job đã lưu. `finalPlan` là plan dựng lại từ persisted inputs với `selected_overrides` của job; `planItems` là các resource item của **một tầng** (gồm cả item được thêm lại do lan truyền, contract 11), hoặc các item gỡ/hủy/gỡ liên kết khi phiên bản được deploy không còn resource đó.
  - Mỗi item đã resolve tới một `Resource Definition` thuộc phiên bản catalog `deployment.catalog_version_id`, có `resourceDefinitionId`, `provisionerReference` (với `MANAGED`) hoặc `existing_resource_reference` (với `EXISTING`), và `supportedContexts` phù hợp với `Deployment Context`/`deploymentTarget`.
  - Resource Instance mục tiêu được xác định theo khóa chủ sở hữu `(application_id, environment, resource_requirement_id, deployment_target)`: với item update/reuse/hủy/gỡ liên kết, instance đó tồn tại; với item create hoặc liên kết lần đầu, chưa có instance đang dùng cho khóa này. Với item là Platform Requirement, row `application_component` loại `PLATFORM_REQUIREMENT` tương ứng đã tồn tại (Deployment Worker đã gọi `ensurePlatformRequirements()`).
  - Mọi resource mà các item phụ thuộc đã sẵn sàng ở các tầng trước. Với mỗi item create/update `MANAGED`, `requiredResourceOutputs` chứa output của mọi Platform Requirement mà definition của item `requires`, lấy từ instance `READY`. Provisioner Adapter và Terraform/OpenTofu Runner tương ứng sẵn sàng với item `MANAGED`.
- **Postconditions**:
  - Với mỗi item create (`MANAGED`) thành công, provisioner được gọi với tham số đã resolve và `requiredResourceOutputs`; một instance `Resource Instance` và một row `resource_instance` được tạo với `application_id`, `environment`, `resource_requirement_id`, `resource_definition_id` (definition của phiên bản catalog), `deployment_target`, `infrastructure_reference`, `provider_state_reference`, `applied_input_fingerprint` (hash của definition, tham số đã resolve và dấu vân tay output của các Platform Requirement được `requires`), `created_at`, `updated_at` và `status = READY`.
  - Với mỗi item update (`MANAGED`) thành công, provisioner được gọi với input hiện tại; `resource_definition_id`, `infrastructure_reference`, `provider_state_reference`, `applied_input_fingerprint`, `status = READY`, `updated_at` được cập nhật theo lần apply này; identity và `created_at` không đổi.
  - Với mỗi item reuse, instance hiện hữu được giữ và provisioner không được gọi; identity, `infrastructure_reference` và `applied_input_fingerprint` không đổi, `status` được xác nhận là `READY`. Nếu Deployment dùng phiên bản catalog khác nhưng đầu vào không đổi, `resource_definition_id` được trỏ sang definition cùng `name` của phiên bản catalog đang dùng.
  - Với mỗi item liên kết thứ có sẵn `EXISTING` (resource dùng chung hoặc cụm Kubernetes nội bộ), không provisioner nào được gọi và hạ tầng thật không thay đổi; instance của chủ sở hữu được tạo hoặc giữ với `infrastructure_reference = existing_resource_reference`, `applied_input_fingerprint = NULL` và `status = READY`.
  - Với mỗi item hủy (`MANAGED`, resource không còn trong phiên bản — hoặc mọi resource của chủ sở hữu khi `deployment.kind = TEARDOWN`), provisioner thực hiện destroy; instance được giữ lại với `status = DESTROYED`.
  - Với mỗi item gỡ liên kết (`EXISTING`, resource không còn trong phiên bản — hoặc mọi resource `EXISTING` của chủ sở hữu khi `deployment.kind = TEARDOWN`), không provisioner nào được gọi, hạ tầng thật không thay đổi; instance được giữ lại với `status = UNLINKED`.
  - Execution result chứa `infrastructureReferences` trỏ tới đúng các Resource Instance đã xử lý. `deployment.status` giữ `DEPLOYING`; operation không đặt trạng thái trung gian nào cho Deployment.
  - `output_fingerprint` không bị đổi bởi operation này; nó được cập nhật sau khi output được thu thập và so sánh (contract 11). Không row `resource_output` nào được tạo.
- **Exceptions / Guarantees**:
  - A2: Resource Instance thất bại có `status = FAILED`; nếu đã có durable provider/resource state thì reference khả dụng và `updated_at` phản ánh state thực tế thay vì báo thành công giả.
  - Khi bất kỳ item thất bại, Deployment Worker không triển khai các tầng sau và `deployment.status` chuyển sang `FAILED`; tầng, thành phần liên quan và `errorSummary` được giữ để `saveDeploymentRecord()` persist. Các Resource Instance đã xử lý thành công trước lỗi không bị tuyên bố rollback nếu provider không hỗ trợ rollback.
  - Không item nào trỏ tới resource `EXISTING` được phép gọi provisioner apply hoặc destroy.

## 7. `resolveEnvironmentConfiguration()`

- **Operation**: `resolveEnvironmentConfiguration(configuration, waveWorkloads, resourceOutputs, workloadOutputs)`
- **Cross References**: UC-03 – Deploy Application, bước Deployment Worker resolve Environment Configuration cho các workload của một tầng; A1 – reference không hợp lệ; A2 – lỗi trong deployment execution.
- **Preconditions**:
  - Deployment đang ở `status = DEPLOYING`. `configuration` là Environment Configuration đã persist và được Deployment hiện tại tham chiếu; `configuration.applicationId` và `configuration.environment` khớp `deployment.application_id` và `deployment.environment`.
  - `waveWorkloads` là các Workload (theo ID cố định) của một tầng, thuộc phiên bản `deployment.application_definition_version_id`.
  - Với mỗi workload trong tầng, mỗi direct Environment Variable có một `Direct Configuration Value`; mỗi output-based variable có đúng một `Resource Output Reference` hoặc `Workload Output Reference`; mỗi Secret có đúng một `secretReference` hoặc sensitive `Resource Output Reference`.
  - Mỗi Resource Output Reference trỏ tới Resource Requirement mà workload depends on trong phiên bản được deploy, và resource đó đã `READY` (ở tầng trước hoặc là instance đang dùng); `resourceOutputs` chứa đúng output cần thiết và cờ `sensitive` phù hợp.
  - Mỗi Workload Output Reference trỏ tới Workload mà workload depends on trong phiên bản được deploy; `workloadOutputs` chứa output của workload đó, được thu thập ở tầng trước hoặc từ bản đang chạy ngoài phạm vi (contract 10).
  - Mọi `secretReference` còn hợp lệ và có thể được Secret Store/Secret Materializer sử dụng mà không persist plaintext.
- **Postconditions**:
  - Một instance `Resolved Configuration` `TRANSIENT` được tạo cho các workload của tầng với `deploymentId` của Deployment hiện tại; association execution-scoped `Resolved Configuration` → `Environment Configuration` được hình thành.
  - `resolvedEnvironmentVariables` chứa một entry cho mỗi `Environment Variable` của các workload trong tầng: direct value được lấy từ `Direct Configuration Value`, Resource Output được lấy theo `resourceRequirementId` + `outputName`, và Workload Output được lấy theo `workloadId` + `outputName`.
  - `resolvedSecrets` chứa secret value/reference phù hợp cho mỗi `Secret` của các workload trong tầng và duy trì phân loại sensitive; plaintext Secret không trở thành thuộc tính của bất kỳ persistent domain object nào.
  - `resolvedDependencies` chứa mapping đã resolve cho các Dependency/configuration reference của các workload trong tầng; association execution-scoped tới các `Resource Output` và `Workload Output` đã consume được hình thành.
  - `deployment.status` giữ `DEPLOYING`; operation không đặt trạng thái trung gian nào cho Deployment.
  - Không row trong `environment_configuration`, `environment_variable`, `configuration_value` hoặc `secret` bị thay đổi; logical references không bị thay thế bằng resolved values. Không table/row `resource_output` hoặc `resolved_configuration` được tạo.
- **Exceptions / Guarantees**:
  - A1: reference tới resource/workload/output không hợp lệ hoặc required binding bị thiếu thì không có instance `Resolved Configuration` hoàn chỉnh và persistent Environment Configuration giữ nguyên.
  - A2: nếu output/secret reference hợp lệ về cấu trúc nhưng không resolve được tại runtime, Deployment Worker không triển khai các tầng sau và `deployment.status` chuyển sang `FAILED`; failed step và error summary được chuyển cho `saveDeploymentRecord()`. Resolved secret không được ghi vào Deployment Record hoặc error detail.

## 8. `publishDesiredDeploymentState()`

- **Operation**: `publishDesiredDeploymentState(desiredState)`
- **Cross References**: UC-03 – Deploy Application, bước Deployment Worker publish desired deployment state của một tầng tới CD abstraction, và lần publish cuối gỡ các workload không còn trong phiên bản; A2 – Provisioning, delivery hoặc workload thất bại; UC-05 – Remove Application from Environment, bước publish desired state không còn workload nào của environment đó.
- **Preconditions**:
  - Deployment đang ở `status = DEPLOYING`. Các workload của tầng đã có `Resolved Configuration` và `Resolved Specification` hoàn chỉnh; mỗi `Workload Deployment.imageRepository` + `imageVersion` của tầng đã xuất hiện đúng trong resolved specification.
  - Base Kubernetes manifest đã được sinh bởi `score-k8s`, target-specific adaptation đã được áp dụng cho `deploymentTarget`, và Environment Variable/Secret đã được materialize đầy đủ vào `desiredState`.
  - `desiredState` không chứa unresolved Resource Output Reference hoặc Workload Output Reference; secret material tuân thủ cơ chế Kubernetes Secret hoặc secret reference mà không làm lộ plaintext ngoài delivery boundary.
  - `desiredState` gồm workload của tầng hiện tại và các tầng trước, và **vẫn giữ** workload đang chạy nhưng không còn trong phiên bản; chỉ lần publish cuối (sau khi mọi tầng đã healthy) mới bỏ các workload đó.
  - Khi `deployment.kind = TEARDOWN`, đây là một lần publish gỡ thuần: `desiredState` không upsert workload nào, nên các precondition về tầng, Resolved Configuration, Resolved Specification và image ở trên không áp dụng.
  - CD Integration đã resolve được một Concrete CD Provider cho target.
  - Application có một `Delivery Repository`, hoặc có thể tạo được: quy ước đặt tên do platform cấu hình và thông tin đăng nhập hệ thống lưu trữ Git nằm trong Secret Store và còn dùng được.
- **Postconditions**:
  - Application có đúng một instance `Delivery Repository` và một row tương ứng với `application_id`, `repository_url`, `branch` và secret reference của cặp khóa. Ở lần deploy đầu tiên của application, nơi chứa được tạo theo quy ước đặt tên, một cặp khóa riêng của application được sinh và gắn vào nơi chứa đó, rồi row được tạo; các lần sau dùng lại row đã có. Nơi chứa đã tồn tại đúng tên được dùng lại thay vì báo lỗi.
  - Desired deployment state chỉ được ghi vào `Delivery Repository` của application này, tách theo `deploymentTarget` và `environment`; không application nào khác ghi vào nơi chứa đó.
  - Trong CD System, desired deployment state cho application/environment/target được tạo hoặc cập nhật và association delivery tới Kubernetes deployment target được hình thành; CD System đọc nơi chứa bằng khóa đọc của chính application này.
  - Execution state nhận một `deliveryReference` định danh durable desired-state/CD delivery; acknowledgment của CD không được dùng làm trạng thái Deployment (D5).
  - Với mỗi workload của tầng, row `workload_instance` theo `(workload_id, environment, deployment_target)` được tạo hoặc cập nhật với `current_workload_deployment_id` trỏ tới Workload Deployment của deployment này và `status = DEPLOYING`. Với `kind = TEARDOWN` không workload nào được upsert nên không row nào chuyển sang `DEPLOYING`.
  - Ở lần publish cuối gỡ workload — hoặc ở tầng gỡ của một `TEARDOWN`, khi đó là **mọi** workload của chủ sở hữu — mỗi workload bị gỡ có `workload_instance.status = REMOVED` sau khi IDP xác minh nó đã biến mất khỏi cụm; dòng được giữ lại. Việc code chưa bảo đảm bước xác minh này trong mọi nhánh teardown được theo dõi ở D13.
  - `deployment.status` giữ `DEPLOYING`; không có trạng thái "đã gửi sang CD". Việc workload healthy được xác nhận riêng bởi `waitForWorkloadsHealthy()` trước contract 10.
  - Không `Deployment Record` hoặc association `deployment_record_resource_instance` nào được tạo bởi operation này; `deliveryReference` chỉ được persist bởi `saveDeploymentRecord()`.
  - Không Application Definition Version, Environment Configuration hoặc Resource Instance nào bị thay đổi bởi việc publish.
  - Khóa của application và thông tin đăng nhập hệ thống lưu trữ Git không được persist ngoài Secret Store: `delivery_repository` chỉ giữ secret reference, và chúng không xuất hiện trong Deployment Record, `deployment_step` hay log.
- **Exceptions / Guarantees**:
  - A2: nếu việc tạo `Delivery Repository`, sinh/gắn cặp khóa, manifest generation/materialization trước publish hoặc CD delivery thất bại, không có success status giả; Deployment Worker không triển khai các tầng sau, `deployment.status` chuyển sang `FAILED`, và delivery reference (nếu provider đã cấp), tầng, thành phần cùng error summary được giữ cho `saveDeploymentRecord()`.
  - Việc CD System accept desired state không đồng nghĩa Kubernetes workload đã ready; contract chỉ bảo đảm delivery reference phản ánh đúng acknowledgment nhận được.

## 9. `saveDeploymentRecord()`

- **Operation**: `saveDeploymentRecord(deployment, workloadImages, infrastructureReferences, deliveryReferences, removedComponents)`
- **Cross References**: UC-03 – Deploy Application, bước Deployment Worker lưu Deployment Record sau khi xử lý xong các tầng và các thành phần cần gỡ; A2 – Provisioning, delivery hoặc workload thất bại; UC-04 – View Deployment Result (nguồn dữ liệu được đọc về sau); UC-05 – Remove Application from Environment, bước lưu kết quả lần gỡ.
- **Preconditions**:
  - Deployment Worker đang thực thi đúng một `Deployment` với `status = DEPLOYING`; row `deployment` tồn tại với đúng một Deployment Context. `kind = DEPLOY` có ít nhất một Workload Deployment; `kind = TEARDOWN` không có Workload Deployment nào.
  - `workloadImages` bằng snapshot đã persist trong các row `workload_deployment` của Deployment, gồm cả workload `CASCADED` được thêm trong lúc thực thi. Với `kind = TEARDOWN`, `workloadImages` rỗng và toàn bộ kết quả nằm ở `removedComponents`.
  - Mỗi item trong `infrastructureReferences` định danh một row `resource_instance` đã xử lý cho execution này; không truyền resolved Resource Output.
  - `removedComponents` là danh sách ID cố định của workload đã gỡ và resource đã hủy/gỡ liên kết trong execution này (rỗng nếu không có).
  - Ở success path, mọi tầng đã healthy và các thành phần cần gỡ đã được xử lý. Ở A2 path, execution context chứa tầng, thành phần liên quan (nếu có) và `errorSummary`; `deliveryReferences` có thể rỗng.
- **Postconditions**:
  - Nếu Deployment chưa có record, một instance `Deployment Record` và một row `deployment_record` được tạo; association 1:0..1 `Deployment` → `Deployment Record` được hình thành qua `deployment_id`. Nếu đã có, chính record đó được cập nhật; `UNIQUE (deployment_id)` bảo đảm không có record thứ hai.
  - `environment`, `deployment_target`, `delivery_reference`, `removed_components`, `status`, `error_summary`, `created_at`, `updated_at` của `deployment_record` phản ánh kết quả thực tế của execution; `created_at` không đổi khi update.
  - Images được truy xuất qua `deployment_record.deployment_id` → `workload_deployment.deployment_id`, không tạo bản sao image.
  - Với mỗi Resource Instance được tham chiếu, đúng một row `deployment_record_resource_instance` tồn tại; association cũ không còn thuộc execution result bị phá vỡ khi record được cập nhật.
  - Ở success path, `deployment.status` được đổi bằng compare-and-swap từ `DEPLOYING` sang `SUCCEEDED` (cập nhật `updated_at`), và `error_summary = NULL`. `SUCCEEDED` nghĩa là mọi workload trong phạm vi (kể cả `CASCADED`) đã healthy khi `kind = DEPLOY`, hoặc mọi thành phần trong plan gỡ đã được gỡ, hủy hay gỡ liên kết khi `kind = TEARDOWN`; trạng thái này do IDP tự quyết, không lấy từ acknowledgment của CD (D5).
  - Ở A2 path, `deployment.status` chuyển từ `DEPLOYING` sang `FAILED` và `deployment_record.error_summary` khác `NULL`; `delivery_reference` được giữ nếu provider đã cấp trước khi lỗi.
  - Tiến trình theo tầng/thành phần được thể hiện qua các row `deployment_step` (có `wave_number`, `related_component_reference`); ai ghi các row này và ghi lúc nào để lại cho D4.
  - Không `Resource Output`, `Workload Output`, `Resolved Configuration`, resolved secret hoặc plaintext Secret nào được persist trong `deployment_record` hay `deployment_step`.
- **Exceptions / Guarantees**:
  - A2 được persist ngay cả khi failure xảy ra ở infrastructure provisioning, manifest generation/materialization, CD delivery, chờ workload healthy hoặc bước gỡ/hủy; record giữ tầng, thành phần liên quan và error summary để UC-04 đọc được.
  - Việc upsert `deployment_record`, đổi `deployment.status` và ghi `deployment_record_resource_instance` là một transaction: nếu repository write thất bại, không để lại record/association chỉ được persist một phần.

## 10. `collectWorkloadOutputs()`

- **Operation**: `collectWorkloadOutputs(target, workloads)`
- **Cross References**: UC-03 – Deploy Application, bước Deployment Worker thu thập Workload Output của các workload vừa healthy trong một tầng, hoặc của workload phụ thuộc đang chạy ngoài phạm vi; A2 – workload thất bại.
- **Preconditions**:
  - Deployment đang ở `status = DEPLOYING`.
  - Với workload thuộc tầng vừa triển khai: `waitForWorkloadsHealthy(target, workloads)` đã xác nhận mọi pod của các workload đó healthy trên `target`.
  - Với workload phụ thuộc ngoài phạm vi: `workload_instance.status = HEALTHY` trên cùng `(environment, target)`.
  - Mỗi output cần đọc được khai báo trong `Workload.exposedOutputs` của phiên bản mà workload đang chạy.
- **Postconditions**:
  - Một tập `Workload Output` `TRANSIENT` được tạo cho execution hiện tại, mỗi phần tử gồm `workloadInstanceId`, `outputName`, `resolvedValue`, dùng làm input cho `resolveEnvironmentConfiguration()` của các tầng sau và cho `propagateOutputChanges()`.
  - Không table/row `workload_output` nào được tạo; `workload_instance.output_fingerprint` không bị đổi bởi operation này.
  - Không workload, resource hay deployment state nào bị thay đổi; operation chỉ đọc.
- **Exceptions / Guarantees**:
  - A2: nếu workload không healthy hoặc output khai báo không đọc được, Deployment Worker không triển khai các tầng sau và `deployment.status` chuyển sang `FAILED`.
  - Cơ chế cụ thể để đọc Workload Output từ workload đang chạy chưa được chốt (mục hoãn); contract chỉ quy định kết quả, không quy định cách đọc.

## 11. `propagateOutputChanges()`

- **Operation**: `propagateOutputChanges(waves, waveComponents, collectedOutputs)`
- **Cross References**: UC-03 – Deploy Application, bước sau mỗi tầng Deployment Worker so sánh dấu vân tay output và tự động làm lại các resource và workload phụ thuộc ở các tầng sau.
- **Preconditions**:
  - Deployment đang ở `status = DEPLOYING`; mọi thành phần trong `waveComponents` (kể cả cụm Kubernetes và network) đã sẵn sàng (resource `READY`, workload healthy) và `collectedOutputs` chứa output của chúng.
  - Dấu vân tay output lần trước được đọc từ `resource_instance.output_fingerprint` và `workload_instance.output_fingerprint` theo khóa chủ sở hữu tương ứng.
- **Postconditions**:
  - Với mỗi thành phần trong `waveComponents`, dấu vân tay mới (SHA-256 của output đã chuẩn hóa) được tính và so với dấu vân tay lần trước; `resource_instance.output_fingerprint` / `workload_instance.output_fingerprint` được cập nhật thành giá trị mới, và `workload_instance.status = HEALTHY` với workload của tầng.
  - Nếu dấu vân tay của một resource (kể cả `k8s-cluster`, `network`) thay đổi, mọi resource `MANAGED` của cùng application, environment và target có definition `requires` resource đó và chưa được reconcile sau thay đổi này được đưa vào các tầng sau của `waves` với action update, kể cả khi chúng nằm ngoài phạm vi; chúng được reconcile lại bằng contract 6 với output mới.
  - Nếu dấu vân tay của một thành phần thay đổi, mọi Workload (bắc cầu) của cùng application, environment và target depends on thành phần đó (Dependency của phiên bản, hoặc chạy trên `k8s-cluster` đó), chưa nằm trong phạm vi và đang có `workload_instance.status = HEALTHY` được thêm vào các tầng sau của `waves`; với mỗi workload như vậy, một row `workload_deployment` được insert với `image_repository`/`image_version` lấy từ `current_workload_deployment_id` đang chạy, `inclusion_reason = CASCADED` và `wave_number` tương ứng.
  - Nếu không có dấu vân tay nào thay đổi, `waves` không đổi, không resource nào được đưa vào lại và không row `workload_deployment` nào được thêm.
  - Chỉ dấu vân tay được lưu; giá trị output không được persist.
- **Exceptions / Guarantees**:
  - Thành phần được thêm do lan truyền phải thuộc cùng phiên bản đang chạy trên environment/target; lan truyền không vượt sang application khác. Việc output của resource dùng chung (`EXISTING`) thay đổi không tự deploy lại các application khác (mục hoãn).
  - Resource `EXISTING` (resource dùng chung, cụm nội bộ) không bao giờ được reconcile lại; nếu output của nó đổi thì chỉ các thành phần dựa trên nó được làm lại.
  - Lan truyền dừng khi không còn dấu vân tay nào thay đổi; mỗi workload được thêm tối đa một lần và mỗi resource được reconcile lại tối đa một lần cho mỗi thay đổi của thành phần nó `requires` trong một deployment.

## 12. `createTeardown()`

- **Operation**: `createTeardown(applicationId, environment, target)`
- **Cross References**: UC-05 – Remove Application from Environment, từ bước Developer chọn environment và nơi triển khai đến bước hiển thị plan gỡ bỏ; A1 – Không có gì để gỡ hoặc đang có deployment khác chạy dở.
- **Preconditions**:
  - `application_definition.application_id = applicationId` định danh một `Application Definition` hợp lệ; `environment` là `STAGING` hoặc `PRODUCTION`.
  - Không có row `deployment` nào của cùng `(application_id, environment, deployment_target)` ở `status = CONFIRMED` hoặc `DEPLOYING`. Một row ở `AWAITING_CONFIRMATION` **không** chặn: plan chưa xác nhận không giữ chỗ trên chủ sở hữu, và nó sẽ bị phát hiện là lỗi thời qua `plan_fingerprint` nếu sau đó có ai xác nhận nó.
  - Tồn tại ít nhất một deployment trước đó của đúng `(applicationId, environment, target)`. Deployment gần nhất trong số đó cung cấp phiên bản Application Definition (để lấy tên thành phần), phiên bản catalog (để resolve `Resource Definition` của các instance đang tồn tại) và Deployment Context. `environment_configuration_id` của nó được chép sang row mới cho Deployment Record; **giá trị cấu hình không được đọc**, vì gỡ bỏ dựa trên những gì đang tồn tại chứ không dựa trên cấu hình. Operation **không** nhận `version`, `catalogVersion`, `images` hay `context` từ Developer.
  - Còn ít nhất một `Workload Instance` chưa ở trạng thái đã gỡ, hoặc một `Resource Instance` chưa ở trạng thái đã hủy/gỡ liên kết, thuộc đúng khóa chủ sở hữu `(application_id, environment, resource_requirement_id, deployment_target)`.
- **Postconditions**:
  - Một instance `Deployment` được tạo; một row `deployment` được tạo với `kind = TEARDOWN`, `status = AWAITING_CONFIRMATION`, cùng `application_definition_version_id`, `catalog_version_id`, `environment_configuration_id`, `environment`, `deployment_target`, `plan_fingerprint`, `plan_fingerprint_algo` lấy theo deployment gần nhất của cùng chủ sở hữu.
  - **Không** có `Workload Deployment` nào được tạo: gỡ bỏ không triển khai workload nào.
  - Một `Deployment Graph` `TRANSIENT` được dựng ở dạng **chỉ gồm node**: các Workload và Resource Requirement mà phiên bản khai báo, kèm tên, không resolve definition và không có cạnh phụ thuộc — gỡ bỏ chỉ cần biết tên và những gì đang tồn tại. Rồi một plan gỡ bỏ `TRANSIENT` được tạo: tầng đầu là toàn bộ Workload đang chạy của chủ sở hữu; các tầng sau là Resource Instance, xếp theo quy tắc **một resource chỉ được gỡ khi không còn resource nào chưa gỡ có `resource_definition.requires` chứa loại của nó**, nên cụm Kubernetes và network nằm ở tầng cuối. Hành động là hủy khi `Resource Definition.managementMode = MANAGED` và gỡ liên kết khi `= EXISTING`; mỗi thành phần bị hủy mang cảnh báo mất dữ liệu.
  - `plan_fingerprint` và `plan_fingerprint_algo` được tính và persist như với `createDeployment()`, nên `confirmDeployment()` phát hiện được plan đã đổi giữa lúc xem và lúc xác nhận.
  - Không `Resource Instance`, `Workload Instance`, namespace, desired state hay hạ tầng nào bị thay đổi; không `Deployment Execution Job`, `Deployment Record` hay `Deployment Step` nào được tạo.
- **Exceptions / Guarantees**:
  - A1: nếu application chưa từng được deploy lên `(environment, target)`, nếu không còn gì để gỡ, hoặc nếu đang có deployment của cùng chủ sở hữu đã được xác nhận hay đang chạy, không aggregate `Deployment` nào tồn tại; row phát sinh trong attempt được rollback và hạ tầng không thay đổi.
  - Việc thực thi thuộc `confirmDeployment()` (dùng chung với UC-03) và Deployment Worker; operation này chỉ lập plan.
- **Scope boundaries**:
  - Operation chỉ tác động tới đúng một `(application, environment, deployment target)`. Application Definition, các phiên bản, Environment Configuration của mọi environment, `Delivery Repository` và cặp khóa của application không nằm trong postcondition của bất kỳ contract nào của UC-05: chúng được giữ lại.
