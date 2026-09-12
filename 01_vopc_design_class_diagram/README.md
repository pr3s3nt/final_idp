# Step 1: VOPC / Design Class Diagram

Thư mục này mô tả **View Of Participating Classes (VOPC)** cho bốn use case của Internal Developer Platform. `design_class_diagram.puml` là góc nhìn hợp nhất toàn hệ thống; bốn file `vopc_uc*.puml` là lát cắt theo từng use case và chỉ chứa các participating class được liệt kê cho use case tương ứng tại Bước 3.

## Cách đọc diagram

- Mỗi `package` tương ứng một trong bảy layer đã thống nhất: Boundary/UI, Application services, Domain components, Integration abstractions, Integration implementations, External systems và Persistence.
- Attribute có visibility `-`; operation có visibility `+`.
- Operation được đặt tại class nhận message trong sequence diagram. Ví dụ, `Deployment Graph Builder` nhận `buildDeploymentGraph()` nên sở hữu operation này; `Deployment Repository` nhận `saveDeploymentRecord()` và các query operation nên sở hữu các operation tương ứng.
- `Deployment Orchestrator` chỉ dựng/rebuild typed plan và atomically enqueue; HTTP confirm trả `202 Accepted`. `Deployment Worker` sở hữu chuỗi dài reconcile → resolve → manifest → publish → record.
- Đường liền biểu diễn association; multiplicity `"1"` được ghi tại các quan hệ một controller/service sử dụng một collaborator tương ứng trong phạm vi xử lý.
- Mũi tên nét đứt `..>` biểu diễn dependency/call direction.
- Mũi tên `Interface <|.. Implementation` biểu diễn realization. `Concrete CD Provider` realize cả publish interface của UC-03 và status provider của UC-04.
- `Terraform/OpenTofu Runner` là external participant được `Provisioner Adapter / Provisioner Interface` gọi qua dependency: operation `reconcile(finalPlan)` của adapter chuyển thành lời gọi `applyInfrastructureModules(finalPlan)` trên runner. Quan hệ này không dùng realization vì hai operation không cùng signature.
- Tài liệu nguồn chưa chọn concrete implementation cho `Secret Store / Secret Management Adapter`; đồng thời `Workload Status Provider / Kubernetes Adapter` đã là một participant gộp. Vì vậy diagram không thêm class giả chỉ để tạo realization cho hai abstraction này.

## Quy ước stereotype

| Stereotype | Ý nghĩa |
|---|---|
| `<<boundary>>` | Điểm tương tác với actor hoặc external system boundary, gồm Web UI và các hệ thống ngoài. |
| `<<control>>` | API/Controller, application service, domain component điều phối/xử lý logic, hoặc integration implementation. |
| `<<entity>>` | Repository quản lý dữ liệu bền vững và domain record liên quan. |
| `<<interface>>` | Integration abstraction che giấu provider/backend cụ thể. |
| `<<transient>>` | Typed execution object không có table; canonical input của fingerprint. |

## Danh mục participating class

