---
id: UC-06-CONTEXT
artifact: use-case-context
status: current
last_reviewed: 2026-09-18
---

# UC-06 context — Đăng nhập bằng tài khoản nội bộ

## Trạng thái bàn giao

Đã triển khai và có kiểm thử tự động. UC-06 tạo lớp xác thực tối thiểu trước UC-02:
tài khoản nội bộ do Platform Operator cấp qua CLI, đăng nhập/đăng xuất trên web,
session phía server và middleware bảo vệ UI/API. OIDC, SSO, MFA, đăng ký công
khai, khôi phục mật khẩu qua email và phân quyền chi tiết chưa thuộc phạm vi.

## Đọc theo thứ tự

1. [Đặc tả](specification.md)
2. [Use Case Realization](realization.md)
3. [Thiết kế giao diện đăng nhập](ui/README.md)
4. [Sequence diagram](sequence.puml)
5. [VOPC](vopc.puml)

## Artifact dùng chung

- [ADR-019 — Xác thực bằng tài khoản nội bộ sau một boundary trung lập](../../decisions/ADR-019-local-authentication-boundary.md)
- [Kiến trúc xác thực và secret](../../architecture/security-and-secrets.md)
- [Domain objects](../../architecture/domain/domain-objects.md)
- [Database schema](../../architecture/database/schema.md)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Auth Session state machine](../../architecture/state-machines/auth-session.puml)

## Vấn đề hoãn

- [D15 — Cô lập browser draft giữa các local user](../../backlog/D15-authenticated-draft-isolation.md)

## Ranh giới triển khai

Backend hiện có migration cho account/credential/session/login-attempt,
Argon2id password service, repository PostgreSQL, middleware bảo vệ route,
login/logout API và CLI quản trị local user. Frontend React hiện có route
`/ui/login`, shared CSRF-aware API transport và action đăng xuất trong shell.
Middleware gắn `Principal` trung lập vào request; các use case nghiệp vụ không
đọc mật khẩu, cookie hoặc phụ thuộc trực tiếp vào cơ chế local/OIDC.

Kiểm thử unit/component bao phủ password policy, session, rate limit, CSRF,
route policy, login states và shared frontend transport. Kiểm thử integration
PostgreSQL bao phủ create/sign-in/reset/disable và session revocation. Chưa có
kiểm chứng real-browser hoặc môi trường triển khai; xem
[bằng chứng UC-06](../../verification/2026-09-18-uc06-local-authentication.md).
