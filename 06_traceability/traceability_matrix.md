# Step 6: Traceability

Tài liệu này là acceptance check nối các artifact đã chốt từ Use Case Realization, VOPC/Design Class Diagram, Domain Model, Database/ERD, Operation Contracts đến State Machines.

Quy ước đọc matrix:

- **Class sở hữu** là class nhận đúng message mang tên system operation trong sequence diagram và khai báo operation đó trong VOPC. Khi cùng message đi qua Controller và Service, cả hai class nhận message được ghi lại; class xử lý nghiệp vụ nằm sau dấu `→`.
- **Bảng DB liên quan** chỉ dùng tên table có thật trong Step 3. Hậu tố “deferred” nghĩa là operation sửa draft, còn write vật lý chỉ xảy ra ở operation `save...()` sau đó. Dấu `—` nghĩa là artifact hiện tại không chỉ ra table nào bị truy cập hoặc thay đổi bởi operation đó.
- **State ảnh hưởng** là “Có” khi operation xuất hiện như trigger/action/failure point trong một trong hai state machine ở Step 5. Operation read-only của UC-04 chỉ quan sát state nên được ghi “Không”.
- Tất cả đường dẫn sequence diagram là tương đối từ repository root.

## A. Main traceability matrix

| Use Case | Bước use case (flow step) | System Operation (Bước 2) | Sequence diagram (file + ai nhận message) | Class sở hữu (Bước 1 VOPC) | Bảng DB liên quan (Bước 3) | Operation Contract (Bước 4: có/không) | State ảnh hưởng (Bước 5: có/không) |
|---|---|---|---|---|---|---|---|
| UC-01 | Luồng chính: Developer chọn **Create Application** | `createApplication()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | `application_definition` *(draft; write deferred tới `saveApplicationDefinition()`)* | Không | Không |
| UC-01 | Luồng chính: Developer mở application hiện có để chỉnh sửa | `updateApplication()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | Đọc `application_definition`, `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` qua `Application Repository.findById()`; write deferred tới save | Không | Không |
| UC-01 | Luồng chính: **Add Resource** và khai báo name/type | `addResourceRequirement()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | `resource_requirement` *(draft; write deferred)* | Không | Không |
| UC-01 | Luồng chính: **Add Workload** và khai báo type/image repository/port | `addWorkload()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | `workload` *(draft; write deferred)* | Không | Không |
| UC-01 | Luồng chính: khai báo Environment Variable và Secret cho workload | `defineConfigurationRequirement()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | `environment_variable_definition`, `secret_definition` *(draft; write deferred)* | Không | Không |
| UC-01 | Luồng chính: khai báo quan hệ **depends on** | `defineDependency()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service | Application API / Controller → Application Service | `dependency` *(draft; write deferred)* | Không | Không |
| UC-01 | Luồng chính/A1: IDP kiểm tra Application Definition | `validateApplicationDefinition()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application Definition Validator | Application Definition Validator | — *(validate object graph trong memory)* | Không | Không |
| UC-01 | Luồng chính: Developer chọn **Save Application**, IDP lưu definition | `saveApplicationDefinition()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application API / Controller → Application Service; internal repository message là `save()` | Application API / Controller → Application Service | Ghi `application_definition`, `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` | **Có — Contract 1** | Không |
| UC-01 | Luồng chính: IDP sinh/cập nhật application specification | `generateApplicationSpecification()` | `sequence_digrams/uc_01_create_configure_application.puml`: Application Specification Generator | Application Specification Generator | Ghi `application_specification` | **Có — Contract 2** | Không |
| UC-02 | Luồng chính: Developer chọn environment cần cấu hình | `selectEnvironment()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration API / Controller → Environment Configuration Service | Environment Configuration API / Controller → Environment Configuration Service | — *(sequence chưa đọc configuration hiện có; operation kế tiếp tải requirements)* | Không | Không |
| UC-02 | Luồng chính: IDP hiển thị Variable/Secret đã khai báo trong UC-01 | `loadConfigurationRequirements()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Application Query / Application Repository | Application Query / Application Repository | Đọc `application_definition`, `workload`, `environment_variable_definition`, `secret_definition`, `dependency` | Không | Không |
| UC-02 | Luồng chính: Developer nhập direct value cho Variable hoặc Secret | `setDirectConfigurationValue()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration API / Controller → Environment Configuration Service; với Secret có internal call tới Secret Store | Environment Configuration API / Controller → Environment Configuration Service | `configuration_value` hoặc `secret` *(draft; Secret chỉ giữ `secret_ref`; write deferred)* | Không | Không |
| UC-02 | Luồng chính: Developer chọn Resource Output / sensitive Resource Output | `bindResourceOutput()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration API / Controller → Environment Configuration Service | Environment Configuration API / Controller → Environment Configuration Service | `configuration_value` hoặc `secret` *(deferred)*; output hợp lệ được query từ `resource_definition` qua catalog | Không | Không |
| UC-02 | Luồng chính: Developer chọn Workload Output | `bindWorkloadOutput()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration API / Controller → Environment Configuration Service | Environment Configuration API / Controller → Environment Configuration Service | `configuration_value` *(deferred)*; output hợp lệ lấy từ `workload.exposed_outputs` | Không | Không |
| UC-02 | Luồng chính/A1: IDP kiểm tra value và output references | `validateEnvironmentConfiguration()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration Validator | Environment Configuration Validator | — *(validate configuration draft và catalog results trong memory)* | Không | Không |
| UC-02 | Luồng chính: Developer chọn **Save Configuration**, IDP lưu theo environment | `saveEnvironmentConfiguration()` | `sequence_digrams/uc_02_configure_application_environment.puml`: Environment Configuration API / Controller → Environment Configuration Service; internal repository message là `save()` | Environment Configuration API / Controller → Environment Configuration Service | Ghi `environment_configuration`, `environment_variable`, `configuration_value`, `secret` | **Có — Contract 3** | Không |
| UC-03 | Luồng chính: chọn image/context và tạo deployment để lập infrastructure plan | `createDeployment()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment API / Controller → Deployment Orchestrator; Orchestrator canonicalize/hash plan; internal repository message persist aggregate cùng fingerprint ở `AWAITING_CONFIRMATION` | Deployment API / Controller → Deployment Orchestrator | Đọc `application_definition`, `workload`, `environment_configuration`, `resource_definition`, `resource_instance`; ghi `deployment` gồm `plan_fingerprint`, `plan_fingerprint_algo`, cùng `workload_deployment`, `deployment_context` với status `AWAITING_CONFIRMATION`; plan vẫn transient | **Có — Contract 4** | **Có — Deployment:** execution-only Created/Validating → `AWAITING_CONFIRMATION`, hoặc Rejected/A1 |
| UC-03 | Luồng chính/A1: IDP kiểm tra image, configuration và context | `validateDeploymentInput()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment Orchestrator tự nhận message | Deployment Orchestrator | Đọc state đã load từ `application_definition`, `workload`, `environment_configuration`, `environment_variable`, `configuration_value`, `secret` | Không *(nằm trong Contract 4)* | **Có — Deployment:** guard valid/invalid từ Created/Validating |
| UC-03 | Luồng chính: dựng dependency/resource graph | `buildDeploymentGraph()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment Graph Builder | Deployment Graph Builder | Đọc dữ liệu đã load từ `workload`, `resource_requirement`, `dependency`, `environment_configuration`, `configuration_value`, `secret`; `Deployment Graph` là transient, không có table | Không | **Có — Deployment:** action trước khi vào `AWAITING_CONFIRMATION` |
| UC-03 | Luồng chính: resolve Resource Definition theo graph/context | `resolveResourceDefinitions()` | `sequence_digrams/uc_03_deploy_application.puml`: Resource Definition Resolver | Resource Definition Resolver | Đọc `resource_requirement`, `resource_definition` | Không | **Có — Deployment:** action trước khi vào `AWAITING_CONFIRMATION` |
| UC-03 | Luồng chính: xác định create/update/reuse infrastructure | `planInfrastructureChanges()` | `sequence_digrams/uc_03_deploy_application.puml`: Infrastructure Planner; chạy khi create và chạy lại khi confirm trước khi tính lại fingerprint | Infrastructure Planner | Đọc `resource_instance`; plan là transient, chỉ fingerprint + algorithm được persist trên `deployment` | Không | **Có — Deployment:** action trước `AWAITING_CONFIRMATION` và guard rebuild khi confirm; **Resource Instance:** vào phase Planned (`PLANNED`) cho create/update |
| UC-03 | Luồng chính: hiển thị parameter Developer được phép override | `loadInfrastructureOverrides()` | `sequence_digrams/uc_03_deploy_application.puml`: Infrastructure Planner | Infrastructure Planner | — *(allowed overrides nằm trong transient plan)* | Không | Không |
| UC-03 | Luồng chính: Developer đặt permitted overrides | `applyInfrastructureOverrides()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment Orchestrator → Infrastructure Planner | Deployment Orchestrator → Infrastructure Planner | — *(final plan transient)* | Không *(postcondition được gộp trong Contract 5)* | **Có — Deployment:** action của transition xác nhận hợp lệ |
| UC-03 | Luồng chính/A1: Developer xác nhận và chọn **Deploy** | `confirmDeployment()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment API / Controller → Deployment Orchestrator; load persisted inputs, rebuild plan, recompute/compare fingerprint, validate overrides, rồi CAS status trước reconcile | Deployment API / Controller → Deployment Orchestrator | Đọc `deployment`, `workload_deployment`, `deployment_context`, `application_definition`, `environment_configuration`, `resource_definition`, `resource_instance`; có thể refresh fingerprint khi `PLAN_CHANGED`; atomic CAS cập nhật `deployment.status = CONFIRMED` | **Có — Contract 5** | **Có — Deployment:** self-transition `PLAN_CHANGED`/invalid override; `AWAITING_CONFIRMATION` → `CONFIRMED` chỉ khi fingerprint match và CAS thành công; CAS loser không transition |
| UC-03 | Luồng chính/A2: reconcile infrastructure qua provisioner | `reconcileInfrastructure()` | `sequence_digrams/uc_03_deploy_application.puml`: Infrastructure Reconciler; sau success, Deployment Orchestrator gọi `updateDeploymentStatus(..., INFRASTRUCTURE_READY)` | Infrastructure Reconciler | Ghi/đọc `resource_instance`; cập nhật `deployment.status = INFRASTRUCTURE_READY` qua Deployment Repository | **Có — Contract 6** | **Có — Deployment:** `CONFIRMED` → `INFRASTRUCTURE_READY`/`FAILED`; **Resource Instance:** `PLANNED` → `PROVISIONING` → `READY`/`FAILED`, hoặc reuse at `READY` |
| UC-03 | Luồng chính: thu thập Resource Output khi infrastructure ready | `collectResourceOutputs()` | `sequence_digrams/uc_03_deploy_application.puml`: Resource Output Resolver / Collector | Resource Output Resolver / Collector | Đọc `resource_instance`; không có table `resource_output` vì object transient | Không | **Có — Deployment:** action trước resolve configuration; **Resource Instance:** self-transition tại `READY` |
| UC-03 | Luồng chính/A1/A2: resolve direct value, Resource/Workload Output | `resolveEnvironmentConfiguration()` | `sequence_digrams/uc_03_deploy_application.puml`: Environment Configuration Resolver; sau success, Deployment Orchestrator gọi `updateDeploymentStatus(..., CONFIGURATION_RESOLVED)` | Environment Configuration Resolver | Đọc `environment_configuration`, `environment_variable`, `configuration_value`, `secret`, `resource_instance`; cập nhật `deployment.status = CONFIGURATION_RESOLVED` qua Deployment Repository | **Có — Contract 7** | **Có — Deployment:** `INFRASTRUCTURE_READY` → `CONFIGURATION_RESOLVED`/`FAILED` |
| UC-03 | Luồng chính: tạo resolved application specification | `generateResolvedApplicationSpecification()` | `sequence_digrams/uc_03_deploy_application.puml`: Resolved Specification Generator | Resolved Specification Generator | Đọc snapshot từ `workload_deployment`; `Resolved Specification` là transient, không có table | Không | **Có — Deployment:** phase của transition `CONFIGURATION_RESOLVED` → `SUBMITTED`; không có state riêng |
| UC-03 | Luồng chính/A2: sinh base Kubernetes manifest bằng score-k8s | `generateKubernetesManifest()` | `sequence_digrams/uc_03_deploy_application.puml`: Manifest Generator / Score Renderer | Manifest Generator / Score Renderer | — *(manifest và resolved specification transient)* | Không | **Có — Deployment:** phase/failure point trước `SUBMITTED` |
| UC-03 | Luồng chính/A2: áp dụng target-specific adaptation/patch | `adaptManifestForTarget()` | `sequence_digrams/uc_03_deploy_application.puml`: Target Manifest Adapter | Target Manifest Adapter | — *(manifest transient)* | Không | **Có — Deployment:** phase/failure point trước `SUBMITTED` |
| UC-03 | Luồng chính/A2: materialize Environment Variable | `materializeEnvironmentConfiguration()` | `sequence_digrams/uc_03_deploy_application.puml`: Environment Configuration Materializer | Environment Configuration Materializer | — *(desired state transient; không persist resolved value)* | Không | **Có — Deployment:** phase/failure point trước `SUBMITTED` |
| UC-03 | Luồng chính/A2: materialize Secret/secret reference | `materializeSecretConfiguration()` | `sequence_digrams/uc_03_deploy_application.puml`: Secret Materializer | Secret Materializer | — *(desired state transient; không persist plaintext Secret)* | Không | **Có — Deployment:** phase/failure point trước `SUBMITTED` |
| UC-03 | Luồng chính/A2: publish desired deployment state sang CD abstraction | `publishDesiredDeploymentState()` | `sequence_digrams/uc_03_deploy_application.puml`: CD Integration / CD Provider Interface; sau accepted, Deployment Orchestrator gọi `updateDeploymentStatus(..., SUBMITTED)` | CD Integration / CD Provider Interface | Cập nhật `deployment.status = SUBMITTED` qua Deployment Repository; `delivery_reference` vẫn chỉ được persist sau bởi `saveDeploymentRecord()` | **Có — Contract 8** | **Có — Deployment:** `CONFIGURATION_RESOLVED` → `SUBMITTED`/`FAILED` |
| UC-03 | Luồng chính/A2: lưu Deployment Record, image, infra reference, progress/error | `saveDeploymentRecord()` | `sequence_digrams/uc_03_deploy_application.puml`: Deployment Repository | Deployment Repository | Ghi/cập nhật `deployment`, `deployment_record`, `deployment_step`, `deployment_record_resource_instance`; actual images đọc từ `workload_deployment` | **Có — Contract 9** | **Có — Deployment:** persist `SUBMITTED` hoặc `FAILED` và transition action tương ứng |
| UC-04 | Luồng chính: mở Deployments và xem lịch sử | `listDeployments()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Deployment Query API / Controller → Deployment Query Service | Deployment Query API / Controller → Deployment Query Service | Đọc `deployment`, `deployment_record` | Không | Không — read-only, chỉ quan sát state |
| UC-04 | Luồng chính: chọn một deployment và lấy detail | `getDeploymentDetail()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Deployment Query API / Controller → Deployment Query Service → Deployment Repository | Deployment Query API / Controller → Deployment Query Service → Deployment Repository | Đọc `deployment`, `deployment_record`, `deployment_record_resource_instance` | Không | Không — read-only |
| UC-04 | Luồng chính: hiển thị deployment progress theo từng bước | `getDeploymentProgress()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Deployment Repository | Deployment Repository | Đọc `deployment_step` | Không | Không — đọc progress marker, không đổi Deployment state |
| UC-04 | Luồng chính: hiển thị infrastructure status | `getInfrastructureStatus()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Resource Instance Repository | Resource Instance Repository | Đọc `deployment_record_resource_instance`, `resource_instance` | Không | Không — đọc Resource Instance state |
| UC-04 | Luồng chính: hiển thị CD status | `getCDStatus()` | `sequence_digrams/uc_04_view_deployment_result.puml`: CD Integration / CD Status Provider | CD Integration / CD Status Provider | `deployment_record.delivery_reference` là input đã load; status hiện tại lấy từ external CD System | Không | Không — read-only external status |
| UC-04 | Luồng chính/A2: hiển thị workload health | `getWorkloadStatus()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Workload Status Provider / Kubernetes Adapter | Workload Status Provider / Kubernetes Adapter | `deployment.deployment_target` và `workload_deployment` là input đã load; health lấy từ Kubernetes Cluster | Không | Không — read-only runtime status |
| UC-04 | Luồng chính: hiển thị exposed endpoint nếu có | `getDeploymentEndpoints()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Workload Status Provider / Kubernetes Adapter | Workload Status Provider / Kubernetes Adapter | `deployment.deployment_target` và `workload_deployment` là input đã load; endpoint lấy từ Kubernetes Cluster | Không | Không — read-only runtime data |
| UC-04 | Luồng chính: hiển thị image repository/version thực tế | `getDeploymentImages()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Deployment Repository | Deployment Repository | Đọc `workload_deployment` qua `deployment`/`deployment_record` | Không | Không — read-only |
| UC-04 | Luồng ngoại lệ A1: hiển thị failed step, component và error summary | `getDeploymentFailureDetail()` | `sequence_digrams/uc_04_view_deployment_result.puml`: Deployment Query API / Controller → Deployment Query Service → Deployment Repository trong nhánh deployment failed (A1) | Deployment Query API / Controller → Deployment Query Service → Deployment Repository | Đọc `deployment_record.error_summary` và các row `deployment_step` failed: `status`, `related_component_reference`, `error_summary` | Không | Không — read-only, chỉ quan sát `FAILED` |

