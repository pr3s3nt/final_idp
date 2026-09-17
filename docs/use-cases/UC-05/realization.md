---
id: UC-05-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-05
last_reviewed: 2026-09-17
---

# UC-05 — Remove Application from Environment: Use Case Realization

## Responsibility

Chịu trách nhiệm gỡ application khỏi **một** environment và một nơi triển khai: lập plan gỡ bỏ theo thứ tự ngược với thứ tự triển khai dựa trên đúng phiên bản đang chạy (Developer không chọn lại phiên bản hay image), cho Developer xem kèm cảnh báo mất dữ liệu, rồi khi được xác nhận thì chạy nền để gỡ workload, hủy resource do IDP tạo và gỡ liên kết resource dùng chung. Không xóa Application Definition, phiên bản, Environment Configuration hay Delivery Repository, và không đụng tới environment khác.

**Ranh giới giữa năm Use Case**: UC-01 định nghĩa cần gì → UC-02 định nghĩa giá trị theo environment là gì → UC-03 quyết định triển khai chúng như thế nào → UC-04 cho biết kết quả ra sao → UC-05 thu hồi những gì UC-03 đã tạo ra trên một environment.

## System operations

- **createTeardown()** - Nhận application, environment và nơi triển khai; lấy lại phiên bản Application Definition, phiên bản catalog và Environment Configuration của lần deploy gần nhất, dựng plan gỡ bỏ theo thứ tự ngược và lưu deployment loại gỡ bỏ ở trạng thái chờ xác nhận.

- **confirmDeployment()** - Dùng lại operation của UC-03: kiểm tra plan chưa đổi, ghi xác nhận và tạo job trong một transaction, rồi trả lời ngay.

- **removeWorkloadsAndInfrastructure()** - Deployment Worker chạy các tầng gỡ theo thứ tự ngược: publish desired state không còn workload, xác minh workload đã biến mất, gỡ đối tượng khỏi CD system và xóa desired state của environment, xóa không gian tên, rồi hủy hoặc gỡ liên kết từng resource.

- **saveDeploymentRecord()** - Dùng lại operation của UC-03 để lưu kết quả lần gỡ.

## Participating components

### Boundary/UI

- **Web UI** - Cho Developer chọn environment và nơi triển khai cần gỡ, hiển thị plan gỡ bỏ kèm cảnh báo mất dữ liệu, rồi theo dõi tiến trình gỡ.

- **Deployment API / Controller** - Nhận yêu cầu lập plan gỡ bỏ và yêu cầu xác nhận, chuyển sang orchestrator. Dùng chung controller với UC-03.

### Application services

- **Deployment Orchestrator** - Lấy lại phiên bản, phiên bản catalog và Environment Configuration của lần deploy gần nhất trên đúng chủ sở hữu, dựng plan gỡ bỏ, lưu deployment loại gỡ bỏ, và khi được xác nhận thì tạo job rồi trả lời ngay. Không tự thực thi.

- **Deployment Worker** - Chạy nền các tầng gỡ theo thứ tự ngược: gỡ workload, gỡ đối tượng khỏi CD system, xóa không gian tên, hủy hoặc gỡ liên kết resource, rồi lưu Deployment Record.

### Domain components

- **Deployment Graph Builder** - Dựng lại graph của phiên bản đang chạy để biết thứ tự phụ thuộc cần đảo ngược khi gỡ.

- **Infrastructure Planner** - Lập plan gỡ bỏ: tầng đầu là workload, các tầng sau là resource với hành động hủy (do IDP tạo) hoặc gỡ liên kết (có sẵn, dùng chung), kèm cảnh báo mất dữ liệu.

- **Infrastructure Reconciler** - Hủy hạ tầng do IDP tạo theo đúng thứ tự ngược.

### Integration abstractions

- **CD Integration / CD Provider Interface** - Gửi desired state không còn workload, rồi gỡ đối tượng của application khỏi CD system và xóa phần desired state của environment khỏi Delivery Repository.

- **Workload Status Provider / Kubernetes Adapter** - Xác minh workload đã biến mất khỏi cụm và xóa không gian tên của application.

- **Provisioner Adapter / Provisioner Interface** - Hủy hạ tầng qua provisioner tương ứng của từng Resource Definition.

### Integration implementations

- **Concrete CD Provider** - Hiện thực cụ thể cho Fleet, Argo CD hoặc CD system khác.

### External systems

- **CD System**, **Kubernetes Cluster**, **Terraform/OpenTofu Runner** - Nơi thao tác gỡ bỏ thực sự diễn ra.

### Persistence

- **Deployment Repository** - Tìm deployment gần nhất của chủ sở hữu, lưu deployment loại gỡ bỏ, tạo job, lưu Deployment Record của lần gỡ.

- **Resource Instance Repository** - Đọc Resource Instance thuộc chủ sở hữu và cập nhật sang trạng thái đã hủy hoặc đã gỡ liên kết.

- **Workload Instance Repository** - Đọc workload đang chạy và cập nhật sang trạng thái đã gỡ sau khi đã xác minh.

- **Delivery Repository Registry** - Đọc nơi chứa desired state của application để xóa đúng phần của environment bị gỡ; row đăng ký không bị xóa.

- **Application Repository**, **Environment Configuration Repository**, **Resource Definition Catalog** - Chỉ đọc, để dựng lại graph và plan của phiên bản đang chạy.

Luồng responsibility: Web UI → Deployment API → Deployment Orchestrator → Deployment Repository + Application Repository + Environment Configuration Repository + Resource Definition Catalog + Graph Builder + Infrastructure Planner → (Developer xác nhận) → Deployment Worker → CD Integration → Concrete CD Provider → CD System, Kubernetes Adapter → Kubernetes Cluster, Infrastructure Reconciler → Provisioner → Terraform Runner → Resource/Workload Instance Repository → Deployment Repository.

UC-05 không thêm class mới nào so với UC-03: nó dùng lại đúng các thành phần đó, chỉ khác ở chỗ plan chỉ gồm các tầng gỡ bỏ và thứ tự là ngược lại. Application Repository và Environment Configuration Repository chỉ được đọc; không use case nào trong bốn use case còn lại, và cả UC-05, xóa Application Definition hay Environment Configuration.

## Detailed design artifacts

- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../../04_operation_contracts/operation_contracts.md)
- [Traceability matrix](../../../06_traceability/traceability_matrix.md)
