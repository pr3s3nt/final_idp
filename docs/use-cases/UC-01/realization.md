---
id: UC-01-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-01
last_reviewed: 2026-09-17
---

# UC-01 — Create / Configure Application: Use Case Realization

## Responsibility

Chịu trách nhiệm khai báo cấu trúc logic của application: workload, resource requirement, dependency và các Environment Variable/Secret mà workload cần. Mỗi lần lưu, ghi Application Definition thành một phiên bản mới (không ghi đè phiên bản cũ, các thành phần giữ ID cố định qua phiên bản) và sinh/cập nhật application specification. Không deploy và không làm thay đổi environment nào.

## System operations

- **createApplication()** - Tạo một Application Definition mới.

- **updateApplication()** - Cập nhật thông tin application đã tồn tại.

- **addResourceRequirement()** - Thêm resource logic mà application cần, ví dụ PostgreSQL hoặc Redis.

- **addWorkload()** - Thêm workload và các thông tin như type, image repository, port.

- **defineConfigurationRequirement()** - Khai báo Environment Variable và Secret mà workload cần.

- **defineDependency()** - Khai báo quan hệ depends on giữa workload và resource/workload khác.

- **validateApplicationDefinition()** - Kiểm tra tính hợp lệ của workload, resource, dependency và configuration requirement.

- **saveApplicationDefinition()** - Lưu Application Definition thành một phiên bản mới; phiên bản cũ giữ nguyên, các thành phần giữ ID cố định qua phiên bản.

- **generateApplicationSpecification()** - Sinh hoặc cập nhật application specification, ví dụ score.yaml, từ Application Definition đã lưu.

Ở mức Use Case Realization, không tách nhỏ hơn thành các operation như addEnvironmentVariable(), addSecret() hoặc validatePort() để tránh làm sequence diagram quá vụn.

## Participating components

### Boundary/UI

- **Web UI** - Cho Developer khai báo application, workload, resource, dependency và configuration requirement.

- **Application API / Controller** - Nhận request từ UI, validate ở mức request và điều phối sang application service.

### Application services

- **Application Service** - Chịu trách nhiệm xử lý use case tạo/cập nhật Application Definition.

### Domain components

- **Application Definition Validator** - Kiểm tra tính hợp lệ của workload, resource, dependency, port, image repository và configuration requirement.

- **Application Specification Generator** - Chuyển Application Definition thành application specification tương ứng, ví dụ score.yaml.

### Persistence

- **Application Repository** - Lưu và đọc Application Definition theo phiên bản: mỗi lần lưu thêm một phiên bản mới, không sửa hay xóa phiên bản cũ; giữ ID cố định của các thành phần qua phiên bản.

- **Specification Repository / Config Repo Service** - Lưu application specification đã sinh nếu hệ thống cần persist hoặc version hóa artifact này.

Luồng trách nhiệm: Web UI → API/Controller → Application Service → Validator → Repository + Specification Generator.

UC-01 chưa cần Deployment Orchestrator, Resource Definition Resolver, Infrastructure Reconciler, CD Integration hoặc Kubernetes Cluster vì use case chỉ dừng ở việc định nghĩa application và sinh specification, chưa deploy.

## Detailed design artifacts

- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../../06_traceability/traceability_matrix.md)