## B. Coverage / completeness check

### B1. Mọi Step-2 system operation có trong sequence diagram

| Use Case | Số operation Bước 2 | Có message trong sequence | Kết quả |
|---|---:|---:|---|
| UC-01 | 9 | 9 | PASS |
| UC-02 | 7 | 7 | PASS |
| UC-03 | 18 | 18 | PASS |
| UC-04 | 9 | 9 | PASS |
| **Tổng** | **43** | **43** | **PASS** |

**Kết luận B1: PASS.** Cả 43/43 Step-2 system operation đều có message trong sequence diagram tương ứng; `getDeploymentFailureDetail()` đã có đầy đủ flow UI → Controller → Service → Repository trong nhánh UC-04 A1.

### B2. Mọi class trong VOPC có operation hoặc được giải thích vai trò participant

| VOPC class | Operation evidence hoặc justification | Kết quả |
|---|---|---|
| Web UI | Presentation boundary: các operation `show...()`; nhận input/hiển thị output cho cả bốn UC | PASS |
| Application API / Controller | Nhận các system operation UC-01 từ `createApplication()` đến `saveApplicationDefinition()` | PASS |
| Environment Configuration API / Controller | Nhận `selectEnvironment()`, set/bind và `saveEnvironmentConfiguration()` | PASS |
| Deployment API / Controller | Nhận `createDeployment()` và boundary `confirmDeployment()`; `loadDeploymentForm()` là internal UI query | PASS |
| Deployment Query API / Controller | Nhận `listDeployments()`, `getDeploymentDetail()`, `getDeploymentFailureDetail()` | PASS |
| Application Service | Xử lý các system operation UC-01 | PASS |
| Environment Configuration Service | Xử lý select/set/bind/save của UC-02 | PASS |
| Deployment Orchestrator | Nhận/tự gọi create, validate, apply override và confirm trong UC-03 | PASS |
| Deployment Query Service | Điều phối `listDeployments()`, `getDeploymentDetail()`, `getDeploymentFailureDetail()` | PASS |
| Application Definition Validator | `validateApplicationDefinition()` | PASS |
| Application Specification Generator | `generateApplicationSpecification()` | PASS |
| Resource Output Catalog / Resource Definition Query | Internal query participant: `listValidOutputs()`, `listValidSensitiveOutputs()` | PASS |
| Workload Output Catalog | Internal query participant: `listValidOutputs()` | PASS |
| Environment Configuration Validator | `validateEnvironmentConfiguration()` | PASS |
| Deployment Graph Builder | `buildDeploymentGraph()` | PASS |
| Resource Definition Resolver | `resolveResourceDefinitions()` | PASS |
| Infrastructure Planner | `planInfrastructureChanges()`, `loadInfrastructureOverrides()`, `applyInfrastructureOverrides()` | PASS |
| Infrastructure Reconciler | `reconcileInfrastructure()` | PASS |
| Resource Output Resolver / Collector | `collectResourceOutputs()` | PASS |
| Environment Configuration Resolver | `resolveEnvironmentConfiguration()` | PASS |
| Resolved Specification Generator | `generateResolvedApplicationSpecification()` | PASS |
| Manifest Generator / Score Renderer | `generateKubernetesManifest()` | PASS |
| Target Manifest Adapter | `adaptManifestForTarget()` | PASS |
| Environment Configuration Materializer | `materializeEnvironmentConfiguration()` | PASS |
| Secret Materializer | `materializeSecretConfiguration()` | PASS |
| Deployment Result Aggregator | Internal data/view-model assembler: `aggregate()` | PASS |
| Secret Store / Secret Management Adapter | External persistence abstraction for secret value: `storeSecret()`; không phải UC system operation | PASS |
| Provisioner Adapter / Provisioner Interface | Integration abstraction: `reconcile()` delegates provider execution | PASS |
| CD Integration / CD Provider Interface | `publishDesiredDeploymentState()` | PASS |
| CD Integration / CD Status Provider | `getCDStatus()` | PASS |
| Workload Status Provider / Kubernetes Adapter | `getWorkloadStatus()`, `getDeploymentEndpoints()` | PASS |
| Concrete CD Provider | Integration implementation: internal `publish()` và `getStatus()` | PASS |
| Terraform/OpenTofu Runner | External participant: `applyInfrastructureModules()` | PASS |
| score-k8s | External renderer: `render()` | PASS |
| CD System | External participant: `synchronize()`, `queryDeploymentStatus()` | PASS |
| Kubernetes Cluster | External runtime participant: deploy/query operations | PASS |
| Application Repository | Persistence holder: internal `findById()`, `save()` | PASS |
| Specification Repository / Config Repo Service | Persistence holder: internal `save()` | PASS |
| Application Query / Application Repository | `loadConfigurationRequirements()` | PASS |
| Environment Configuration Repository | Persistence holder: internal `save()`, `findByApplicationAndEnvironment()`, `findById()` | PASS |
| Resource Instance Repository | `getInfrastructureStatus()` và internal find/save/load output operations | PASS |
| Deployment Repository | `persistDeployment()`, `findByIdWithPersistedInputs()`, `compareAndSetPlanFingerprint()`, atomic `updateDeploymentStatus()`, `saveDeploymentRecord()`, `getDeploymentDetail()`, `getDeploymentFailureDetail()`, `getDeploymentProgress()`, `getDeploymentImages()` và internal `findByApplication()` | PASS |

