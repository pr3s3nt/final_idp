---
id: UC-03-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-03
last_reviewed: 2026-09-17
---

# UC-03 — Deploy Application: Use Case Realization

## Responsibility

Chịu trách nhiệm biến một phiên bản Application Definition + phiên bản catalog + Environment Configuration của một environment (staging hoặc production) + nơi triển khai (cloud hoặc cụm Kubernetes nội bộ) và deployment context + image version của các workload được chọn thành một deployment thực tế. Trách nhiệm chia làm hai phần:

- **Nhận yêu cầu deploy:** kiểm tra configuration khớp phiên bản, dựng dependency/resource graph (kể cả cụm Kubernetes và network của nơi triển khai), xác định phạm vi và chia tầng theo thứ tự phụ thuộc, tìm resource để dùng lại theo đúng chủ sở hữu, lập plan (gồm cả thành phần sẽ bị gỡ, hủy hoặc gỡ liên kết) cho Developer xem; khi Developer xác nhận thì lưu job triển khai và trả lời ngay.
- **Thực thi chạy nền (Deployment Worker):** với từng tầng reconcile infrastructure, kể cả VPC và cụm trên cloud (hoặc chỉ liên kết cụm nội bộ và resource dùng chung), resolve configuration, sinh manifest, bảo đảm Delivery Repository của application tồn tại rồi gửi desired state sang CD system, chờ workload healthy và thu output; tự động làm lại resource và workload phụ thuộc khi output thay đổi; gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản.

## System operations

- **createDeployment()** - Tạo deployment mới từ application, phiên bản Application Definition và phiên bản catalog được chọn, environment (staging hoặc production), nơi triển khai và image version của các workload được chọn.

- **validateDeploymentInput()** - Kiểm tra image version, deployment context, Environment Configuration khớp với phiên bản được chọn, và luật deploy một phần chỉ dùng phiên bản Application Definition và phiên bản catalog đang chạy.

- **buildDeploymentGraph()** - Dựng dependency/resource graph từ workload, resource, dependency, configuration reference, phiên bản catalog và deployment context; tự thêm cụm Kubernetes và những gì Resource Definition khai báo là cần thêm (`requires`), ví dụ network; phát hiện các quan hệ tạo thành vòng.

- **planDeploymentWaves()** - Xác định phạm vi deployment (workload được chọn, resource mà chúng depends on trực tiếp, cụm Kubernetes/network mà các resource đó cần), chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy; sau khi có infrastructure plan, liệt kê thành phần có thể bị làm lại (chỉ tính từ thành phần sẽ được tạo, cập nhật hoặc deploy).

- **resolveResourceDefinitions()** - Chọn Resource Definition phù hợp trong phiên bản catalog được chọn cho từng resource, cụm Kubernetes và network trong graph (được gọi trong lúc dựng graph), gồm cả việc thuộc loại do IDP quản lý hạ tầng hay loại trỏ tới thứ có sẵn (resource dùng chung, cụm nội bộ).

- **planInfrastructureChanges()** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target) và xác định resource nào cần tạo mới, cập nhật, tái sử dụng hoặc liên kết tới thứ có sẵn; resource cần cập nhật khi đầu vào hiện tại (công thức trong phiên bản catalog, tham số, output của những gì nó cần) khác đầu vào lần apply gần nhất; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **loadInfrastructureOverrides()** - Lấy các infrastructure parameter mà Developer được phép override.

- **applyInfrastructureOverrides()** - Ghi nhận các giá trị override mà Developer lựa chọn.

- **confirmDeployment()** - Xác nhận deployment sau khi Developer kiểm tra các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị làm lại và các override. Operation chỉ đổi trạng thái deployment sang đã xác nhận và tạo job triển khai (kèm các giá trị override đã chọn) trong cùng một lần lưu, rồi trả lời ngay; không tự thực thi việc triển khai.

