---
id: PROJECT-TASK-BACKLOG
artifact: project-task-backlog
status: current
last_reviewed: 2026-09-17
---

# Công việc cần xử lý sau

File này là nơi ghi nhanh các công việc của dự án chưa thể thực hiện ngay. Đây là danh sách quản lý công việc, không phải nguồn định nghĩa yêu cầu hay thiết kế.

Nếu một công việc là vấn đề thiết kế chưa được giải quyết, hãy tạo một mục `Dxx` riêng trong [backlog vấn đề thiết kế](README.md) rồi đặt liên kết vào bảng dưới đây. Khi bắt đầu một đợt triển khai, chuyển công việc phù hợp sang [kế hoạch iteration](../iterations/README.md).

## Cách sử dụng

- Thêm một dòng mới với ID tăng dần theo dạng `T001`, `T002`, ...
- Dùng một trong các trạng thái: `Chưa làm`, `Đang làm`, `Bị chặn`, `Hoàn thành`, `Hủy`.
- Ghi liên kết tới use case, backlog item, issue, tài liệu hoặc mã nguồn liên quan trong cột **Liên quan**.
- Khi công việc hoàn thành hoặc bị hủy, ghi ngày và kết quả ngắn gọn trong cột **Ghi chú**; định kỳ chuyển các dòng cũ xuống mục **Đã đóng**.

## Công việc đang mở

| ID | Công việc | Ưu tiên | Trạng thái | Liên quan | Ngày thêm | Ghi chú |
|---|---|---|---|---|---|---|
| [T001](#t001--rà-soát-tính-đúng-đắn-của-các-tài-liệu-nguồn-chuẩn) | Rà soát tính đúng đắn của các tài liệu nguồn chuẩn | Cao | Chưa làm | [Documentation index](../INDEX.md), [Documentation rules](../DOCUMENTATION_RULES.md), [`AGENTS.md`](../../AGENTS.md) | 2026-09-17 | Kiểm tra nội dung, ranh giới trách nhiệm và tính nhất quán giữa các tài liệu. |

### T001 — Rà soát tính đúng đắn của các tài liệu nguồn chuẩn

**Mục tiêu:** xác minh mỗi nhóm thông tin được ghi đúng trong tài liệu sở hữu chuẩn, phản ánh đúng trạng thái dự án và không mâu thuẫn với các nguồn có thẩm quyền liên quan.

**Phạm vi rà soát:**

- Hành vi bắt buộc của use case: [`docs/use-cases/UC-*/specification.md`](../use-cases/README.md).
- Cách các thành phần cộng tác để thực hiện use case: [`docs/use-cases/UC-*/realization.md`](../use-cases/README.md).
- Kiến trúc và mô hình dùng chung: [`docs/architecture/`](../architecture/README.md).
- Bảng, cột, ràng buộc và kiểu liệt kê của cơ sở dữ liệu: [`docs/architecture/database/schema.md`](../architecture/database/schema.md).
- Quyết định thiết kế đã được chấp nhận và lý do: [`docs/decisions/`](../decisions/README.md).
- Vấn đề đang mở, rủi ro hoặc nội dung bị hoãn: [`docs/backlog/`](README.md).
- Phạm vi đã triển khai và giới hạn hiện tại: [`docs/CURRENT_STATE.md`](../CURRENT_STATE.md).
- Sai lệch đã biết giữa thiết kế và mã nguồn: [`docs/implementation/deviations.md`](../implementation/deviations.md).
- Ánh xạ giữa thiết kế và mã nguồn: [`docs/implementation/code-map.md`](../implementation/code-map.md).
- Liên kết giữa yêu cầu, thiết kế và kiểm thử: [`docs/traceability/matrix.md`](../traceability/matrix.md).
- Quy trình chạy và xử lý sự cố hiện hành: [`docs/operations/uc03/RUNBOOK.md`](../operations/uc03/RUNBOOK.md).
- Kết quả của các lần kiểm chứng cụ thể: [`docs/verification/`](../verification/README.md).
- Thuật ngữ dùng chung: [`docs/GLOSSARY.md`](../GLOSSARY.md).
- Kế hoạch và kết quả của các vòng phát triển: [`docs/iterations/`](../iterations/README.md).
- Quy tắc tổ chức và chỉnh sửa tài liệu: [`docs/DOCUMENTATION_RULES.md`](../DOCUMENTATION_RULES.md).
- Quy tắc làm việc dành cho AI: [`AGENTS.md`](../../AGENTS.md).

**Điều kiện hoàn thành:**

1. Tất cả nhóm tài liệu trong phạm vi đã được đọc và đối chiếu với mã nguồn, migration, test hoặc bằng chứng kiểm chứng phù hợp.
2. Mọi thông tin sai, lỗi thời, trùng lặp hoặc đặt sai tài liệu sở hữu đã được sửa; vấn đề chưa thể chốt đã được ghi thành backlog item hoặc design deviation phù hợp.
3. Các liên kết truy vết bị ảnh hưởng đã được cập nhật và không còn mâu thuẫn đã biết nhưng chưa được ghi nhận.
4. `python3 scripts/check_docs.py` và `git diff --check` đều thành công; PlantUML được kiểm tra trong môi trường có cài đặt công cụ.

## Đã đóng

| ID | Công việc | Kết quả | Ngày đóng | Liên quan |
|---|---|---|---|---|