**Kết luận B2: PASS.** Toàn bộ 42 class hợp nhất trong VOPC có operation cụ thể hoặc có justification rõ ràng là presentation/persistence holder, integration implementation hay external participant; `getDeploymentFailureDetail()` hiện có owner ở Controller, Service và Repository.

### B3. Mọi PERSISTENT domain object có table và có operation ghi

| Persistent domain object(s) | Table/mapping | Operation ghi | Kết quả |
|---|---|---|---|
| Application Definition; Workload; Resource Requirement; Environment Variable Definition; Secret Definition; Dependency | `application_definition`, `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` | `saveApplicationDefinition()` | PASS |
| Application Specification | `application_specification` | `generateApplicationSpecification()` | PASS |
| Environment Configuration; Environment Variable; Secret; Configuration Value; Direct Configuration Value; Resource Output Reference; Workload Output Reference | `environment_configuration`, `environment_variable`, `secret`, `configuration_value` (các value/reference subtype embedded đúng schema) | `saveEnvironmentConfiguration()` | PASS |
| Resource Definition | `resource_definition` | **Không có Step-2 operation ghi catalog**; chỉ có query/resolve | **GAP** |
| Deployment; Workload Deployment; Deployment Context | `deployment` (gồm `plan_fingerprint`, `plan_fingerprint_algo`), `workload_deployment`, `deployment_context` | `createDeployment()` qua `persistDeployment(..., AWAITING_CONFIRMATION)`; Contract 5 có fingerprint refresh/CAS; Contracts 5–8 qua `updateDeploymentStatus()` | PASS |
| Resource Instance | `resource_instance` | `reconcileInfrastructure()` | PASS |
| Deployment Record; Deployment Step | `deployment_record`, `deployment_step` | `saveDeploymentRecord()` | PASS |

