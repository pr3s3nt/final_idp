---
id: DESIGN-CLASS-INDEX
artifact: design-class-and-vopc-index
status: current
last_reviewed: 2026-09-18
---

# VOPC / Design Class Diagram

Tài liệu này mô tả **View Of Participating Classes (VOPC)** cho sáu use case của Internal Developer Platform. [`design-class-diagram.puml`](design-class-diagram.puml) là góc nhìn hợp nhất toàn hệ thống; VOPC của từng use case nằm cạnh specification và realization trong `docs/use-cases/UC-*/vopc.puml`.

## Cách đọc diagram

- Mỗi `package` tương ứng một trong bảy layer đã thống nhất: Boundary/UI, Application services, Domain components, Integration abstractions, Integration implementations, External systems và Persistence.
- Attribute có visibility `-`; operation có visibility `+`.
- Operation được đặt tại class nhận message trong sequence diagram. Ví dụ, `Deployment Graph Builder` nhận `buildDeploymentGraph()` nên sở hữu operation này; `Deployment Repository` nhận `saveDeploymentRecord()` và các query operation nên sở hữu các operation tương ứng.
- `Deployment Orchestrator` chỉ giữ các operation mà nó trực tiếp nhận trong request của Developer: tải deployment form, tạo/xác nhận deployment và tính plan fingerprint.
- `Deployment Worker` là tiến trình chạy nền chủ động lấy job từ `Deployment Repository`, nên không nhận message nào và không có operation. Các lời gọi nó gửi đi thuộc về class nhận; ví dụ `reconcileInfrastructure(finalPlan, planItems)` thuộc về `Infrastructure Reconciler`.
- Đường liền biểu diễn association; multiplicity `"1"` được ghi tại các quan hệ một controller/service sử dụng một collaborator tương ứng trong phạm vi xử lý.
- Mũi tên nét đứt `..>` biểu diễn dependency/call direction.
- Mũi tên `Interface <|.. Implementation` biểu diễn realization. `Concrete CD Provider` realize cả publish interface của UC-03 và status provider của UC-04.
- `Terraform/OpenTofu Runner` là external participant được `Provisioner Adapter / Provisioner Interface` gọi qua dependency: operation `reconcile(planItems)` của adapter chuyển thành lời gọi `applyInfrastructureModules(planItems)` trên runner, và `destroy(planItems)` chuyển thành `destroyInfrastructureModules(planItems)` khi hủy resource không còn trong phiên bản. Quan hệ này không dùng realization vì hai operation không cùng signature.
- Tài liệu nguồn chưa chọn concrete implementation cho `Secret Store / Secret Management Adapter`; đồng thời `Workload Status Provider / Kubernetes Adapter` đã là một participant gộp. Vì vậy diagram không thêm class giả chỉ để tạo realization cho hai abstraction này.

## Quy ước stereotype

| Stereotype | Ý nghĩa |
|---|---|
| `<<boundary>>` | Điểm tương tác với actor hoặc external system boundary, gồm Web UI và các hệ thống ngoài. |
| `<<control>>` | API/Controller, application service, domain component điều phối/xử lý logic, hoặc integration implementation. |
| `<<entity>>` | Repository quản lý dữ liệu bền vững và domain record liên quan. |
| `<<interface>>` | Integration abstraction che giấu provider/backend cụ thể. |

## Danh mục participating class

