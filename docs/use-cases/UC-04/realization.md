---
id: UC-04-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-04
last_reviewed: 2026-09-17
---

# UC-04 — View Deployment Result: Use Case Realization

## Responsibility

Chịu trách nhiệm đọc và hiển thị trạng thái/kết quả của deployment, gồm infrastructure, configuration resolution, CD status, workload health, endpoint và image version thực tế. Không thay đổi application hay infrastructure.

## System operations

- **listDeployments()** - Lấy danh sách lịch sử deployment của application.

- **getDeploymentDetail()** - Lấy thông tin chi tiết của một deployment cụ thể.

- **getDeploymentProgress()** - Lấy tiến trình thực thi của deployment theo từng tầng, từng thành phần và từng bước.

- **getInfrastructureStatus()** - Lấy trạng thái của infrastructure liên quan đến deployment.

- **getCDStatus()** - Lấy trạng thái đồng bộ/triển khai từ CD system thông qua CD abstraction.

- **getWorkloadStatus()** - Lấy trạng thái health của từng workload trong deployment.

- **getDeploymentEndpoints()** - Lấy endpoint được expose sau deployment nếu có.

- **getDeploymentImages()** - Lấy image repository và image version thực tế đã dùng cho từng workload.

- **getDeploymentFailureDetail()** - Lấy failed step, workload/resource liên quan và error summary khi deployment thất bại.

UC-04 là read-only: đọc dữ liệu từ Deployment Record và các nguồn trạng thái liên quan, sau đó tổng hợp để hiển thị cho Developer. UC-04 không trigger reconcile, không sửa infrastructure và không redeploy.

## Participating components

### Boundary/UI

- **Web UI** - Hiển thị lịch sử deployment, trạng thái từng bước, workload health, infrastructure status, image version, endpoint và lỗi nếu có.

- **Deployment Query API / Controller** - Nhận request đọc dữ liệu từ UI và chuyển sang query service.

### Application services

- **Deployment Query Service** - Điều phối việc tổng hợp dữ liệu cần hiển thị cho một deployment.

### Domain components

- **Deployment Result Aggregator** - Tổng hợp Deployment Record + infrastructure status + CD status + workload health thành một view model thống nhất cho UI.

### Integration abstractions

- **CD Integration / CD Status Provider** - Lấy trạng thái deployment/sync từ concrete CD implementation thông qua abstraction.

- **Workload Status Provider / Kubernetes Adapter** - Lấy workload health, pod/deployment status và endpoint từ Kubernetes cluster.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể để truy vấn Fleet, Argo CD, Flux hoặc CD system khác.

### External systems

- **CD System** - Hệ thống CD bên ngoài, ví dụ Fleet, Argo CD hoặc Flux, cung cấp trạng thái deployment và trạng thái đồng bộ.

- **Kubernetes Cluster** - Nguồn trạng thái runtime thực tế của workload.

### Persistence

- **Deployment Repository** - Đọc Deployment Record, image version, target, trạng thái từng bước và error summary.

- **Resource Instance Repository** - Đọc thông tin infrastructure reference và trạng thái resource liên quan đến deployment.

- **Workload Instance Repository** - Đọc image version đang chạy của từng workload theo environment và deployment target, để cho biết deployment đang xem có còn là bản đang chạy hay đã được thay thế.

Luồng responsibility: Web UI → Deployment Query API → Deployment Query Service → Deployment Repository + Resource Repository + Workload Instance Repository + CD Status Provider → Concrete CD Provider → CD System + Kubernetes Adapter → Kubernetes Cluster → Result Aggregator → Web UI.

UC-04 không dùng Deployment Orchestrator để thực hiện hành động. UC-04 đi theo query path riêng vì chỉ đọc và tổng hợp trạng thái, không reconcile infrastructure hoặc trigger deployment.

## Detailed design artifacts

- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../../06_traceability/traceability_matrix.md)