- **reconcileInfrastructure()** - Do Deployment Worker gọi. Thực thi việc tạo/cập nhật infrastructure của các resource do IDP quản lý trong một tầng (kể cả VPC, cụm Kubernetes trên cloud) thông qua provisioner phù hợp, với input gồm output của những gì resource cần; hoặc liên kết Resource Instance tới thứ có sẵn (resource dùng chung, cụm nội bộ) mà không đụng hạ tầng thật; với thành phần không còn trong phiên bản, thực thi việc hủy resource hoặc gỡ liên kết resource dùng chung.

- **collectResourceOutputs()** - Thu thập Resource Output sau khi infrastructure resource sẵn sàng.

- **resolveEnvironmentConfiguration()** - Resolve configuration của các workload trong một tầng từ direct value, Resource Output và Workload Output.

- **generateResolvedApplicationSpecification()** - Tạo resolved application specification chứa image version, configuration và dependency đã resolve.

- **generateKubernetesManifest()** - Sinh base Kubernetes manifest từ resolved specification bằng score-k8s.

- **adaptManifestForTarget()** - Áp dụng target-specific patch/adaptation cho Kubernetes manifest nếu deployment target yêu cầu.

- **materializeEnvironmentConfiguration()** - Chuyển Environment Variable đã resolve thành Kubernetes configuration, ví dụ ConfigMap hoặc cấu hình tương ứng.

- **materializeSecretConfiguration()** - Chuyển Secret đã resolve thành Kubernetes Secret hoặc secret reference phù hợp.

- **publishDesiredDeploymentState()** - Bảo đảm application có Delivery Repository của riêng nó (tạo nơi chứa và cặp khóa ở lần đầu, ghi nhận lại), rồi gửi desired deployment state của các workload trong một tầng sang CD abstraction; khi deploy phiên bản mới, desired state không còn các workload bị gỡ để CD system gỡ chúng khỏi cluster.

- **waitForWorkloadsHealthy()** - Chờ tới khi pod của các workload trong tầng healthy trên deployment target.

- **collectWorkloadOutputs()** - Thu thập Workload Output từ workload vừa healthy, hoặc từ workload đang chạy ngoài phạm vi deployment.

- **propagateOutputChanges()** - So sánh dấu vân tay output mới với lần triển khai trước; nếu thay đổi, reconcile lại các resource cần thành phần đó và thêm các workload dựa trên nó vào các tầng sau với image version đang chạy.

- **saveDeploymentRecord()** - Lưu Deployment Record, image version, infrastructure reference, tiến trình theo tầng/thành phần và trạng thái thực thi.

Chuỗi chính gồm hai phần:

- **Trong request của Developer:** tạo deployment (chọn phiên bản Application Definition và phiên bản catalog) → kiểm tra input và configuration khớp phiên bản → dựng graph (resolve Resource Definition, thêm cụm/network) → chia tầng → plan/override infra (gồm gỡ/hủy/gỡ liên kết) → liệt kê thành phần có thể bị làm lại → xác nhận: lưu job và trả lời ngay.
- **Deployment Worker chạy nền:** với mỗi tầng: reconcile infra (kể cả VPC, cụm) hoặc liên kết thứ có sẵn → thu Resource Output → resolve config → sinh resolved spec → sinh base manifest → adapt theo target → materialize config/secret → publish sang CD → chờ healthy → thu Workload Output → lan truyền thay đổi output (reconcile lại resource, triển khai lại workload); sau các tầng: gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản → lưu deployment record.

## Participating components

### Boundary/UI

- **Web UI** - Cho Developer chọn environment, nơi triển khai (cloud hoặc cụm nội bộ), phiên bản Application Definition, phiên bản catalog, workload cần deploy và image version, xem các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị làm lại, nhập override và xác nhận deploy.

- **Deployment API / Controller** - Nhận request từ UI, validate ở mức request và chuyển sang Deployment Orchestrator.

### Application services

- **Deployment Orchestrator** - Điều phối phần xử lý trong request của Developer: tạo deployment theo phiên bản Application Definition và phiên bản catalog được chọn, kiểm tra input, dựng graph, chia tầng, lập plan; khi Developer xác nhận thì đổi trạng thái deployment và lưu job triển khai trong cùng một lần lưu rồi trả lời ngay. Không tự thực thi việc triển khai.

