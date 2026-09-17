---
id: ADR-005
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/design-decisions-log.md
---

# ADR-005 — Phiên bản Application Definition bất biến

**Current status (2026-09-17):** accepted and applied to the current design and implementation.

**Vấn đề:** Contract 1 (`saveApplicationDefinition`) quy định thành phần bị loại khỏi definition thì **bị xóa hẳn** row, nhưng cũng trong contract đó cam kết không tạo, xóa hay sửa `Deployment`, `Workload Deployment`, `Environment Configuration`, `Resource Instance`. Hai điều này mâu thuẫn, vì nhiều bảng đang trỏ tới workload/definition qua khóa ngoại:

| Bảng | Trỏ tới |
|---|---|
| `workload_deployment.workload_id` | workload đã deploy (lịch sử) |
| `environment_variable.variable_definition_id`, `secret.secret_definition_id` | cấu hình theo environment |
| `configuration_value.workload_id`, `configuration_value.resource_requirement_id` | tham chiếu output |
| `dependency.target_workload_id`, `dependency.target_resource_requirement_id` | quan hệ phụ thuộc |

Vấn đề 2 và 3 còn thêm Workload Instance → workload và Resource Instance → resource requirement.

Ví dụ `worker` đã deploy 20 lần; bảng lịch sử chỉ lưu mã workload (W7) rồi tra bảng `workload` để hiển thị tên. Xóa hẳn dòng W7 thì chỉ có ba khả năng, đều sai: DB từ chối xóa (không bao giờ xóa được workload đã deploy); DB xóa dây chuyền (mất lịch sử); hoặc lịch sử trỏ vào chỗ trống (UC-04 hiển thị `???`). Thiết kế cũng không nói hạ tầng thật (database trên AWS, pod trên cluster) xử lý thế nào khi thành phần bị xóa.

**Quá trình bàn:**

1. Người dùng đưa quy tắc: muốn xóa một thành phần thì phải sửa trước những gì đang dùng nó (dependency, biến môi trường tham chiếu output của nó).
2. Quy tắc đó xử lý được phần "đang dùng" nhưng không xử lý được lịch sử; đã cân nhắc "ngừng dùng" (`retired_at`) và "lịch sử lưu bản chụp" (snapshot).
3. Gỡ hạ tầng ngay khi Save ở UC-01 bị loại: UC-01 không biết environment nào, app đang chạy vẫn còn dùng, hủy database là không lấy lại được, resource dùng chung không được hủy.
4. Phát hiện gốc vấn đề: cả hai environment dùng chung **một** bản định nghĩa app bị ghi đè khi Save, nên developer không thể thử thay đổi ở staging mà giữ nguyên production.

**Quyết định (cách B — định nghĩa app có phiên bản):**

1. **Environment cố định:** mọi application có đúng hai environment `staging` và `production`; không khai báo environment ở UC-01.
2. **Application Definition có phiên bản bất biến:** mỗi lần Save ở UC-01 tạo một phiên bản mới; phiên bản cũ không bao giờ bị sửa hay xóa.
3. **Deploy chọn phiên bản:** UC-03 deploy một phiên bản cụ thể vào một environment. Thử ở staging xong thì deploy cùng phiên bản đó lên production (promote).
4. **Xóa thành phần** nghĩa là phiên bản mới không còn thành phần đó. Lịch sử deploy trỏ tới phiên bản đã dùng nên vẫn hiển thị đúng; không cần snapshot riêng.
5. **Kiểm tra khi Save ở UC-01** chỉ trong nội bộ phiên bản: dependency trỏ tới thành phần có thật, không tạo vòng.
6. **Kiểm tra khi deploy phiên bản X vào environment Y (UC-03):** cấu hình của Y phải khớp với X; biến còn tham chiếu output của thành phần không có trong X → A1.
7. **Gỡ pod và hạ tầng thật** khi một environment deploy phiên bản không còn thành phần đó: plan hiển thị `REMOVE` (workload) hoặc `DESTROY` (resource, kèm cảnh báo mất dữ liệu) và developer xác nhận. Resource dùng chung (vấn đề 3) chỉ gỡ liên kết, không hủy hạ tầng. Giữa lúc Save và lần deploy đó, mọi thứ đang chạy giữ nguyên.
8. **Đổi sang phiên bản mới phải deploy toàn bộ app;** deploy một phần (vấn đề 2) chỉ dùng phiên bản đang chạy ở environment đó, ví dụ đổi image của một workload. *(Đề xuất đi kèm, ghi nhận cùng cách B.)*
9. **Cấu hình ở UC-02 chưa có phiên bản:** vẫn quản lý theo environment như hiện tại và được kiểm tra khớp với phiên bản khi deploy. *(Đề xuất đi kèm, ghi nhận cùng cách B.)*

