# Step 4: Operation Contracts

Tài liệu này đặc tả các system operation quan trọng của UC-01 đến UC-03 theo kiểu Larman. Tên domain object dùng đúng Step 2; tên table/column `snake_case` dùng đúng Step 3. Các nhãn như `AWAITING_CONFIRMATION`, `CONFIRMED`, `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `SUBMITTED` và `FAILED` mô tả **logical state** của thuộc tính `status`; ERD hiện chỉ chốt kiểu `ENUM`, chưa chốt tập literal vật lý.

Các execution-scoped object `Deployment Graph`, `Resource Resolution`, Infrastructure Plan, `Resource Output`, `Resolved Configuration` và `Resolved Specification` là `TRANSIENT`; postcondition có thể tạo chúng trong execution hiện tại nhưng không tạo table/row tương ứng. Mọi postcondition bên dưới mô tả state sau khi operation hoàn tất, không mô tả trình tự gọi component.

## 1. `saveApplicationDefinition()`

- **Operation**: `saveApplicationDefinition(applicationDefinition)`
- **Cross References**: UC-01 – Create / Configure Application, luồng chính và A1 – Dữ liệu không hợp lệ.
- **Preconditions**:
  - Developer đã đăng nhập và có quyền tạo hoặc chỉnh sửa Application Definition được truyền vào.
  - Nếu là update, một instance `Application Definition` với `applicationId` tương ứng đã tồn tại trong `application_definition`; nếu là create, `applicationId` chưa định danh một row khác và `name` chưa được dùng bởi Application Definition khác.
  - `applicationDefinition` có ít nhất một `Workload`. Trong cùng Application Definition, `Workload.name` và `Resource Requirement.name` không trùng; mỗi `Workload.imageRepository` hợp lệ; mỗi `Workload.port`, nếu có, nằm trong khoảng `1..65535`.
  - Mỗi `Environment Variable Definition` và `Secret Definition` thuộc đúng một Workload của Application Definition; tên definition là duy nhất trong Workload tương ứng.
  - Mỗi `Dependency` có một source là Workload của Application Definition và đúng một target là `Workload` hoặc `Resource Requirement` cùng Application Definition; dependency logic không bị trùng.
  - Definition không chứa image tag/version, Environment Configuration hoặc deployment target; các dữ liệu này không thuộc UC-01.
- **Postconditions**:
  - Khi create, một instance `Application Definition` được tạo; một row `application_definition` tồn tại với `application_id`, `name`, `description`, `created_at` và `updated_at` tương ứng.
  - Khi update, các thuộc tính `name`, `description` và `updatedAt` của `Application Definition` được sửa; các column `name`, `description`, `updated_at` của row `application_definition` tương ứng có giá trị mới, còn `applicationId`/`application_id` không đổi.
  - Tập instance `Workload`, `Resource Requirement`, `Environment Variable Definition`, `Secret Definition` và `Dependency` thuộc Application Definition bằng đúng tập trong definition đã submit: instance mới được tạo, instance còn tồn tại được cập nhật, và instance đã bị loại khỏi definition được xóa.
  - Các row `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition` và `dependency` phản ánh đúng thuộc tính của các instance trên, bao gồm `image_repository` nhưng không bao gồm image version.
  - Association ownership giữa `Application Definition` và từng `Workload`, `Resource Requirement`, `Dependency` được hình thành qua `application_id`; association giữa `Workload` và từng `Environment Variable Definition`/`Secret Definition` được hình thành qua `workload_id`.
  - Với mỗi `Dependency`, association tới source Workload được hình thành qua `source_workload_id`; đúng một association target được hình thành qua `target_workload_id` hoặc `target_resource_requirement_id` theo `target_type`. Các association không còn trong definition bị phá vỡ khi row tương ứng bị xóa.
  - Không instance `Deployment`, `Workload Deployment`, `Environment Configuration` hoặc `Resource Instance` nào được tạo, xóa hay sửa; deployment hiện tại không tự động thay đổi.
  - Việc tạo/cập nhật `Application Specification` không thuộc state change trực tiếp của contract này; đó là postcondition của `generateApplicationSpecification()` kế tiếp trong luồng UC-01.
- **Exceptions / Guarantees**:
  - A1: nếu validation thất bại do tên trùng, image repository/port không hợp lệ hoặc dependency tham chiếu sai, không instance/row nào của aggregate `Application Definition` bị tạo, sửa hoặc xóa; association hiện có được giữ nguyên.
  - Save aggregate có tính atomic: không tồn tại trạng thái chỉ lưu một phần Workload, definition hoặc Dependency.

## 2. `generateApplicationSpecification()`

- **Operation**: `generateApplicationSpecification(applicationDefinition)`
- **Cross References**: UC-01 – Create / Configure Application, bước sinh/cập nhật application specification sau khi lưu Application Definition.
- **Preconditions**:
  - `applicationDefinition` là một `Application Definition` hợp lệ đã được persist trong `application_definition`, cùng toàn bộ Workload, Resource Requirement, configuration requirement và Dependency hiện hành.
  - Application Definition có ít nhất một Workload và thỏa các invariant của UC-01; không chứa image tag/version, Environment Configuration hoặc deployment target.
  - Nếu một `Application Specification` hiện hành đã tồn tại, row `application_specification.application_id` tham chiếu đúng `application_definition.application_id` của input.
- **Postconditions**:
  - Nếu chưa có specification, một instance `Application Specification` được tạo và association `Application Definition` → `Application Specification` được hình thành; một row `application_specification` được tạo với `application_id` tương ứng.
  - Nếu specification đã tồn tại, chính instance/row hiện hành được cập nhật; không tạo specification thứ hai cho cùng `application_id`, phù hợp constraint `UNIQUE` và cardinality `0..1`.
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
  - Một `Application Definition` với `configuration.applicationId` đã tồn tại trong `application_definition` và có ít nhất một `Environment Variable Definition` hoặc `Secret Definition` cần cấu hình.
  - `configuration.environment` xác định đúng một environment; nếu update, một `Environment Configuration` cho cặp `(applicationId, environment)` đã tồn tại, phù hợp `UNIQUE (application_id, environment)`.
  - Mỗi `Environment Variable` tham chiếu một `Environment Variable Definition` và Workload sở hữu definition đó trong cùng Application Definition; mỗi definition bắt buộc có đúng một `Configuration Value`.
  - `Direct Configuration Value` chỉ được dùng cho Environment Variable thông thường. Mỗi `Resource Output Reference` tham chiếu một `Resource Requirement` cùng Application Definition và một output có trong `Resource Definition.exposedOutputs`; mỗi `Workload Output Reference` tham chiếu một Workload cùng Application Definition và một output có trong `Workload.exposedOutputs`.
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
  - A1: nếu thiếu giá trị bắt buộc, resource/workload không tồn tại hoặc output không được expose, toàn bộ state trong `environment_configuration`, `environment_variable`, `configuration_value` và `secret` giữ nguyên như trước operation.
  - Plaintext Secret không xuất hiện trong `Environment Configuration Repository`, `Deployment Repository` hoặc log của contract; chỉ opaque `secret_ref` hoặc sensitive Resource Output Reference được persist.

## 4. `createDeployment()`

- **Operation**: `createDeployment(applicationId, environment, target, images, context)`
- **Cross References**: UC-03 – Deploy Application, từ bước chọn image/context đến bước hiển thị infrastructure plan; A1 – Deployment input hoặc dependency không hợp lệ.
- **Preconditions**:
  - `application_definition.application_id = applicationId` định danh một `Application Definition` hợp lệ có ít nhất một Workload; mỗi Workload cần deploy có `image_repository` hợp lệ.
  - Một row `environment_configuration` tồn tại cho đúng `(applicationId, environment)` và mọi `Environment Variable Definition.required = true`/`Secret Definition.required = true` đều có binding hợp lệ.
  - `images` chứa đúng một image version hợp lệ cho mỗi Workload được deploy; repository của image tương ứng lấy từ `workload.image_repository`, không từ UC-01 dưới dạng version.
  - `target` được platform hỗ trợ; `context` chứa `cloudProvider`, `region` và `targetSpecificInput` bắt buộc đối với target đó.
  - Mọi Dependency thuộc cùng Application Definition và có thể resolve; với mỗi `Resource Requirement` trong graph, Platform Resource Definition Catalog có một `Resource Definition` phù hợp với `resourceType` và `supportedContexts`, cùng `provisionerReference` hợp lệ.
- **Postconditions**:
  - Một instance `Deployment` được tạo; một row `deployment` được tạo với `deployment_id`, `application_id`, `environment_configuration_id`, `environment`, `deployment_target`, `plan_fingerprint`, `plan_fingerprint_algo`, `created_at`, `updated_at`, và `status` mang logical state `AWAITING_CONFIRMATION`.
  - Association `Deployment` → `Application Definition` và `Deployment` → `Environment Configuration` được hình thành qua `application_id` và `environment_configuration_id`; environment của configuration, snapshot `deployment.environment` và input `environment` bằng nhau.
  - Với mỗi Workload được deploy, đúng một instance `Workload Deployment` và một row `workload_deployment` được tạo; association tới Deployment và Workload được hình thành qua `deployment_id`, `workload_id`; `image_repository` là repository thực tế và `image_version` là version thực tế từ `images`.
  - Đúng một instance `Deployment Context` và một row `deployment_context` được tạo cho Deployment; association 1:1 được hình thành qua `deployment_id`, và `cloud_provider`, `region`, `target_specific_input` phản ánh `context`.
  - Một instance `Deployment Graph` `TRANSIENT` được tạo cho execution hiện tại với `deploymentId`, `workloadIds`, `resourceRequirementIds`, `dependencyIds`, `configurationReferenceIds`; association execution-scoped tới Deployment Context, Workload, Resource Requirement, Dependency và Configuration Value tương ứng được hình thành.
  - Với mỗi Resource Requirement trong graph, một instance `Resource Resolution` `TRANSIENT` được tạo và liên kết tới đúng `Resource Definition`; execution state chứa infrastructure plan phân loại mỗi resource là create, update hoặc reuse và danh sách override được phép.
  - Infrastructure plan được canonicalize bằng stable resource/key ordering và canonical JSON, normalize unit/number, đồng thời loại timestamps và auto-generated ID không mang ý nghĩa nghiệp vụ. Fingerprint domain bao phủ action `CREATE`/`UPDATE`/`REUSE` của từng resource, toàn bộ Resource Definition liên quan bao gồm catalog version, Resource Instance identity được tham chiếu, provisioner reference, resolved parameters, allowed-overrides definition (tập key được phép cùng min-max/enum), và deployment target.
  - `Deployment.planFingerprint`/`deployment.plan_fingerprint` được gán SHA-256 hex của canonical plan; `Deployment.planFingerprintAlgo`/`deployment.plan_fingerprint_algo` được gán phiên bản thuật toán tương ứng, ví dụ `sha256-v1`. Cả hai được persist atomically cùng aggregate, và infrastructure plan + allowed overrides + fingerprint + algorithm version được trả về UI.
  - Chưa có `Resource Instance` nào bị create/update/reuse bởi operation này; chưa có `Deployment Record`, `Deployment Step`, desired deployment state hay CD delivery reference nào được tạo.
- **Exceptions / Guarantees**:
  - A1: nếu image version, required configuration, dependency, output reference, Resource Definition hoặc context không hợp lệ, không aggregate `Deployment` hoàn chỉnh nào tồn tại; các row `deployment`, `workload_deployment`, `deployment_context` phát sinh trong attempt được rollback và infrastructure không thay đổi.
  - Không có row độc lập cho `Deployment Graph`, `Resource Resolution` hoặc infrastructure plan; các object này chỉ tồn tại trong execution state. Chỉ `plan_fingerprint` và `plan_fingerprint_algo`, không phải plan payload, được persist trên `deployment`.

## 5. `confirmDeployment()`

- **Operation**: `confirmDeployment(deploymentId, overrides)`
- **Cross References**: UC-03 – Deploy Application, bước Developer đặt permitted overrides và chọn Deploy; A1 – Deployment input hoặc dependency không hợp lệ.
- **Preconditions**:
  - Một `Deployment` với `deploymentId` tồn tại, có `status` mang logical state `AWAITING_CONFIRMATION`, có đúng một `Deployment Context`, ít nhất một `Workload Deployment`, và có `planFingerprint`/`planFingerprintAlgo` đã persist.
  - Các input bền vững để rebuild plan còn tồn tại và nhất quán: Application Definition được tham chiếu, Environment Configuration, snapshot Workload Deployment/images, Deployment Context và deployment target; Resource Definition Catalog cùng current Resource Instance state có thể được đọc lại.
  - Không giả định execution của request `createDeployment()` hoặc infrastructure plan in-memory còn tồn tại. `overrides` là candidate values từ confirm request và chỉ được validate sau khi rebuilt-plan fingerprint khớp fingerprint đã lưu.
- **Postconditions**:
  - Infrastructure plan được rebuild từ persisted inputs và current state bằng cùng chuỗi `buildDeploymentGraph()` → `resolveResourceDefinitions()` → `findResourceInstances()` → `planInfrastructureChanges()`; allowed-overrides definition được load lại và fingerprint được tính lại bằng đúng canonicalization/hash version trong `planFingerprintAlgo`.
  - Khi rebuilt fingerprint khớp `planFingerprint`, `overrides` được validate theo allowed-overrides definition hiện hành; infrastructure plan trong execution state được thay bằng final infrastructure plan chứa đúng các override hợp lệ, còn parameter không được override giữ giá trị rebuilt plan.
  - Sau khi fingerprint khớp và overrides hợp lệ, `deployment.status` được đổi bằng một atomic compare-and-swap có predicate `deployment_id = deploymentId AND status = AWAITING_CONFIRMATION`; khi CAS ảnh hưởng đúng một row, `Deployment.status`/`deployment.status` trở thành `CONFIRMED` và `updatedAt`/`updated_at` được cập nhật.
  - Chỉ sau CAS thành công, Orchestrator mới gọi `reconcileInfrastructure(finalPlan)`; không có reconcile nào được khởi động trước guard này.
  - Association của Deployment với `Application Definition`, `Environment Configuration`, `Workload Deployment` và `Deployment Context` không đổi; image version, environment và deployment target đã snapshot không đổi.
- **Exceptions / Guarantees**:
  - `PLAN_CHANGED`: nếu rebuilt fingerprint khác `deployment.plan_fingerprint`, không apply overrides và không reconcile infrastructure; Deployment giữ `AWAITING_CONFIRMATION`. Hệ thống atomically refresh fingerprint lưu trên Deployment cho rebuilt plan khi stored fingerprint vẫn là giá trị vừa so sánh, rồi trả error cùng rebuilt plan, allowed overrides và rebuilt fingerprint để Developer review và re-confirm; nếu refresh cạnh tranh thất bại, request vẫn không reconcile và lần confirm sau sẽ rebuild/compare lại.
  - A1: nếu plan không rebuild được hoặc override không được phép/không hợp lệ sau khi fingerprint khớp, Deployment giữ logical state `AWAITING_CONFIRMATION`, final infrastructure plan không được hình thành, CAS không chạy và infrastructure không thay đổi.
  - `DEPLOYMENT_ALREADY_CONFIRMED`: nếu status ban đầu không còn là `AWAITING_CONFIRMATION`, hoặc atomic status CAS ảnh hưởng zero rows do confirmation đồng thời đã thắng, request bị từ chối, không tạo Deployment/Workload Deployment mới và không gọi reconcile từ losing request.
- **Scope boundaries**:
  - Fingerprint bao phủ **allowed-overrides definition** (tập parameter được phép cùng min-max/enum), không bao phủ các override value Developer chọn; các value này chỉ đến trong confirm request và được validate sau khi fingerprint khớp.
  - Fingerprint là application-level defense-in-depth guard: nó thu hẹp nhưng không loại bỏ TOCTOU. Contention trên shared Resource Instance giữa các deployment và drift trong lúc reconcile cần cơ chế bổ sung như Resource-Instance-level version/optimistic lock hoặc per-resource reconcile lock, cùng provisioner idempotency (ví dụ Terraform refresh + plan); các cơ chế bổ sung này nằm ngoài phạm vi fingerprint.

## 6. `reconcileInfrastructure()`

- **Operation**: `reconcileInfrastructure(finalPlan)`
- **Cross References**: UC-03 – Deploy Application, bước reconcile infrastructure; A2 – Provisioning hoặc delivery thất bại.
- **Preconditions**:
  - `finalPlan` thuộc một Deployment đã confirm và chứa đúng một quyết định create, update hoặc reuse cho mỗi Resource Requirement cần thiết trong `Deployment Graph`.
  - Mỗi item của plan đã resolve tới một `Resource Definition` có `resourceDefinitionId`, `provisionerReference` và `supportedContexts` phù hợp với `Deployment Context`/`deploymentTarget`.
  - Với item update/reuse, `resource_instance.resource_instance_id` mục tiêu đã tồn tại, tham chiếu đúng `resource_definition` và `deployment_target`; với item create, chưa có `infrastructure_reference` trùng.
  - Provisioner Adapter và Terraform/OpenTofu Runner tương ứng sẵn sàng; operation chưa thu thập hoặc sử dụng Resource Output trước khi Resource Instance ready.
- **Postconditions**:
  - Với mỗi item create thành công, một instance `Resource Instance` và một row `resource_instance` được tạo; association tới `Resource Definition` được hình thành qua `resource_definition_id`; `deployment_target`, `infrastructure_reference`, `provider_state_reference`, `created_at`, `updated_at` được gán và `status` biểu thị resource ready.
  - Với mỗi item update thành công, thuộc tính `infrastructureReference`, `providerStateReference`, `status`, `updatedAt` của `Resource Instance` được sửa theo provider state thực tế; các column tương ứng của `resource_instance` được cập nhật và `created_at` không đổi.
  - Với mỗi item reuse, association tới Resource Instance hiện hữu được giữ; identity và `infrastructure_reference` không đổi, `status` được xác nhận là ready và `updated_at` chỉ thay đổi nếu trạng thái provider được refresh.
  - Execution result chứa `infrastructureReferences` trỏ tới đúng các Resource Instance create/update/reuse; thuộc tính `Deployment.status` được sửa thành logical state `INFRASTRUCTURE_READY` và `deployment.updated_at` được cập nhật khi toàn bộ plan thành công.
  - Không row `resource_output` được tạo: `Resource Output` vẫn là runtime view `TRANSIENT` và chỉ được collector nạp sau khi Resource Instance ready.
- **Exceptions / Guarantees**:
  - A2: Resource Instance thất bại không có `status` ready; nếu đã có durable provider/resource state thì `resource_instance.status`, reference khả dụng và `updated_at` phản ánh state thực tế thay vì báo thành công giả.
  - Khi bất kỳ plan item thất bại, `Deployment.status` mang logical state `FAILED`; failed step, `relatedComponentReference` và `errorSummary` tồn tại trong execution result để `saveDeploymentRecord()` persist. Các Resource Instance đã reconcile thành công trước lỗi không bị tuyên bố rollback nếu provider không hỗ trợ rollback.

## 7. `resolveEnvironmentConfiguration()`

- **Operation**: `resolveEnvironmentConfiguration(configuration, resourceOutputs, workloadOutputs)`
- **Cross References**: UC-03 – Deploy Application, bước resolve Environment Configuration và dependency; A1 – reference không hợp lệ; A2 – lỗi trong deployment execution.
- **Preconditions**:
  - `configuration` là Environment Configuration đã persist và được Deployment hiện tại tham chiếu; `configuration.applicationId` và `configuration.environment` khớp `deployment.application_id` và `deployment.environment`.
  - Mỗi direct Environment Variable có một `Direct Configuration Value`; mỗi output-based variable có đúng một `Resource Output Reference` hoặc `Workload Output Reference`; mỗi Secret có đúng một `secretReference` hoặc sensitive `Resource Output Reference`.
  - Mỗi Resource Output Reference trỏ tới Resource Requirement đã được resolve thành Resource Instance ready; `resourceOutputs` chứa đúng `resourceInstanceId` + `outputName` cần thiết và cờ `sensitive` phù hợp.
  - Mỗi Workload Output Reference trỏ tới Workload trong Deployment Graph và `workloadOutputs` chứa logical output đã khai báo trong `Workload.exposedOutputs`.
  - Mọi `secretReference` còn hợp lệ và có thể được Secret Store/Secret Materializer sử dụng mà không persist plaintext.
- **Postconditions**:
  - Một instance `Resolved Configuration` `TRANSIENT` được tạo với `deploymentId` của Deployment hiện tại; association execution-scoped `Resolved Configuration` → `Environment Configuration` được hình thành.
  - `resolvedEnvironmentVariables` chứa một entry cho mỗi `Environment Variable`: direct value được lấy từ `Direct Configuration Value`, Resource Output được lấy theo `resourceRequirementId` + `outputName`, và Workload Output được lấy theo `workloadId` + `outputName`.
  - `resolvedSecrets` chứa secret value/reference phù hợp cho mỗi `Secret` và duy trì phân loại sensitive; plaintext Secret không trở thành thuộc tính của bất kỳ persistent domain object nào.
  - `resolvedDependencies` chứa mapping đã resolve cho các Dependency/configuration reference của Deployment Graph; association execution-scoped tới các `Resource Output` đã consume được hình thành.
  - Thuộc tính `Deployment.status` được sửa thành logical state `CONFIGURATION_RESOLVED` và `deployment.updated_at` được cập nhật khi toàn bộ configuration được resolve thành công.
  - Không row trong `environment_configuration`, `environment_variable`, `configuration_value` hoặc `secret` bị thay đổi; logical references không bị thay thế bằng resolved values. Không table/row `resource_output` hoặc `resolved_configuration` được tạo.
- **Exceptions / Guarantees**:
  - A1: reference tới resource/workload/output không hợp lệ hoặc required binding bị thiếu thì không có instance `Resolved Configuration` hoàn chỉnh và persistent Environment Configuration giữ nguyên.
  - A2: nếu output/secret reference hợp lệ về cấu trúc nhưng không resolve được tại runtime, `Deployment.status` mang logical state `FAILED`; failed step và error summary được chuyển cho `saveDeploymentRecord()`. Resolved secret không được ghi vào Deployment Record hoặc error detail.

## 8. `publishDesiredDeploymentState()`

- **Operation**: `publishDesiredDeploymentState(desiredState)`
- **Cross References**: UC-03 – Deploy Application, bước publish desired deployment state tới CD abstraction; A2 – Provisioning hoặc delivery thất bại.
- **Preconditions**:
  - Deployment hiện tại đã có `Resolved Configuration` và `Resolved Specification` hoàn chỉnh; mỗi `Workload Deployment.imageRepository` + `imageVersion` đã xuất hiện đúng trong resolved specification.
  - Base Kubernetes manifest đã được sinh bởi `score-k8s`, target-specific adaptation đã được áp dụng cho `deploymentTarget`, và Environment Variable/Secret đã được materialize đầy đủ vào `desiredState`.
  - `desiredState` không chứa unresolved Resource Output Reference hoặc Workload Output Reference; secret material tuân thủ cơ chế Kubernetes Secret hoặc secret reference mà không làm lộ plaintext ngoài delivery boundary.
  - CD Integration đã resolve được một Concrete CD Provider cho target và Deployment chưa có một publish thành công xung đột với cùng desired-state identity.
- **Postconditions**:
  - Trong CD System, desired deployment state cho application/environment/target được tạo hoặc cập nhật và association delivery tới Kubernetes deployment target được hình thành.
  - Execution state nhận một `deliveryReference` định danh durable desired-state/CD delivery và một `deliveryStatus` phản ánh ít nhất việc CD System đã accept/synchronization đã bắt đầu.
  - Thuộc tính `Deployment.status` được sửa thành logical state `SUBMITTED` và `deployment.updated_at` được cập nhật; trạng thái này không khẳng định workload đã Healthy hoặc CD sync đã hoàn tất.
  - Không `Deployment Record` hoặc association `deployment_record_resource_instance` nào được tạo bởi operation này; `deliveryReference` và `deliveryStatus` chỉ được persist bởi `saveDeploymentRecord()` kế tiếp.
  - Không Application Definition, Environment Configuration hoặc Resource Instance nào bị thay đổi bởi việc publish.
- **Exceptions / Guarantees**:
  - A2: nếu manifest generation/materialization trước publish hoặc CD delivery thất bại, không có success status giả; `Deployment.status` mang logical state `FAILED`, và delivery reference (nếu provider đã cấp), failed step cùng error summary được giữ trong execution result cho `saveDeploymentRecord()`.
  - Việc CD System accept desired state không đồng nghĩa Kubernetes workload đã ready; contract chỉ bảo đảm delivery reference/status phản ánh đúng acknowledgment nhận được.

## 9. `saveDeploymentRecord()`

- **Operation**: `saveDeploymentRecord(environment, target, images, infrastructureReferences, deliveryStatus)`
- **Cross References**: UC-03 – Deploy Application, bước lưu Deployment Record; A2 – Provisioning hoặc delivery thất bại; UC-04 – View Deployment Result (nguồn dữ liệu được đọc về sau).
- **Preconditions**:
  - Orchestration context xác định đúng một `Deployment` hiện hành; row `deployment` tồn tại, có `environment = environment`, `deployment_target = target`, đúng một Deployment Context và ít nhất một Workload Deployment.
  - `images` bằng snapshot đã persist trong các row `workload_deployment` của Deployment: mỗi Workload có đúng `image_repository` và `image_version` thực tế.
  - Mỗi item trong `infrastructureReferences` định danh một row `resource_instance` durable đã create/update/reuse cho execution này; không truyền resolved Resource Output.
  - Ở success path, `deliveryStatus` chứa status và `deliveryReference` do CD Integration trả về. Ở A2 path, orchestration context chứa status thất bại, failed step, `relatedComponentReference` nếu có và `errorSummary`; `deliveryReference` có thể không tồn tại.
- **Postconditions**:
  - Nếu Deployment chưa có record, một instance `Deployment Record` và một row `deployment_record` được tạo; association 1:0..1 `Deployment` → `Deployment Record` được hình thành qua `deployment_id`.
  - Nếu record đã tồn tại cho `deployment_id`, chính record đó được cập nhật; constraint `UNIQUE (deployment_id)` bảo đảm không tạo record thứ hai cho cùng Deployment.
  - `Deployment Record.environment`/`environment`, `deploymentTarget`/`deployment_target`, `deliveryReference`/`delivery_reference`, `status`, `errorSummary`/`error_summary`, `createdAt`/`created_at`, `updatedAt`/`updated_at` phản ánh kết quả thực tế của execution; `created_at` không đổi khi update.
  - Association giữa `Deployment Record` và toàn bộ `Workload Deployment` của Deployment được hình thành ở domain model để record actual images; trong ERD, images được truy xuất qua `deployment_record.deployment_id` → `workload_deployment.deployment_id`, không tạo bản sao image hoặc association table mới.
  - Với mỗi Resource Instance được tham chiếu, association `Deployment Record` ↔ `Resource Instance` được hình thành bằng đúng một row `deployment_record_resource_instance`; association cũ không còn thuộc execution result bị phá vỡ khi record được cập nhật.
  - Ít nhất một instance `Deployment Step` và row `deployment_step` tồn tại; association tới đúng Deployment Record và Deployment được hình thành qua `deployment_record_id`, `deployment_id`; `sequence_number`, `step_name`, `status`, `related_component_reference`, `error_summary`, `started_at`, `completed_at` phản ánh progress thực tế.
  - Thuộc tính `Deployment.status` và `deployment.updated_at` được cập nhật đồng nhất với trạng thái current/final của Deployment Record.
  - Ở success path, `deployment_record.status` phản ánh delivery status đã nhận, `delivery_reference` khác `NULL`, và `error_summary = NULL`; các successful Deployment Step có status/timestamps tương ứng.
  - Ở A2 path, `deployment_record.status` và `deployment.status` mang logical state `FAILED`, `deployment_record.error_summary` khác `NULL`; Deployment Step bị lỗi có `status = FAILED`, `error_summary` khác `NULL`, và `related_component_reference` chỉ tới workload/resource liên quan nếu xác định được. `delivery_reference` được giữ nếu provider đã cấp trước khi lỗi, nếu không thì bằng `NULL`.
  - Không `Resource Output`, `Resolved Configuration`, resolved secret hoặc plaintext Secret nào được persist trong `deployment_record` hay `deployment_step`.
- **Exceptions / Guarantees**:
  - A2 được persist ngay cả khi failure xảy ra ở infrastructure provisioning, manifest generation/materialization hoặc CD delivery; record giữ failed step và error summary để UC-04 đọc được.
  - Việc upsert `deployment_record`, đồng bộ `deployment.status`, các `deployment_step` và `deployment_record_resource_instance` là một transaction: nếu repository write thất bại, không để lại record/step/association chỉ được persist một phần.