- **Deployment Worker** - Tiến trình chạy nền nhận job triển khai đã lưu và điều phối phần thực thi: triển khai lần lượt từng tầng tới khi mọi workload trong phạm vi healthy, lan truyền khi output thay đổi (reconcile lại resource, triển khai lại workload), gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản, và cập nhật trạng thái deployment.

### Domain components

- **Deployment Graph Builder** - Dựng dependency/resource graph từ Application Definition, Environment Configuration, phiên bản catalog và deployment context; tự thêm cụm Kubernetes và những gì Resource Definition `requires` (gọi Resource Definition Resolver trong lúc dựng); phát hiện các quan hệ tạo thành vòng.

- **Deployment Wave Planner** - Xác định phạm vi deployment, chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy, liệt kê thành phần có thể bị làm lại, và lan truyền khi output thay đổi bằng cách reconcile lại resource phụ thuộc và thêm workload phụ thuộc vào các tầng sau.

- **Resource Definition Resolver** - Chọn Resource Definition phù hợp trong phiên bản catalog được chọn cho từng logical resource, cụm Kubernetes và network, dựa trên type và deployment context; phân biệt definition do IDP quản lý hạ tầng với definition trỏ tới thứ có sẵn (resource dùng chung, cụm nội bộ).

- **Infrastructure Planner** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target), so sánh đầu vào hiện tại (công thức trong phiên bản catalog, tham số, output của những gì resource cần) với đầu vào lần apply gần nhất để xác định cần create, update, reuse hay liên kết tới thứ có sẵn; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **Infrastructure Reconciler** - Điều phối việc reconcile infrastructure của từng tầng theo plan đã xác định: tạo/sửa resource do IDP quản lý (kể cả VPC, cụm Kubernetes trên cloud) với input từ output của những gì resource cần, chỉ liên kết (không đụng hạ tầng thật) với thứ có sẵn (resource dùng chung, cụm nội bộ), và thực hiện hủy resource hoặc gỡ liên kết resource không còn trong phiên bản.

- **Resource Output Resolver / Collector** - Thu thập output từ infrastructure đã provision, ví dụ host, port, username, password.

- **Workload Output Collector** - Thu thập output từ workload đã healthy hoặc đang chạy, ví dụ endpoint. Cơ chế đọc output cụ thể chưa được chốt.

- **Environment Configuration Resolver** - Resolve direct value, Resource Output reference và Workload Output reference thành configuration thực tế cho các workload trong một tầng.

- **Resolved Specification Generator** - Tạo resolved application specification chứa image version, dependency và configuration đã resolve.

- **Manifest Generator / Score Renderer** - Gọi score-k8s để sinh base Kubernetes manifest từ resolved specification.

- **Target Manifest Adapter** - Áp dụng patch/adaptation riêng theo deployment target.

- **Environment Configuration Materializer** - Chuyển Environment Variable đã resolve thành Kubernetes configuration tương ứng, ví dụ ConfigMap.

- **Secret Materializer** - Chuyển Secret thành Kubernetes Secret hoặc secret reference phù hợp mà không làm lộ plaintext.

### Integration abstractions

- **Provisioner Adapter / Provisioner Interface** - Abstraction cho cơ chế provision infrastructure; implementation cụ thể có thể gọi Terraform/OpenTofu/module tương ứng.

- **CD Integration / CD Provider Interface** - Abstraction để publish desired deployment state mà không phụ thuộc trực tiếp vào Fleet, Argo CD, Flux hay implementation cụ thể.

- **Delivery Repository Provider** - Abstraction để bảo đảm nơi chứa desired state của một application tồn tại: tạo nơi chứa theo quy ước đặt tên, sinh và gắn cặp khóa của application. Không phụ thuộc vào một hệ thống lưu trữ Git cụ thể.

- **Workload Status Provider / Kubernetes Adapter** - Kiểm tra workload đã healthy chưa và đọc dữ liệu runtime của workload phục vụ Workload Output Collector.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể của CD abstraction, ví dụ Fleet Adapter, Argo CD Adapter hoặc Flux Adapter.