**Hệ quả cần xử lý khi sửa tài liệu:**

- Workload và Resource Requirement cần **identity logic ổn định qua các phiên bản**, để Resource Instance và Workload Instance (vấn đề 2, 3) không bị tạo mới mỗi khi có phiên bản mới.
- Deployment ghi phiên bản Application Definition đã dùng.
- Contract 1 bỏ quy định xóa thành phần bị loại, thay bằng tạo phiên bản mới.
- Plan của UC-03 thêm hành động `REMOVE`/`DESTROY` (liên quan vấn đề 6).
- UC-01 A1 chỉ còn lỗi nội bộ phiên bản; UC-03 A1 thêm lỗi cấu hình environment không khớp phiên bản.
- Ví dụ environment `dev staging production` trong UC-02 đổi thành `staging`, `production`.
- Liên quan vấn đề 9: environment trở thành tập giá trị cố định.

**Sẽ ảnh hưởng:**

- [`docs/use-cases/`](../use-cases/README.md):
  - Đặc tả UC-01: Save tạo phiên bản mới; A1 chỉ còn lỗi nội bộ phiên bản; quy tắc nghiệp vụ về phiên bản.
  - Đặc tả UC-02: environment cố định `staging`, `production`.
  - Đặc tả UC-03: chọn phiên bản để deploy, promote từ staging lên production; plan có hành động gỡ/hủy; A1 thêm lỗi cấu hình environment không khớp phiên bản; đổi phiên bản phải deploy toàn bộ.
  - Đặc tả UC-04: hiển thị phiên bản định nghĩa mà deployment đã dùng.
  - Bước 1: trách nhiệm của UC-01 (lưu phiên bản) và UC-03 (deploy một phiên bản vào một environment, gỡ thành phần không còn trong phiên bản).
  - Bước 2: `saveApplicationDefinition()` (tạo phiên bản), `createDeployment()` (chọn phiên bản), `validateDeploymentInput()` (cấu hình khớp phiên bản), `planInfrastructureChanges()`/`reconcileInfrastructure()` (gỡ/hủy).
  - Bước 3: Application Repository (lưu và đọc theo phiên bản), Infrastructure Reconciler (gỡ/hủy).
- Các artifact khác: sequence UC-01, UC-03; VOPC; domain model (Application Definition có phiên bản, Deployment trỏ phiên bản); ERD (bảng phiên bản, identity logic); operation contracts 1, 3, 4, 5; state machine Deployment (nếu cần); traceability.

**Quyết định bổ sung (chốt khi lập plan cho lượt sửa chung):**

10. **ID cố định qua phiên bản:** Workload, Resource Requirement, Environment Variable Definition và Secret Definition có ID logic không đổi qua các phiên bản; đổi tên vẫn giữ ID. Cấu hình UC-02, Resource Instance, Workload Instance và Workload Deployment tham chiếu ID logic này.
11. **Lưu phiên bản bằng bảng phiên bản + dòng con:** bảng `application_definition_version` cùng các dòng `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` theo từng phiên bản; ID logic nằm ở identity table `application_component`; giữ khóa ngoại đầy đủ (không lưu phiên bản dạng khối JSON).

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (đặc tả UC-01 đến UC-04, Bước 1–3), sequence UC-01 và UC-03, VOPC (Application Repository theo phiên bản), domain model (Application Definition Version, Deployment trỏ phiên bản), ERD (`application_definition_version`, `application_component`, bảng theo phiên bản, `environment` ENUM, `removed_components`), contracts 1–5 và 8–9 (phiên bản, gỡ workload), state machines (`DESTROYED`, `UNLINKED`, `REMOVED`), traceability.
