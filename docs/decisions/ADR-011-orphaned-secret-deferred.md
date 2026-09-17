---
id: ADR-011
artifact: architecture-decision-record
status: current
outcome: deferred
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/design-decisions-log.md
---

# ADR-011 — Xử lý Secret bị orphan khi lưu lỗi

**Current interpretation:** the decision was to defer this concern. Current work is tracked in [D07](../backlog/D07-orphaned-secret.md).

**Vấn đề:** trong sequence UC-02, secret được ghi vào Secret Store ngay lúc Developer nhập (`storeSecret`), còn secret reference chỉ được lưu vào DB khi bấm Save. Mọi trường hợp không đi tới được bước lưu DB đều để lại secret không ai trỏ tới: Developer đóng tab không Save, validate thất bại rồi bỏ đi, lưu DB lỗi, nhập lại secret nhiều lần, hoặc đổi secret ở lần cấu hình sau (bản cũ vẫn còn). Hậu quả là secret thật tồn tại mà không ai quản lý hay thu hồi (rủi ro bảo mật), rác tích tụ theo thời gian, và contract 3 không nói gì về việc dọn dẹp.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem [D07](../backlog/D07-orphaned-secret.md) — gồm bảng các tình huống sinh secret orphan, ba hướng có thể cân nhắc (ghi lúc Save kèm xóa bù; lưu tạm có hạn dùng; dọn rác định kỳ) và câu hỏi cần chốt.

**Lưu ý:** vấn đề này gắn với D1 (bản nháp — secret nằm ở đâu trước khi Save); nên cân nhắc giải quyết cùng lúc. Khi sửa tài liệu cho các vấn đề đã chốt, giữ nguyên luồng lưu secret hiện tại của UC-02.

**Đã áp dụng:** `deferred_issues.md` (D7). Chưa sửa file thiết kế nào.
