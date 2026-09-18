---
id: D15
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-18
related: ADR-016, ADR-019, UC-01, UC-06
---

# D15 — Cô lập browser draft giữa các local user

## Bối cảnh

ADR-016 giao draft UC-01/UC-02 cho browser tab và lưu phần không nhạy cảm trong
`sessionStorage`. Key hiện được scope theo use case và application/environment,
nhưng chưa theo `Principal.userId`. Trước UC-06, ứng dụng chưa có khái niệm hai
user lần lượt dùng cùng một tab.

Khi UC-06 được bật, user A có thể đăng xuất hoặc hết session rồi user B đăng
nhập trong cùng tab. Nếu không có policy bổ sung, frontend có thể phục hồi draft
của user A cho user B. Draft không chứa password, token hoặc plaintext Secret,
nhưng vẫn có thể chứa thông tin application chưa được lưu.

## Quyết định hoãn

Không thay đổi schema draft hoặc key `sessionStorage` trong lát cắt triển khai
UC-06 đầu tiên. Authentication không được coi việc giữ draft qua đổi account là
một quyền truy cập mới; server vẫn validate và authorize mọi durable read/write.

## Các hướng cần đánh giá

1. Namespace key theo `Principal.userId`, với một bootstrap/session endpoint để
   frontend biết identity hiện hành.
2. Xóa mọi draft IDP khi logout hoặc khi frontend phát hiện account đã đổi.
3. Kết hợp hai cách: namespace để cô lập và cleanup để tránh draft cũ tồn tại
   không giới hạn trong tab.

## Điều kiện đóng

- Chọn rõ hành vi khi session hết hạn rồi cùng hoặc khác user đăng nhập lại.
- UC-01/UC-02 specification và ADR-016 mô tả ownership cuối cùng.
- Storage adapter và tests chứng minh draft không được phục hồi chéo account.
- Logout/session-expiry UX nói rõ draft được giữ hay xóa.