**Kết luận B3: GAP được chấp nhận theo scope boundary.** Tất cả persistent object đều có mapping table. Deployment aggregate đã có repository write path tường minh trong sequence/VOPC. `Resource Definition` vẫn là platform-managed reference data, không có operation create/update/import trong bốn Developer use case; đây là ranh giới phạm vi được chấp nhận, không được giải quyết theo chủ đích.

### B4. Mọi table Step 3 được ít nhất một operation đọc hoặc ghi

| Table(s) | Operation đọc/ghi đại diện | Kết quả |
|---|---|---|
| `application_definition` | W: `saveApplicationDefinition()`; R: `updateApplication()`, `createDeployment()` | PASS |
| `workload` | W: `saveApplicationDefinition()`; R: `loadConfigurationRequirements()`, `createDeployment()` | PASS |
| `resource_requirement` | W: `saveApplicationDefinition()`; R: `buildDeploymentGraph()`, `resolveResourceDefinitions()` | PASS |
| `environment_variable_definition`, `secret_definition`, `dependency` | W: `saveApplicationDefinition()`; R: `loadConfigurationRequirements()`/`buildDeploymentGraph()` | PASS |
| `application_specification` | W: `generateApplicationSpecification()` | PASS |
| `environment_configuration`, `environment_variable`, `configuration_value`, `secret` | W: `saveEnvironmentConfiguration()`; R: `createDeployment()`/`resolveEnvironmentConfiguration()` | PASS |
| `resource_definition` | R: output catalog queries và `resolveResourceDefinitions()`; không có writer trong bốn UC | PASS cho tiêu chí read-or-write |
| `resource_instance` | W: `reconcileInfrastructure()`; R: `planInfrastructureChanges()`, `collectResourceOutputs()`, `getInfrastructureStatus()` | PASS |
| `deployment`, `workload_deployment`, `deployment_context` | W: `createDeployment()` persist fingerprint + algorithm, `confirmDeployment()` refresh fingerprint khi plan đổi và CAS status, cùng các `updateDeploymentStatus()`; R: confirm rebuild và các query UC-04 | PASS |
| `deployment_record` | W: `saveDeploymentRecord()`; R: `listDeployments()`, `getDeploymentDetail()`, `getDeploymentFailureDetail()` | PASS |
| `deployment_step` | W: `saveDeploymentRecord()`; R: `getDeploymentProgress()`, `getDeploymentFailureDetail()` | PASS |
| `deployment_record_resource_instance` | W: `saveDeploymentRecord()`; R: detail/infrastructure status path của UC-04 | PASS |

