---
id: ADR-013
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/design-decisions-log.md"
---

# ADR-013 — Delivery Repository riêng cho từng application

**Current status (2026-09-17):** accepted and applied to the current design and implementation. Statements below that code was “chưa sửa” describe the historical decision point.

**Vấn đề:** thiết kế chỉ nói IDP "publish desired deployment state tới CD abstraction", không nói desired state nằm ở đâu. Bản cài đặt đặt mọi application vào **một** repo Git dùng chung, phân tách bằng thư mục `<nơi triển khai>/<app>-<environment>/workloads/…`, và mọi cụm dùng chung một cặp khóa. Hệ quả:

- Khóa ghi của IDP và khóa đọc nạp vào Argo CD của một cụm mở được desired state của **mọi** application, kể cả app không deploy lên cụm đó.
- Lịch sử commit của các application trộn lẫn; không phân quyền hay audit theo application được.

**Quyết định:**

1. **Mỗi application có một Delivery Repository riêng** — nơi chứa desired state của application đó. Trong repo vẫn chia theo nơi triển khai và environment như cũ (`<target>/<app>-<env>/workloads/…`), vì một application dùng chung một repo cho mọi environment.
2. **IDP tự tạo repo ở lần deploy đầu tiên của application.** Platform không phải làm thủ công cho từng app. Tên repo suy ra từ mẫu platform cấu hình, ví dụ `<tổ chức>/idp-<app>-gitops`. Repo đã tồn tại đúng tên thì dùng lại, không báo lỗi.
3. **IDP sinh một cặp khóa riêng cho từng application** khi tạo repo: khóa ghi để IDP đẩy manifest, khóa đọc nạp vào Argo CD của cụm dưới dạng thông tin truy cập của riêng repo đó. Nhờ vậy khóa của application này không mở được repo của application khác. Khóa lưu trong Secret Store; database, log và Deployment Record chỉ thấy reference.
4. **Thông tin gọi hệ thống lưu trữ Git** (token của tài khoản máy, quyền tạo repo và gắn deploy key) nằm trong Secret Store, chỉ Deployment Worker đọc.
5. **Việc đăng ký application → repo được lưu bền vững** (URL, nhánh). IDP ghi khi tự tạo; platform cũng đăng ký tay được một repo có sẵn.
6. **Gỡ application khỏi một environment/nơi triển khai** chỉ xóa phần desired state của environment đó trong repo. Repo và khóa giữ lại để còn lịch sử.
7. **Tạo repo, sinh khóa hoặc đăng ký thất bại** → deployment FAILED ở bước giao hàng (A2), không publish sang repo nào khác.

**Hệ quả cần xử lý khi sửa tài liệu:**

- Thêm domain object **Delivery Repository** (thuộc Application) và bảng tương ứng; thêm thành phần lưu trữ để đọc/ghi đăng ký.
- Thêm abstraction cho hệ thống lưu trữ Git (tạo repo, gắn khóa) và implementation cụ thể; UC-03 vẫn không phụ thuộc vào một sản phẩm cụ thể.
- Contract 8 mô tả việc bảo đảm repo của application tồn tại trước khi publish; A2 thêm trường hợp tạo repo thất bại.
- Đặc tả UC-03 thêm quy tắc: desired state của mỗi application nằm ở repo riêng, application này không ghi được vào repo của application khác.

**Sẽ ảnh hưởng:** [đặc tả và realization UC-03](../use-cases/UC-03/README.md), sequence UC-03, VOPC UC-03 + design class diagram + README, domain model + domain objects + persistence classification, ERD, operation contract 8, traceability.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** toàn bộ danh sách trên. Code trong `idp/backend/` chưa sửa theo quyết định này.