| Class | Layer | Responsibility |
|---|---|---|
| Authentication Web UI | Boundary/UI | Feature React tải login context, thu username/password, trình bày lỗi/session state và cung cấp logout trong authenticated shell; không lưu credential hoặc session token bằng browser storage. |
| Authentication API / Controller | Boundary/UI | Nhận sign-in/sign-out, validate return path, cookie và CSRF, rồi ánh xạ kết quả thành redirect/HTTP response. |
| Authentication Middleware | Boundary/UI | Bảo vệ page/API, xác thực session và gắn `Principal` trung lập vào request context. |
| Local User CLI | Boundary/UI | Tạo/reset/enable/disable local user qua terminal tin cậy với hidden password prompt. |
| Web UI | Boundary/UI | Nhận thao tác của Developer và hiển thị application, configuration, deployment plan cùng deployment result. |
| Application API / Controller | Boundary/UI | Nhận request tạo/cập nhật Application Definition và chuyển sang Application Service. |
| Environment Configuration API / Controller | Boundary/UI | Nhận request chọn environment, gán value/reference và lưu Environment Configuration. |
| Deployment API / Controller | Boundary/UI | Nhận deployment input (gồm phiên bản Application Definition, phiên bản catalog và nơi triển khai), tải deployment form và chuyển xác nhận deploy sang orchestrator. |
| Deployment Query API / Controller | Boundary/UI | Cung cấp read-only API cho deployment history và deployment detail. |
| Application Service | Application services | Điều phối UC-01 để tạo/cập nhật, validate, lưu Application Definition và sinh specification. |
| Environment Configuration Service | Application services | Điều phối UC-02 để tải requirement, gán value/reference, bảo vệ Secret, validate và lưu configuration. |
| Deployment Orchestrator | Application services | Điều phối phần UC-03 và UC-05 trong request của Developer (UC-05 lập plan gỡ bỏ theo thứ tự ngược, không nhận phiên bản hay image): tải form (gồm các phiên bản catalog), tạo deployment theo phiên bản Application Definition và phiên bản catalog, dựng graph, chia tầng, lập plan; khi xác nhận thì đổi trạng thái và tạo job trong một transaction rồi trả lời ngay. |
| Deployment Worker | Application services | Tiến trình chạy nền lấy job và điều phối phần thực thi của UC-03 và UC-05: triển khai theo tầng, chờ workload healthy, thu output, lan truyền thay đổi output, gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản, lưu Deployment Record. |
| Deployment Query Service | Application services | Điều phối query path UC-04 và thu thập dữ liệu từ repository cùng status provider. |
| Authentication Service | Application services | Điều phối local sign-in, account provisioning/reset/status; không trả lỗi giúp phân biệt account không tồn tại, disabled hay password sai. |
| Password Hasher | Domain components | Tạo/verify encoded Argon2id hash và phát hiện hash cần nâng tham số. |
| Session Manager | Domain components | Sinh/hash token, tạo/kiểm tra/thu hồi server-side session và tạo `Principal`. |
| Login Rate Limiter | Domain components | Áp dụng bucket theo username chuẩn hóa và nguồn request mà không giữ credential hoặc raw identifier. |
| Principal | Domain components | Identity transient trong request context, tách use case nghiệp vụ khỏi local credential/session provider. |
| Application Definition Validator | Domain components | Kiểm tra workload, resource, dependency, port, image repository và configuration requirement. |
| Application Specification Generator | Domain components | Chuyển Application Definition đã lưu thành application specification như `score.yaml`. |
| Resource Output Catalog / Resource Definition Query | Domain components | Liệt kê output thường và sensitive output hợp lệ của logical resource. |
| Workload Output Catalog | Domain components | Liệt kê output hợp lệ mà workload expose, ví dụ endpoint. |
| Environment Configuration Validator | Domain components | Kiểm tra direct value và các Resource/Workload Output reference của một environment theo phiên bản Application Definition mới nhất, gồm việc output thuộc thành phần mà workload depends on. |
| Deployment Graph Builder | Domain components | Dựng dependency/resource graph từ phiên bản Application Definition, configuration, images, phiên bản catalog và deployment context; tự thêm cụm Kubernetes và những gì Resource Definition `requires` (gọi Resource Definition Resolver trong lúc dựng); phát hiện vòng. |
| Deployment Wave Planner | Domain components | Xác định phạm vi deployment (kể cả cụm/network mà resource cần), chia tầng theo thứ tự phụ thuộc, liệt kê thành phần có thể bị làm lại, và lan truyền khi output thay đổi bằng cách reconcile lại resource phụ thuộc và thêm workload phụ thuộc vào các tầng sau. |
| Resource Definition Resolver | Domain components | Chọn Resource Definition trong phiên bản catalog được chọn cho logical resource, cụm Kubernetes và network theo deployment context; phân biệt loại `MANAGED` và `EXISTING` (resource dùng chung, cụm nội bộ). |
| Infrastructure Planner | Domain components | Tìm Resource Instance theo chủ sở hữu, so sánh desired/current state, lập plan (create/update/reuse/liên kết; gỡ/hủy/gỡ liên kết) và quản lý permitted overrides. |
| Infrastructure Reconciler | Domain components | Điều phối thực thi plan theo tầng: tạo/sửa resource `MANAGED` (kể cả VPC, cụm trên cloud) với input từ output của những gì resource cần, liên kết resource `EXISTING` (resource dùng chung, cụm nội bộ), hủy hoặc gỡ liên kết resource không còn trong phiên bản, lưu resource state/reference. |
| Resource Output Resolver / Collector | Domain components | Thu thập Resource Output từ infrastructure instance đã sẵn sàng. |
| Workload Output Collector | Domain components | Thu thập Workload Output từ workload đã healthy hoặc đang chạy; cơ chế đọc cụ thể chưa chốt. |
| Environment Configuration Resolver | Domain components | Resolve direct value, Resource Output và Workload Output thành configuration thực tế cho các workload trong một tầng. |
| Resolved Specification Generator | Domain components | Tạo resolved application specification chứa image, dependency và configuration đã resolve. |
| Manifest Generator / Score Renderer | Domain components | Gọi `score-k8s` để sinh base Kubernetes manifest. |
| Target Manifest Adapter | Domain components | Áp dụng patch/adaptation riêng cho deployment target lên base manifest. |
| Environment Configuration Materializer | Domain components | Materialize Environment Variable đã resolve thành Kubernetes configuration. |
| Secret Materializer | Domain components | Materialize Secret value/reference mà không làm lộ plaintext. |
| Deployment Result Aggregator | Domain components | Hợp nhất record, progress, image, infrastructure, CD và workload status thành view model. |
| Secret Store / Secret Management Adapter | Integration abstractions | Lưu Secret an toàn và trả về secret reference để persistence không giữ plaintext. |
| Provisioner Adapter / Provisioner Interface | Integration abstractions | Cung cấp abstraction cho việc reconcile và hủy infrastructure theo các plan item. |
| CD Integration / CD Provider Interface | Integration abstractions | Cung cấp abstraction để publish desired deployment state tới CD provider. |
| CD Integration / CD Status Provider | Integration abstractions | Cung cấp abstraction read-only để lấy deployment/sync status từ CD provider. |
| Delivery Repository Provider | Integration abstractions | Bảo đảm nơi chứa desired state của một application tồn tại: tạo theo quy ước đặt tên của platform, sinh và gắn cặp khóa riêng của application. |
| Workload Status Provider / Kubernetes Adapter | Integration abstractions | UC-03: chờ workload healthy và đọc dữ liệu runtime phục vụ Workload Output Collector. UC-04: truy vấn workload health và exposed endpoint từ Kubernetes Cluster. |
| Concrete CD Provider | Integration implementations | Hiện thực publish/status operation cho một CD system cụ thể như Fleet, Argo CD hoặc Flux. |
| Concrete Git Hosting Provider | Integration implementations | Hiện thực Delivery Repository Provider cho một hệ thống lưu trữ Git cụ thể (ví dụ GitHub); đọc thông tin đăng nhập từ Secret Store. |
| Terraform/OpenTofu Runner | External systems | Thực thi infrastructure module (apply hoặc destroy) và trả resource state cùng raw outputs. |
| score-k8s | External systems | Render resolved application specification thành base Kubernetes manifest. |
| CD System | External systems | Nhận desired state, đồng bộ workload xuống cluster và cung cấp CD status. |
| Kubernetes Cluster | External systems | Chạy workload/configuration; cung cấp pod health, dữ liệu runtime của workload và endpoint. Trên cloud, cụm do IDP dựng qua Provisioner; cụm nội bộ có sẵn. Thông tin kết nối lấy từ output của `k8s-cluster`. |
| Application Repository | Persistence | Lưu/đọc Application Definition theo phiên bản bất biến cho UC-01 và UC-03; thành phần giữ ID cố định qua phiên bản; tạo Platform Requirement (cụm Kubernetes, network) của application khi cần lần đầu. |
| Resource Definition Catalog | Persistence | Đọc các phiên bản catalog bất biến và Resource Definition của phiên bản được chọn; catalog do platform quản lý, IDP chỉ đọc. |
| Delivery Repository Registry | Persistence | Lưu/đọc nơi chứa desired state của từng application: URL, nhánh và secret reference của cặp khóa. |
| Specification Repository / Config Repo Service | Persistence | Lưu hoặc version hóa application specification đã sinh. |
| Application Query / Application Repository | Persistence | Đọc workload, variable, Secret và dependency requirement của phiên bản Application Definition mới nhất phục vụ UC-02. |
| Environment Configuration Repository | Persistence | Lưu value/reference theo environment (STAGING, PRODUCTION) và đọc lại khi deployment. |
| Resource Instance Repository | Persistence | Lưu/đọc Resource Instance theo chủ sở hữu (application + environment + resource requirement hoặc cụm/network + target): trạng thái, reference, liên kết tới thứ có sẵn, dấu vân tay output, dấu vân tay đầu vào lần apply gần nhất và infrastructure status. |
| Workload Instance Repository | Persistence | Lưu/đọc trạng thái hiện hành của từng workload theo environment và target: workload deployment đang chạy, trạng thái, dấu vân tay output. |
| Deployment Repository | Persistence | Lưu deployment theo phiên bản Application Definition và phiên bản catalog, job triển khai (kèm override) và Deployment Record; cung cấp history, detail, progress cùng actual image version. |
| User Account Repository | Persistence | Lưu/đọc Local User Account và Local Credential; tạo/reset/status atomically với session revocation khi cần. |
| Auth Session Repository | Persistence | Lưu token/CSRF hash và lifecycle session; lookup, touch và revoke một/toàn bộ session. |
| Login Attempt Repository | Persistence | Lưu bucket rate-limit có thời hạn theo account/source key hash. |

## Phạm vi từng file

| File | Nội dung |
|---|---|
| [`design-class-diagram.puml`](design-class-diagram.puml) | Consolidated design class diagram của toàn bộ UC-01 đến UC-06. |
| [`docs/use-cases/UC-01/vopc.puml`](../use-cases/UC-01/vopc.puml) | Participating classes cho Create / Configure Application. |
| [`docs/use-cases/UC-02/vopc.puml`](../use-cases/UC-02/vopc.puml) | Participating classes cho Configure Application Environment. |
| [`docs/use-cases/UC-03/vopc.puml`](../use-cases/UC-03/vopc.puml) | Participating classes cho Deploy Application. |
| [`docs/use-cases/UC-04/vopc.puml`](../use-cases/UC-04/vopc.puml) | Participating classes cho View Deployment Result. |
| [`docs/use-cases/UC-05/vopc.puml`](../use-cases/UC-05/vopc.puml) | Participating classes cho Remove Application from Environment. |
| [`docs/use-cases/UC-06/vopc.puml`](../use-cases/UC-06/vopc.puml) | Participating classes cho Đăng nhập bằng tài khoản nội bộ. |
