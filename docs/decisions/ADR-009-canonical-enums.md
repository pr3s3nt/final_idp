---
id: ADR-009
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/design-decisions-log.md"
---

# ADR-009 — Nguồn chuẩn cho literal ENUM

**Current status (2026-09-17):** accepted. [`docs/architecture/database/schema.md`](../architecture/database/schema.md) is the canonical catalog of literal ENUM values.

**Vấn đề:** `docs/architecture/database/schema.md` có ba cột ENUM đã ghi giá trị (`dependency.target_type`, `configuration_value.value_source`, `secret.value_source`), nhưng bốn cột trạng thái chỉ ghi `ENUM` mà không có giá trị: `resource_instance.status`, `deployment.status`, `deployment_record.status`, `deployment_step.status`. Chính `operation_contracts.md` (dòng 3) và `docs/architecture/state-machines/README.md` (dòng 5) ghi nhận "chưa chốt tập literal vật lý". Hệ quả:

- Contract và state machine dùng tên "logical state", còn DB không nói giá trị thật; khi code mỗi người tự đặt tên (`READY`, `Ready`, `AVAILABLE`…).
- Guard và truy vấn dựa trên giá trị cụ thể (ví dụ `status = 'AWAITING_CONFIRMATION'`, chỉ tìm Resource Instance `READY`) sẽ không khớp khi giá trị không thống nhất.

**Quyết định:**

1. **Một bảng danh mục ENUM duy nhất trong `schema.md`** liệt kê mọi cột ENUM và tập giá trị; contract, state machine, domain model dùng đúng các giá trị trong bảng này.
2. **Quy ước tên:** `UPPER_SNAKE_CASE`, như các ENUM đang có.
3. **`environment` là ENUM** với hai giá trị cố định (theo vấn đề 5).
4. **Gỡ xong thì giữ dòng Instance với trạng thái kết thúc** (`DESTROYED`, `UNLINKED`, `REMOVED`) thay vì xóa dòng, vì deployment record cũ vẫn trỏ tới Instance (tránh lỗi lịch sử trỏ vào chỗ trống của vấn đề 5).
5. **Danh mục chia hai phần:** "Đã chốt" và "Dự kiến, chưa chốt" (ghi rõ chốt khi giải quyết D3/D4/D5), để không nhầm giá trị dự kiến là đã chốt.

**ENUM đã chốt:**

| ENUM | Giá trị | Nguồn |
|---|---|---|
| `dependency.target_type` | `WORKLOAD`, `RESOURCE` | Có sẵn |
| `configuration_value.value_source` | `DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT` | Có sẵn |
| `secret.value_source` | `SECRET_REF`, `RESOURCE_OUTPUT` | Có sẵn |
| `deployment.status` | `AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED`, `FAILED` | Vấn đề 2 |
| `resource_instance.status` | `PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `DESTROYED`, `UNLINKED` | State machine + vấn đề 3, 5 |
| `workload_instance.status` | `DEPLOYING`, `HEALTHY`, `FAILED`, `REMOVED` | Vấn đề 2, 5 |
| `environment` | `STAGING`, `PRODUCTION` | Vấn đề 5 |
| `workload_deployment.inclusion_reason` | `SELECTED`, `CASCADED` | Vấn đề 2 |
| Loại quản lý của `resource_definition` | `MANAGED` (IDP tạo/sửa/hủy), `EXISTING` (trỏ tới resource có sẵn, chỉ đọc) | Vấn đề 3 |

`UNLINKED` dành cho Resource Instance trỏ tới resource dùng chung: app thôi dùng thì chỉ gỡ liên kết, hạ tầng thật vẫn còn.

**ENUM dự kiến, chưa chốt:**

| ENUM | Giá trị dự kiến | Chốt khi |
|---|---|---|
| Loại thành phần trong plan | `RESOURCE`, `WORKLOAD` | D3 |
| Action cho resource | `CREATE`, `UPDATE`, `REUSE`, `DESTROY`, `UNLINK` | D3 |
| Action cho workload | `DEPLOY`, `REMOVE` | D3 |
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` | D4 |
| `deployment_step.step_name` (đổi từ VARCHAR sang ENUM) | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, `CD_SYNCED`, `APPLICATION_READY`, `REMOVED`, `DESTROYED`, `UNLINKED` | D4 |
| `deployment_record.status` | Bỏ cột, dùng `deployment.status` | D5 |
| `delivery_status` (trường riêng, nếu lưu) | `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED` | D5 |

**Sẽ ảnh hưởng:** `schema.md` (bảng danh mục ENUM, các cột status), `erd.puml`, domain model, operation contracts (dòng 3 và các status được nhắc), `docs/architecture/state-machines/README.md` (dòng 5) cùng các state machine, traceability.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** mục **Danh mục ENUM** trong `docs/architecture/database/schema.md` (đã chốt / dự kiến, thêm `application_component.component_type` và `deployment_execution_job.status` dự kiến theo D6), các cột status trong `schema.md`/`erd.puml`, domain model, đoạn mở đầu operation contracts, `docs/architecture/state-machines/README.md` cùng ba state machine; giá trị dự kiến đã ghi vào `deferred_issues.md` các mục D3, D4, D5, D6.
