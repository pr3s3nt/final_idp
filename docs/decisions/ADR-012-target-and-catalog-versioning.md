---
id: ADR-012
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../../06_traceability/design_decisions.md
---

# ADR-012 — Deployment target, hạ tầng và Catalog Version

**Current status (2026-09-17):** accepted and applied to the current design and implementation. Statements below that code was “chưa sửa” describe the historical decision point. Remaining policy questions are tracked in [D10](../backlog/D10-old-catalog-version-policy.md), [D11](../backlog/D11-catalog-formula-change.md), and [D12](../backlog/D12-uc02-catalog-version.md).

**Vấn đề:** ở UC-03, Developer chọn deployment target và IDP "reconcile infrastructure". Nhưng thiết kế chưa nói có những loại nơi triển khai nào, cụm Kubernetes lấy từ đâu, và VPC mà cụm cần được dựng ở bước nào.

Ví dụ: deploy `shop-app` (frontend, backend, PostgreSQL) lên AWS. IDP phải dựng VPC, rồi EKS và Aurora trên VPC, rồi mới chạy backend và frontend trên EKS. Khi làm theo hướng này, thiết kế còn thiếu 4 chỗ:

1. Đồ thị deploy chỉ có thứ Developer khai báo (frontend, backend, postgresql). Không có chỗ cho cụm, VPC, và quan hệ "Aurora cần VPC".
2. Phạm vi deploy chỉ gồm workload được chọn và resource chúng dùng trực tiếp. Không workload nào dùng trực tiếp VPC, nên VPC không bao giờ được tạo.
3. Khi output thay đổi, IDP chỉ deploy lại workload. VPC đổi subnet thì EKS và Aurora không được cập nhật theo. Cụm bị dựng lại thì app không được đưa lên cụm mới.
4. Platform sửa catalog (vd đổi cấu hình VPC) thì lần deploy sau của mọi app đều nhận thay đổi đó, kể cả khi Developer chỉ deploy lại frontend.

**Quyết định:**

1. **Có hai loại nơi triển khai**; Developer chọn một khi deploy.
   - **Cloud** (vd AWS, region ap-southeast-1): IDP dựng VPC và cụm (vd EKS) riêng cho mỗi app + environment, và xóa chúng khi gỡ app.
   - **Cụm Kubernetes nội bộ:** cụm đã có sẵn. IDP chỉ kết nối vào để deploy, không tạo, không xóa cụm. Platform có thể khai báo nhiều cụm nội bộ (vd một cho staging, một cho production); catalog quyết định app và environment nào dùng cụm nào.
2. **Cụm và VPC nằm trong đồ thị deploy** như resource khác; Developer không khai báo chúng. IDP tự thêm:
   - App nào cũng cần một cụm (`k8s-cluster`).
   - Công thức trong catalog ghi thêm nó cần gì (`requires`), vd EKS và Aurora cần `network`. Công thức cần gì là do platform team quyết định; IDP làm theo.
   - Trên cloud, công thức `k8s-cluster` là `MANAGED`: IDP tạo, sửa, xóa.
   - Trên cụm nội bộ, công thức `k8s-cluster` là `EXISTING`: trỏ tới cụm có sẵn, IDP chỉ liên kết.
3. **Database và các resource khác:** IDP vẫn tạo cho mỗi app + environment, kể cả trên cụm nội bộ (vd Postgres chạy trong cụm). Muốn dùng database có sẵn thì platform khai báo công thức `EXISTING`.
4. **Phạm vi deploy** = workload được chọn + resource chúng dùng trực tiếp + cụm/VPC mà các resource đó cần. Thứ gì không đổi thì dùng lại, không chạy lại Terraform.
5. **Dưới đổi thì trên làm lại:** output của một thành phần đổi thì mọi resource và workload dựa trên nó được làm lại trong cùng lần deploy, kể cả khi nằm ngoài phạm vi. Plan hiện trước những thứ có thể bị làm lại.
6. **Catalog có phiên bản.** Sửa catalog là tạo phiên bản mới; phiên bản cũ giữ nguyên. Developer chọn phiên bản catalog khi deploy, nên app đang chạy không bị ảnh hưởng cho tới khi Developer chọn phiên bản mới.
   - Deploy một phần phải dùng phiên bản catalog đang chạy. Muốn đổi phiên bản thì phải deploy toàn bộ app.
   - Được chọn phiên bản cũ hơn; plan vẫn kiểm tra như bình thường.
   - Form chọn sẵn phiên bản đang chạy và báo nếu có bản mới hơn. Lần deploy đầu thì chọn sẵn bản mới nhất.
   - Promote staging → production dùng cả phiên bản app lẫn phiên bản catalog của staging.

**Hoãn** (ghi vào `deferred_issues.md`):

- **D10:** platform khóa hoặc ngừng hỗ trợ phiên bản catalog cũ.
- **D11:** phiên bản catalog mới đổi hẳn công thức của một resource đang chạy (vd Postgres trong cụm → database dùng chung): báo lỗi hay tự dựng lại.
- **D12:** UC-02 lấy danh sách output từ phiên bản catalog nào, khi cấu hình không gắn với phiên bản catalog.

**Sẽ sửa:** use case realization (đặc tả UC-03, UC-04, Bước 1–3), sequence UC-03 và UC-04, VOPC, domain model, ERD, operation contracts 4–6 và 11, state machine Resource Instance, traceability. Contract 3 (UC-02) chưa sửa vì phụ thuộc D12.

**Đã áp dụng (nhánh `uc03-impl`, 15/09/2026):** use case realization (đặc tả UC-03, UC-04; Bước 1–3 UC-03), sequence UC-03 và UC-04, VOPC (`vopc_uc03`, design class diagram, README), domain model, domain objects, persistence classification, ERD (`catalog_version`, `resource_definition`, `application_component`, `resource_instance`, `deployment`), contracts 4, 5, 6, 11, state machine Resource Instance và ghi chú Deployment, traceability; phần hoãn ghi vào D10, D11, D12. Code trong `uc03/` chưa sửa theo quyết định này.