**Kết luận B4: PASS.** Cả 19 table trong ERD đều có ít nhất một read hoặc write path. `resource_definition` chỉ có read path, được ghi nhận riêng là GAP ở tiêu chí B3.

### B5. Mọi Operation Contract tương ứng một Step-2 operation có thật

| Contract | Step-2 operation | Kết quả |
|---:|---|---|
| 1 | `saveApplicationDefinition()` | PASS |
| 2 | `generateApplicationSpecification()` | PASS |
| 3 | `saveEnvironmentConfiguration()` | PASS |
| 4 | `createDeployment()` | PASS |
| 5 | `confirmDeployment()` | PASS — rebuild/fingerprint guard và atomic status CAS được đặc tả xuyên suốt |
| 6 | `reconcileInfrastructure()` | PASS |
| 7 | `resolveEnvironmentConfiguration()` | PASS |
| 8 | `publishDesiredDeploymentState()` | PASS |
| 9 | `saveDeploymentRecord()` | PASS |

**Kết luận B5: PASS.** Có 9/9 contract, tất cả đều cross-reference đúng một operation trong Step 2; không có contract “mồ côi”.

### B6. Hai state machine được drive bởi operation có trong matrix

| State machine | Step-2 operation drive lifecycle | Kết quả |
|---|---|---|
| Deployment | `createDeployment()`, `validateDeploymentInput()`, `buildDeploymentGraph()`, `resolveResourceDefinitions()`, `planInfrastructureChanges()`, `confirmDeployment()`, `applyInfrastructureOverrides()`, `reconcileInfrastructure()`, `collectResourceOutputs()`, `resolveEnvironmentConfiguration()`, `generateResolvedApplicationSpecification()`, `generateKubernetesManifest()`, `adaptManifestForTarget()`, `materializeEnvironmentConfiguration()`, `materializeSecretConfiguration()`, `publishDesiredDeploymentState()`, `saveDeploymentRecord()` | PASS |
| Resource Instance | `planInfrastructureChanges()`, `reconcileInfrastructure()`, `collectResourceOutputs()` | PASS |

