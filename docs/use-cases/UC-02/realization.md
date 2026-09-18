---
id: UC-02-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-02
last_reviewed: 2026-09-17
---

# UC-02 — Configure Application Environment: Use Case Realization

## Responsibility

Chịu trách nhiệm khai báo giá trị cấu hình theo từng environment, bao gồm giá trị trực tiếp hoặc reference tới Resource Output / Workload Output. Web UI sở hữu `EnvironmentConfigurationDraft`, lưu phần không nhạy cảm trong `sessionStorage` và không lưu plaintext Secret ở đó; Environment Configuration Service không giữ draft giữa các request. Không thực hiện deployment.

## System operations

- **selectEnvironment()** - Tải requirements, Environment Configuration hiện hành, danh sách Catalog Version và danh sách deployment target để Web UI tạo client-owned draft cho environment đã chọn.

- **loadConfigurationRequirements()** - Lấy danh sách Environment Variable và Secret đã được khai báo từ UC-01.

- **setDirectConfigurationValue()** - Web UI gán direct value cho Environment Variable trong draft; với Secret, plaintext được gửi qua secure staging flow và UI chỉ giữ opaque reference.

- **selectCatalogVersionAndTarget()** - Web UI ghi phiên bản Catalog Version và deployment target Developer chọn vào draft. Hai giá trị này quyết định Resource Definition nào áp dụng cho mỗi Resource Requirement, nên quyết định danh sách output hợp lệ. Chúng không được persist ([ADR-020](../../decisions/ADR-020-uc02-catalog-version-and-target.md)).

- **bindResourceOutput()** - Web UI gán configuration trong draft vào một Resource Output của resource mà workload depends on, ví dụ DB_HOST → postgresql.host.

- **bindWorkloadOutput()** - Web UI gán configuration trong draft vào một Workload Output của workload mà workload chứa biến depends on, ví dụ BACKEND_URL → backend.endpoint.

- **validateEnvironmentConfiguration()** - Kiểm tra giá trị, resource, workload và output reference có hợp lệ hay không, gồm việc output được tham chiếu thuộc thành phần mà workload depends on, và việc output của resource nằm trong Resource Definition mà Catalog Version + deployment target của draft resolve ra.

- **saveEnvironmentConfiguration()** - Nhận toàn bộ draft, kiểm tra `baseApplicationDefinitionVersion` và `baseConfigurationRevision`, rồi lưu configuration; stale draft bị từ chối không ghi dữ liệu.

UC-02 chỉ lưu value hoặc reference, chưa resolve giá trị thật của Resource Output / Workload Output. Việc resolve thuộc UC-03 khi deployment thực sự diễn ra.

Catalog Version và deployment target trong draft chỉ dùng để chọn Resource Definition lúc hiển thị và lúc validate. Chúng không được ghi vào `environment_configuration`, và UC-03 vẫn kiểm tra lại configuration theo Catalog Version của lần deploy.

## Participating components

### Boundary/UI

- **Web UI** - Sở hữu `EnvironmentConfigurationDraft`, cho Developer chọn environment, nhập giá trị configuration và chọn nguồn từ Resource Output hoặc Workload Output; serialize/restore phần không nhạy cảm trong `sessionStorage` và gửi toàn bộ draft khi Save.

- **Environment Configuration API / Controller** - Tải durable state/catalog data, nhận secure Secret staging request khi cần và nhận toàn bộ draft khi Save; không nhận từng field/binding edit thông thường.

### Application services

- **Environment Configuration Service** - Stateless giữa các edit request; điều phối load/catalog/Secret staging và kiểm tra concurrency, validate, persist khi Save.

### Domain components

- **Resource Output Catalog / Resource Definition Query** - Resolve một Resource Requirement về đúng một Resource Definition theo Catalog Version, deployment context suy ra từ deployment target và applicability condition của definition - cùng quy tắc UC-03 dùng - rồi cung cấp `exposedOutputs` và `sensitiveOutputs` của definition đó để Developer lựa chọn.

- **Workload Output Catalog** - Cung cấp danh sách output mà workload có thể expose, ví dụ endpoint.

- **Environment Configuration Validator** - Kiểm tra direct value, resource reference, workload reference và output được chọn có hợp lệ hay không, gồm việc output thuộc thành phần mà workload depends on.

### Integration abstractions

- **Secret Store / Secret Management Adapter** - Lưu giá trị Secret theo cơ chế bảo mật, tránh lưu plaintext trực tiếp trong configuration database.

Secret backend implementation cụ thể chưa được chốt, vì vậy tài liệu chưa liệt kê thành phần thuộc nhóm Integration implementations cho UC-02.

### Persistence

- **Application Query / Application Repository** - Lấy danh sách workload, Environment Variable, Secret và dependency đã được khai báo từ UC-01.

- **Environment Configuration Repository** - Lưu configuration và các reference riêng theo từng environment.

Luồng trách nhiệm khi edit thông thường: Web UI + browser `sessionStorage`. Luồng load/query/stage/save: Web UI → Configuration API → Environment Configuration Service → Application/Output Catalog → Validator → Secret Store + Environment Configuration Repository.

UC-02 chưa cần Configuration Resolver. Hệ thống chỉ lưu reference như DB_HOST → postgresql.host; giá trị thật chỉ được resolve trong UC-03 khi deploy.

## Detailed design artifacts

- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../traceability/matrix.md)
