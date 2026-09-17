---
id: D11
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: ../../06_traceability/deferred_issues.md
---

# D11 — Catalog mới đổi công thức của resource đang chạy

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này.

### Vấn đề

Ví dụ: shop-app staging chạy trên cụm nội bộ với catalog v1, PostgreSQL dựng bằng công thức `postgres-k8s` (pod Postgres trong cụm, dữ liệu nằm trong đó). Platform ra catalog v2, trong đó PostgreSQL trên cụm nội bộ đổi sang công thức khác, vd `postgres-shared` (database dùng chung có sẵn). Developer chọn catalog v2 và deploy.

Nếu v2 chỉ đổi tham số trong cùng công thức (vd dung lượng 10GB → 20GB) thì IDP cập nhật resource bình thường; mục này chỉ nói trường hợp đổi sang công thức khác.

### Câu hỏi cần chốt khi giải quyết

1. IDP báo lỗi và không cho deploy (Developer/platform tự chuyển dữ liệu hoặc gỡ app trước), hay tự hủy resource cũ rồi dựng lại theo công thức mới sau khi Developer xác nhận cảnh báo mất dữ liệu?
2. Nếu tự dựng lại: thứ tự hủy/tạo thế nào để workload đang dùng resource không bị gián đoạn lâu?

### Điều kiện đóng

Đặc tả UC-03 (A1 hoặc plan), contract 4 và contract 6 mô tả rõ hành vi khi công thức của một resource đang chạy thay đổi giữa hai phiên bản catalog.