**Kết luận B6: PASS.** Mọi system operation được nêu trên transition/action của hai state machine đều xuất hiện trong main matrix. `provisioner apply success/failure` là internal event nằm trong `reconcileInfrastructure()`, không phải một Step-2 system operation riêng.

## C. Gaps / Notes

### Gaps

1. **RESOLVED — UC-04 orphan operation:** `getDeploymentFailureDetail()` đã có nhánh A1 UI → Controller → Service → Repository, đọc failure fields từ `deployment_step` và `deployment_record`, đồng thời được khai báo trong VOPC/design class diagram.
2. **ACCEPTED SCOPE BOUNDARY — NOT RESOLVED BY DESIGN:** `Resource Definition` là catalog `PERSISTENT (platform-managed)` nhưng không có Step-2 operation create/update/import trong bốn Developer use case. Writer thuộc platform administration ngoài phạm vi hiện tại; không bổ sung writer giả vào các UC này.
3. **RESOLVED — UC-03 Deployment persistence:** Sequence/VOPC đã có `persistDeployment(..., AWAITING_CONFIRMATION)` cho aggregate ban đầu và `updateDeploymentStatus()` tại các transition `CONFIRMED`, `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `SUBMITTED`; `saveDeploymentRecord()` tiếp tục persist delivery/progress/failure như trước.
4. **RESOLVED — Gap 4, transient Infrastructure Plan across requests:** `createDeployment()` canonicalize/hash plan và persist chỉ `plan_fingerprint` + algorithm; `confirmDeployment()` rebuild từ persisted inputs + current catalog/Resource Instance state, trả `PLAN_CHANGED` để review lại khi mismatch, và chỉ apply overrides + atomic status CAS + reconcile khi match. Scope: fingerprint bao phủ allowed-overrides **definition**, không bao phủ Developer-selected values (validate sau match); đây chỉ là application-level defense-in-depth, còn shared-resource contention/drift cần Resource-Instance-level version/optimistic lock hoặc per-resource reconcile lock, cộng provisioner idempotency (ví dụ Terraform refresh + plan), đều ngoài phạm vi fingerprint.

### Notes

- Boundary và orchestration message của confirm hiện thống nhất là `confirmDeployment(deploymentId, overrides)`; `applyInfrastructureOverrides()` chỉ chạy trong nhánh fingerprint match.
- Việc sequence UC-03 chỉ mô tả **Main Flow** giải thích vì sao các A2 failure call tới `saveDeploymentRecord()` không được vẽ thành `alt`; failure semantics vẫn được contract và state machine mô tả. Đây là note về mức chi tiết, không tạo thêm orphan Step-2 operation.
- Các object `Deployment Graph`, `Resource Resolution`, Infrastructure Plan, `Resource Output`, `Resolved Configuration`, `Resolved Specification` là `TRANSIENT`, nên việc không có table tương ứng là chủ đích và không phải GAP; riêng Infrastructure Plan chỉ có fingerprint + algorithm version được persist trên `deployment`.

## Overall acceptance

**Kết quả tổng thể: PASS với accepted scope boundary.** Design hiện trace được 43/43 Step-2 operations, 42/42 VOPC classes, 19/19 tables theo tiêu chí read-or-write, 9/9 operation contracts và 2/2 state machines. Gap 1, Gap 3 và Gap 4 đã đóng; Gap 2 vẫn cố ý không giải quyết vì writer của `Resource Definition` thuộc platform administration ngoài bốn Developer use case.
