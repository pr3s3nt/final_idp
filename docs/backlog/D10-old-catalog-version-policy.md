---
id: D10
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-18
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D10 — Chính sách với Catalog Version cũ

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này; mọi phiên bản catalog đều được chọn khi deploy.

### Vấn đề

Theo vấn đề 12, catalog có phiên bản bất biến và Developer chọn phiên bản khi deploy, kể cả phiên bản cũ hơn phiên bản đang chạy. Platform chưa có cách:

- Cấm dùng một phiên bản có lỗi hoặc lỗ hổng (vd module Terraform tạo cụm với cấu hình không an toàn).
- Buộc application chuyển khỏi phiên bản cũ trước một thời hạn.
- Biết application/environment nào còn chạy trên phiên bản nào để thông báo.

### Câu hỏi cần chốt khi giải quyết

1. Phiên bản catalog có trạng thái (vd `ACTIVE`, `DEPRECATED`, `BLOCKED`) không, và ai đổi trạng thái (platform administration nằm ngoài các use case hiện tại)?
2. Deployment đang chạy trên phiên bản bị khóa thì deploy một phần có bị chặn không, hay chỉ chặn chọn phiên bản đó cho deployment mới?
3. Có cần màn hình cho platform xem application nào đang dùng phiên bản nào không?

### Điều kiện đóng

ERD và domain model thể hiện trạng thái phiên bản catalog (nếu có); UC-03 A1 và contract 4 mô tả hành vi khi chọn hoặc đang chạy phiên bản bị khóa.