| Class | Layer | Responsibility |
|---|---|---|
| Web UI | Boundary/UI | Nhận thao tác của Developer và hiển thị application, configuration, deployment plan cùng deployment result. |
| Application API / Controller | Boundary/UI | Nhận request tạo/cập nhật Application Definition và chuyển sang Application Service. |
| Environment Configuration API / Controller | Boundary/UI | Nhận request chọn environment, gán value/reference và lưu Environment Configuration. |
| Deployment API / Controller | Boundary/UI | Nhận deployment input, tải deployment form và chuyển xác nhận deploy sang orchestrator. |
| Deployment Query API / Controller | Boundary/UI | Cung cấp read-only API cho deployment history và deployment detail. |
| Application Service | Application services | Stateless; mỗi draft edit nhận/trả full client-owned draft, rồi validate/save aggregate. |
| Environment Configuration Service | Application services | Stateless; nhận/trả full client-owned draft, stage/promote/revoke opaque Secret reference. |
| Deployment Orchestrator | Application services | Dựng/rebuild typed plan, fingerprint, validate override và transactionally enqueue. |
| Deployment Worker | Application services | Claim/retry durable job và chạy reconcile → output resolution → manifest → publish → record. |
| Deployment Query Service | Application services | Điều phối query path UC-04 và thu thập dữ liệu từ repository cùng status provider. |
| Application Definition Validator | Domain components | Kiểm tra workload, resource, dependency, port, image repository và configuration requirement. |
| Application Specification Generator | Domain components | Chuyển Application Definition đã lưu thành application specification như `score.yaml`. |
| Resource Output Catalog / Resource Definition Query | Domain components | Liệt kê output thường và sensitive output hợp lệ của logical resource. |
| Workload Output Catalog | Domain components | Liệt kê output `PLAN_TIME` hợp lệ mà workload expose, ví dụ Service endpoint. |
| Environment Configuration Validator | Domain components | Kiểm tra direct value và các Resource/Workload Output reference của một environment. |
| Deployment Graph Builder | Domain components | Dựng dependency/resource graph từ definition, configuration, images và deployment context. |
| Resource Definition Resolver | Domain components | Chọn Resource Definition phù hợp với logical resource và deployment context. |
| Infrastructure Planner | Domain components | So sánh desired/current state, lập create-update-reuse plan và quản lý permitted overrides. |
| Infrastructure Plan / Item / Override Definition | Domain components | Typed transient canonical schema; payload không persist. |
| Infrastructure Reconciler | Domain components | Điều phối thực thi infrastructure plan và lưu resource state/reference. |
| Resource Output Resolver / Collector | Domain components | Thu thập Resource Output từ infrastructure instance đã sẵn sàng. |
| Workload Output Resolver | Domain components | Tính plan-time output như Kubernetes Service DNS từ graph/metadata/context; cấm vòng tròn runtime. |
| Environment Configuration Resolver | Domain components | Resolve direct value, Resource Output và Workload Output thành configuration thực tế. |
| Resolved Specification Generator | Domain components | Tạo resolved application specification chứa image, dependency và configuration đã resolve. |
| Manifest Generator / Score Renderer | Domain components | Gọi `score-k8s` để sinh base Kubernetes manifest. |
| Target Manifest Adapter | Domain components | Áp dụng patch/adaptation riêng cho deployment target lên base manifest. |
| Environment Configuration Materializer | Domain components | Materialize Environment Variable đã resolve thành Kubernetes configuration. |
| Secret Materializer | Domain components | Materialize Secret value/reference mà không làm lộ plaintext. |
| Deployment Result Aggregator | Domain components | Hợp nhất record, progress, image, infrastructure, CD và workload status thành view model. |
| Secret Store / Secret Management Adapter | Integration abstractions | Stage opaque idempotent reference với TTL; promote hoặc compensating revoke. |
| Provisioner Adapter / Provisioner Interface | Integration abstractions | Cung cấp abstraction cho việc reconcile infrastructure theo final plan. |
| CD Integration / CD Provider Interface | Integration abstractions | Cung cấp abstraction để publish desired deployment state tới CD provider. |
| CD Integration / CD Status Provider | Integration abstractions | Cung cấp abstraction read-only để lấy deployment/sync status từ CD provider. |
| Workload Status Provider / Kubernetes Adapter | Integration abstractions | Truy vấn workload health và exposed endpoint từ Kubernetes Cluster. |
| Concrete CD Provider | Integration implementations | Hiện thực publish/status operation cho một CD system cụ thể như Argo CD hoặc Flux. |
| Terraform/OpenTofu Runner | External systems | Thực thi infrastructure module và trả resource state cùng raw outputs. |
| score-k8s | External systems | Render resolved application specification thành base Kubernetes manifest. |
| CD System | External systems | Nhận desired state, đồng bộ workload xuống cluster và cung cấp CD status. |
| Kubernetes Cluster | External systems | Chạy workload/configuration và cung cấp runtime health cùng endpoint. |
| Application Repository | Persistence | Lưu/đọc Application Definition cho UC-01 và UC-03. |
| Specification Repository / Config Repo Service | Persistence | Lưu hoặc version hóa application specification đã sinh. |
| Application Query / Application Repository | Persistence | Đọc workload, variable, Secret và dependency requirement phục vụ UC-02. |
| Environment Configuration Repository | Persistence | Lưu value/reference theo environment và đọc lại khi deployment. |
| Resource Instance Repository | Persistence | Reuse lookup luôn bắt đầu từ exact active binding theo application + environment + logical requirement + target; không fallback sang owner columns. Shared consumer binding phải được platform pre-authorize. |
| Deployment Repository | Persistence | Lifecycle, ba internal steps, record và atomic confirm+enqueue. |
| Deployment Execution Job / Outbox | Persistence | Durable tracking, selected override values, lease, retry và idempotency. |

Web UI là owner duy nhất của `ApplicationDefinitionDraft` và `ConfigurationDefinitionDraft`; service không có attribute draft. Với secret, UI thay plaintext bằng opaque staged reference ngay sau upload và không nhận plaintext trở lại.

## Phạm vi từng file

| File | Nội dung |
|---|---|
| `design_class_diagram.puml` | Consolidated design class diagram của toàn bộ UC-01 đến UC-04. |
| `vopc_uc01.puml` | Participating classes cho Create / Configure Application. |
| `vopc_uc02.puml` | Participating classes cho Configure Application Environment. |
| `vopc_uc03.puml` | Participating classes cho Deploy Application. |
| `vopc_uc04.puml` | Participating classes cho View Deployment Result. |