- **Concrete Git Hosting Provider** - Implementation cụ thể của Delivery Repository Provider cho một hệ thống lưu trữ Git, ví dụ GitHub Adapter; đọc thông tin đăng nhập từ Secret Store.

### External systems

- **Terraform/OpenTofu Runner** - Thực thi Terraform/OpenTofu module để tạo hoặc cập nhật infrastructure theo yêu cầu từ Provisioner Adapter.

- **score-k8s** - Sinh base Kubernetes manifest từ resolved application specification.

- **CD System** - Hệ thống CD bên ngoài, ví dụ Fleet, Argo CD hoặc Flux, nhận desired deployment state và đồng bộ xuống Kubernetes.

- **Kubernetes Cluster** - Nơi workload thực sự chạy. Trên cloud, cụm do IDP dựng qua Provisioner; với cụm nội bộ, cụm có sẵn và được platform khai báo trong catalog. Thông tin kết nối cụm lấy từ output của `k8s-cluster`. Cung cấp trạng thái health và dữ liệu runtime của workload.

### Persistence

- **Application Repository** - Đọc phiên bản Application Definition được chọn (đã lưu từ UC-01); tạo Platform Requirement (cụm Kubernetes, network) của application khi cần lần đầu.

- **Environment Configuration Repository** - Đọc configuration/reference đã lưu từ UC-02.

- **Resource Definition Catalog** - Đọc các phiên bản catalog và Resource Definition của phiên bản được chọn. Catalog do platform quản lý; IDP chỉ đọc.

- **Delivery Repository Registry** - Lưu/đọc Delivery Repository của từng application: nơi chứa desired state, nhánh và tham chiếu tới cặp khóa trong Secret Store.

- **Resource Instance Repository** - Lưu/đọc Resource Instance theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target): trạng thái, reference, liên kết tới thứ có sẵn, dấu vân tay output và dấu vân tay đầu vào lần apply gần nhất, phục vụ reconcile/reuse, gỡ/hủy và lan truyền thay đổi output.

- **Workload Instance Repository** - Lưu/đọc trạng thái hiện hành của từng workload theo environment và deployment target: image đang chạy, trạng thái health và dấu vân tay output.

- **Deployment Repository** - Lưu deployment (kèm phiên bản Application Definition và phiên bản catalog), job triển khai cùng giá trị override đã chọn, Deployment Record, image version, target, infrastructure reference, tiến trình theo tầng/thành phần, status và lỗi nếu có.

Luồng responsibility:

- **Trong request:** Web UI → Deployment API → Deployment Orchestrator → Graph Builder (→ Resource Definition Resolver → Resource Definition Catalog) → Wave Planner → Infrastructure Planner → Wave Planner (thành phần có thể bị làm lại) → Deployment Repository (lưu deployment; khi xác nhận thì lưu job) → Web UI.
- **Chạy nền:** Deployment Worker → (với mỗi tầng) Infrastructure Reconciler → Provisioner → Terraform/OpenTofu Runner → Resource Output Collector → Configuration Resolver → Resolved Spec Generator → Score Renderer → score-k8s → Target Adapter → Config/Secret Materializer → Delivery Repository Provider (bảo đảm nơi chứa của application) → CD Integration → Concrete CD Provider → CD System → Kubernetes → Workload Status Provider → Workload Output Collector → Wave Planner (lan truyền sang resource và workload) → (sau các tầng) gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản → Resource/Workload Instance Repository → Deployment Repository.

Deployment Orchestrator và Deployment Worker chỉ điều phối. Các việc dựng graph, chia tầng và lan truyền thay đổi output, resolve Resource Definition, reconcile infrastructure, resolve configuration, sinh manifest, giao tiếp với CD và thu output nằm ở các component riêng.

## Detailed design artifacts

- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../../04_operation_contracts/operation_contracts.md)
- [Traceability matrix](../../../06_traceability/traceability_matrix.md)
