---
id: ADR-003
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-003 — Phạm vi sở hữu và tái sử dụng Resource Instance

**Current status (2026-09-17):** accepted and applied to the current design and implementation. [D09](../backlog/D09-shared-and-workload-resources.md) tracks the deferred extensions.

**Vấn đề:** UC-03 tìm Resource Instance có sẵn bằng `findResourceInstances(resolvedResources, target)`, tức chỉ theo loại Resource Definition và deployment target. Resource Instance trong domain model và ERD cũng chỉ lưu `resourceDefinitionId` và `deploymentTarget`, không ghi thuộc app, environment hay requirement nào. Hệ quả là IDP có thể dùng nhầm resource:

- Hai app khác nhau cùng cần PostgreSQL RDS trên target `prod` → app sau dùng nhầm database của app trước.
- Hai environment (`dev`, `staging`) cùng deploy lên một cluster → dùng chung database ngoài ý muốn.
- Một app có hai requirement cùng loại (`orders-db`, `users-db`) → không phân biệt được instance nào của requirement nào.

Vấn đề này cũng ảnh hưởng vấn đề 2: deploy một phần phải đọc output của đúng resource mà workload phụ thuộc, và dấu vân tay output được lưu trên Resource Instance.

**Quyết định:** mặc định mỗi resource là riêng; dùng chung phải khai báo tường minh — theo mô hình Humanitec.

1. **Resource Instance thuộc về đúng một chủ:** application + environment + resource requirement + deployment target. Tìm để dùng lại theo đủ bộ này; không khớp thì tạo mới. Resource Instance tương ứng với "active resource" của Humanitec.
2. **Dùng chung giữa các app hoặc environment được khai báo ở Resource Definition** (do platform quản lý): platform tạo definition trỏ tới resource có sẵn, kèm điều kiện áp dụng (ví dụ mọi app ở environment `dev`). App khớp điều kiện thì Resource Instance của nó trỏ tới resource chung đó. Không dùng bảng binding cấp quyền giữa các app (hướng của `88585cc` không được chọn).
3. **Resource Instance trỏ tới resource có sẵn không được tạo, sửa hay xóa hạ tầng thật**, chỉ đọc output. Nhờ vậy deploy của app này không thể làm thay đổi resource mà app khác đang dùng.

Dùng chung trong cùng app + environment đã có sẵn trong thiết kế: nhiều workload cùng depends on một Resource Requirement thì dùng chung Resource Instance của requirement đó. Mức "resource riêng của từng workload" chưa cần trong giai đoạn này.

**Hệ quả cần xử lý khi sửa tài liệu:**

- `resource_instance` thêm liên kết tới application, environment, resource requirement; khóa tìm kiếm là (application, environment, resource requirement, deployment target).
- Bỏ ràng buộc `UNIQUE` trên `resource_instance.infrastructure_reference`, vì nhiều Resource Instance có thể trỏ cùng một resource chung.
- Resource Definition cần phân biệt loại **quản lý hạ tầng** (tạo/sửa/xóa) với loại **trỏ tới resource có sẵn** (chỉ đọc output).
- `findResourceInstances(...)` đổi tiêu chí tìm theo đủ bộ chủ sở hữu.

**Hoãn (đã ghi vào `deferred_issues.md`, mục D9):**

- Khi output của resource dùng chung thay đổi, chỉ app nào deploy lần sau mới phát hiện qua dấu vân tay output; IDP không tự deploy lại các app khác đang dùng chung.
- Resource riêng của từng workload (mức private theo workload của Humanitec).

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (UC-03, Bước 2, Bước 3), sequence UC-03, VOPC (`findResourceInstances` theo khóa chủ sở hữu), domain model (Resource Instance có chủ sở hữu; Resource Definition có `managementMode`), ERD (`resource_instance` có cột chủ sở hữu, partial UNIQUE, bỏ UNIQUE `infrastructure_reference`; `resource_definition.management_mode`), contracts 4–6, state machine Resource Instance (`EXISTING` → `READY`, `UNLINKED`), traceability; phần hoãn ghi vào D9.

**Sẽ ảnh hưởng (danh sách ban đầu):**

- `usecase_realization_step_1_3.md`:
  - Đặc tả UC-03: quy tắc nghiệp vụ về tìm resource để dùng lại theo chủ sở hữu và dùng chung khai báo ở Resource Definition.
  - Bước 2 UC-03: mô tả `resolveResourceDefinitions()`, `planInfrastructureChanges()`, `reconcileInfrastructure()` (tìm theo đủ bộ chủ sở hữu; resource loại `EXISTING` chỉ đọc output).
  - Bước 3 UC-03: Resource Definition Resolver, Infrastructure Planner, Infrastructure Reconciler, Resource Instance Repository.
- Các artifact khác: sequence UC-03, VOPC (`Resource Instance Repository`), domain model (Resource Instance, Resource Definition), ERD (`resource_instance`, `resource_definition`), operation contracts 4–6, state machine Resource Instance (instance trỏ resource có sẵn không đi qua provisioning), traceability.
