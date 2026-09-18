---
id: UC-01-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-01
last_reviewed: 2026-09-18
---

# UC-01 — Create / Configure Application: Use Case Realization

## Responsibility

Chịu trách nhiệm khai báo cấu trúc logic của application: workload, resource requirement, dependency và các Environment Variable/Secret mà workload cần. Web UI sở hữu `ApplicationDefinitionDraft` trong browser tab và lưu phần không nhạy cảm vào `sessionStorage`; Application Service không giữ draft giữa các request. Mỗi lần lưu, Web UI gửi toàn bộ draft cùng `baseVersion`; backend kiểm tra optimistic concurrency, ghi Application Definition thành một phiên bản mới (không ghi đè phiên bản cũ, các thành phần giữ ID cố định qua phiên bản) và sinh/cập nhật application specification. Không deploy và không làm thay đổi environment nào.

## System operations

- **createApplication()** - Web UI khởi tạo một `ApplicationDefinitionDraft` mới trong browser, chưa ghi backend.

- **updateApplication()** - Tải phiên bản Application Definition mới nhất để Web UI tạo draft chỉnh sửa với `baseVersion`.

- **addResourceRequirement()** - Web UI thêm resource logic vào client-owned draft, ví dụ PostgreSQL hoặc Redis.

- **addWorkload()** - Web UI thêm workload và các thông tin như type, image repository, port vào draft.

- **defineConfigurationRequirement()** - Web UI khai báo Environment Variable và Secret mà workload cần trong draft.

- **defineDependency()** - Web UI khai báo quan hệ depends on giữa workload và resource/workload khác trong draft.

- **validateApplicationDefinition()** - Kiểm tra tính hợp lệ của workload, resource, dependency và configuration requirement.

- **saveApplicationDefinition()** - Nhận toàn bộ draft, kiểm tra `baseVersion`, rồi lưu Application Definition thành một phiên bản mới; stale draft bị từ chối không ghi dữ liệu, phiên bản cũ giữ nguyên và các thành phần giữ ID cố định qua phiên bản.

- **generateApplicationSpecification()** - Sinh hoặc cập nhật application specification, ví dụ score.yaml, từ Application Definition đã lưu.

Các operation chỉnh field/component là interaction operation của Web UI và không tạo API round-trip. Ở mức Use Case Realization, không tách nhỏ hơn thành các operation như addEnvironmentVariable(), addSecret() hoặc validatePort() để tránh làm sequence diagram quá vụn.

## Participating components

### Boundary/UI

- **Web UI** - React web application theo [ADR-017](../../decisions/ADR-017-react-web-frontend.md). Sở hữu `ApplicationDefinitionDraft`, cho Developer khai báo application, workload, resource, dependency và configuration requirement, serialize/restore phần không nhạy cảm trong `sessionStorage`, rồi gửi toàn bộ draft khi Save.

  Editor sử dụng cấu trúc Application Builder đã chốt trong
  [UI design](ui/README.md): navigation theo từng component, form tập trung cho
  component đang chọn và Review topology chỉ đọc. Đây là cách trình bày các
  operation client-owned ở trên, không tạo thêm API operation.

- **Application API / Controller** - Tải phiên bản mới nhất cho edit; khi Save, nhận toàn bộ draft, validate ở mức request và điều phối sang application service. Không cung cấp endpoint cho từng field/component edit.

### Application services

- **Application Service** - Stateless giữa các edit request; kiểm tra `baseVersion`, validate và xử lý việc lưu Application Definition.

### Domain components

- **Application Definition Validator** - Kiểm tra tính hợp lệ của workload, resource, dependency, port, image repository và configuration requirement.

- **Application Specification Generator** - Chuyển Application Definition thành application specification tương ứng, ví dụ score.yaml.

### Persistence

- **Application Repository** - Lưu và đọc Application Definition theo phiên bản: mỗi lần lưu thêm một phiên bản mới, không sửa hay xóa phiên bản cũ; giữ ID cố định của các thành phần qua phiên bản.

- **Specification Repository / Config Repo Service** - Lưu application specification đã sinh nếu hệ thống cần persist hoặc version hóa artifact này.

Luồng trách nhiệm khi edit: Web UI + browser `sessionStorage`. Luồng load/save: Web UI → API/Controller → Application Service → Validator → Repository + Specification Generator.

UC-01 chưa cần Deployment Orchestrator, Resource Definition Resolver, Infrastructure Reconciler, CD Integration hoặc Kubernetes Cluster vì use case chỉ dừng ở việc định nghĩa application và sinh specification, chưa deploy.

## Detailed design artifacts

- [Application Builder UI design](ui/README.md)
- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../traceability/matrix.md)
