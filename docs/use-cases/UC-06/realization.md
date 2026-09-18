---
id: UC-06-REALIZATION
artifact: use-case-realization
status: current
use_case: UC-06
last_reviewed: 2026-09-18
---

# UC-06 — Đăng nhập bằng tài khoản nội bộ: Use Case Realization

## Trách nhiệm

UC-06 xác thực local user, quản lý server-side session và tạo `Principal` cho
các request nghiệp vụ. Authentication Middleware là boundary dùng chung trước
các handler UC-01 đến UC-05; handler nghiệp vụ không đọc cookie hoặc password.
CLI quản lý vòng đời tối thiểu của local account mà không tạo public user
management API.

## System operations

- **signIn()** — nhận username, password và đường dẫn quay lại; rate-limit,
  kiểm tra account/Argon2id password, tạo Auth Session và cookie.
- **authenticateRequest()** — hash token từ cookie, kiểm tra session/account,
  cập nhật hoạt động và gắn `Principal` vào request hoặc trả `401`/redirect.
- **signOut()** — kiểm tra CSRF, thu hồi session hiện tại và xóa cookie.
- **createLocalUser()** — CLI validate username/password, tạo Local User Account
  và Local Credential trong một transaction.
- **resetLocalPassword()** — CLI thay hash và thu hồi toàn bộ session của user
  trong một transaction.
- **setLocalUserStatus()** — CLI chuyển `ACTIVE`/`DISABLED`; khi disable, thu
  hồi toàn bộ session trong cùng transaction.

## Thành phần tham gia

### Boundary/UI

- **Authentication Web UI** — feature React gồm login page, trạng thái submit,
  lỗi chung, thông báo session hết hạn và action logout trong authenticated
  shell; không lưu password hay session token.
- **Authentication API / Controller** — nhận login/logout, validate request
  shape, cookie/CSRF và ánh xạ kết quả thành redirect hoặc response HTTP.
- **Local User CLI** — prompt password ẩn, gọi cùng Authentication Service để
  create/reset/enable/disable; không nhận password qua argument hoặc biến môi
  trường.

Authentication Web UI nằm tại `idp/frontend/src/features/authentication/`, dùng
cùng React bundle, router và Primer tokens với UC-01. Backend phục vụ entry point
cho `/ui/login`; login page lấy pre-auth context và submit credential qua JSON
API cùng origin.

### HTTP contract

| Method và path | Public/protected | Hành vi |
|---|---|---|
| `GET /ui/login` | Public | Phục vụ React entry point. Login page gọi login-context; session đang hợp lệ được điều hướng tới safe `return_to` hoặc `/ui/applications`. |
| `GET /api/auth/login-context` | Public | Cấp pre-auth CSRF nonce ngắn hạn và trả safe `returnTo`; nếu request đã có session hợp lệ thì trả `redirectTo` thay vì credential form context. |
| `POST /api/auth/login` | Public, pre-auth CSRF bắt buộc | Nhận JSON username/password/returnTo và CSRF header. Thành công đặt session/CSRF cookie rồi trả `200 {redirectTo}`; credential sai trả problem JSON `401`; rate limit trả `429` cùng `Retry-After`. |
| `POST /api/auth/logout` | Protected, session CSRF bắt buộc | Revoke session hiện tại, xóa cookie và trả `204`. React/Go shell điều hướng full-page tới `/ui/login?reason=logged-out`. Không cung cấp logout bằng `GET`. |
| Mọi protected browser route | Protected | Thiếu session trả `303 /ui/login?return_to=...`. |
| Mọi protected `/api/` route | Protected | Thiếu session trả JSON `401`; không redirect sang HTML. |

React giữ password chỉ trong controlled form state tới khi request kết thúc rồi
xóa nó. Response không trả session token; sau thành công UI dùng
`window.location.assign(redirectTo)` để tạo full-page navigation.

### Frontend integration contract

- Router thêm route public `/ui/login`; các route `/ui/applications...` vẫn
  protected. `/ui/assets/...` là public để login page tải được bundle.
- Shared HTTP transport đọc session CSRF cookie và gửi `X-CSRF-Token` cho mọi
  `POST`, `PUT`, `PATCH`, `DELETE`. Login dùng nonce từ login-context thay cho
  session CSRF.
- Khi protected API trả `401`, transport điều hướng full-page tới
  `/ui/login?return_to=<current internal path>`; nó không biến `401` thành lỗi
  nghiệp vụ của UC-01.
- `403` từ CSRF được trình bày như session/form không còn hợp lệ và không tự
  retry request thay đổi trạng thái.
- Authenticated React shell và shell của Go pages gọi cùng
  `POST /api/auth/logout`. Vite development tiếp tục proxy prefix `/api`, nên
  không tạo contract dev-only.

### Application/security services

- **Authentication Service** — điều phối sign-in và quản lý account; luôn trả
  lỗi đăng nhập chung cho username không tồn tại, password sai hoặc account
  disabled.
- **Authentication Middleware** — bảo vệ page/API, gọi Session Manager và đặt
  `Principal` vào request context.
- **Password Hasher** — tạo/verify Argon2id encoded hash và cho biết khi tham số
  cũ cần rehash.
- **Session Manager** — sinh token/CSRF ngẫu nhiên, hash token, tạo/thu hồi và
  kiểm tra idle/absolute expiry.
- **Login Rate Limiter** — áp dụng bucket theo username chuẩn hóa và nguồn
  request; không giữ password hoặc raw session token.

### Persistence

- **User Account Repository** — lưu Local User Account và Local Credential;
  khóa username chuẩn hóa và cập nhật password/status trong transaction do
  Authentication Service điều phối cùng Auth Session Repository khi cần revoke.
- **Auth Session Repository** — tìm session bằng token hash, cập nhật
  `last_seen_at`, thu hồi một session hoặc toàn bộ session của user.
- **Login Attempt Repository** — lưu bucket rate-limit có thời hạn theo scope và
  key hash; dữ liệu hết hạn được dọn định kỳ.

## Ranh giới giao dịch

- `createLocalUser()` tạo account và credential trong một transaction.
- `signIn()` chỉ tạo session sau khi credential hợp lệ; clear bucket theo
  username và insert session được commit trước khi cookie được gửi.
- `resetLocalPassword()` cập nhật hash và thu hồi toàn bộ session trong một
  transaction.
- `setLocalUserStatus(DISABLED)` đổi trạng thái và thu hồi toàn bộ session trong
  một transaction.
- `signOut()` idempotent đối với session đã hết hạn/đã thu hồi và luôn xóa
  cookie ở response.

## Artifact thiết kế chi tiết

- [Thiết kế UI](ui/README.md)
- [Sequence diagram](sequence.puml)
- [VOPC](vopc.puml)
- [Operation contracts](../../architecture/contracts/operation-contracts.md)
- [Traceability matrix](../../traceability/matrix.md)
