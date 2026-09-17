---
id: D07
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D07 — Secret bị bỏ rơi trong Secret Store

**Tương ứng:** issue #11 trong commit `88585cc` ("Secret bị orphan khi save lỗi").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Bối cảnh

Để DB của IDP không bao giờ chứa secret dạng chữ thường, UC-02 làm như sau: gửi giá trị secret vào **Secret Store**, nhận về một **secret reference**, và DB chỉ lưu reference đó (`secret.secret_ref`).

### Vấn đề

Trong `docs/use-cases/UC-02/sequence.puml`, secret được ghi vào Secret Store **ngay lúc Developer nhập**, còn reference chỉ được lưu vào DB **khi bấm Save**:

```text
Developer nhập DB_PASSWORD
  → setDirectConfigurationValue(secret, secretValue)
  → storeSecret(applicationId, environment, secretValue)   ← ghi vào Secret Store ngay
  → nhận secret reference

... Developer tiếp tục điền các biến khác ...

Developer bấm Save
  → validateEnvironmentConfiguration(configuration)
  → save(configuration with value references)              ← lúc này mới lưu reference vào DB
```

Mọi trường hợp không đi tới được bước lưu DB đều để lại một secret không ai trỏ tới (orphan):

| Tình huống | Kết quả |
|---|---|
| Developer nhập secret rồi đóng tab, không bấm Save | Secret nằm lại trong Secret Store |
| Bấm Save nhưng validate thất bại rồi bỏ đi | Như trên |
| Lưu DB lỗi (ví dụ DB mất kết nối) | Như trên |
| Developer nhập lại secret nhiều lần trước khi Save | Các bản trước nằm lại |
| Đổi secret ở lần cấu hình sau | Bản cũ vẫn nằm trong Secret Store |

Hậu quả:

- **Rủi ro bảo mật:** secret thật tồn tại mà không ai biết, không ai quản lý, không ai thu hồi.
- **Rác tích tụ** theo thời gian, có thể tốn chi phí (nhiều dịch vụ secret tính tiền theo số secret).
- Contract 3 (`saveEnvironmentConfiguration`) chỉ bảo đảm plaintext không vào DB, không nói gì về việc dọn các secret không được dùng.

### Liên quan

- **D1:** câu hỏi "secret nằm ở đâu trong lúc Developer chưa bấm Save" là một phần của câu hỏi bản nháp được giữ ở đâu.
- **Vấn đề 5 (đã chốt):** cấu hình UC-02 chưa có phiên bản; khi đổi secret, bản cũ không còn được cấu hình nào trỏ tới.

### Hướng có thể cân nhắc (chưa chốt)

**A. Chỉ ghi vào Secret Store lúc bấm Save:** validate xong mới ghi secret, rồi lưu reference vào DB; lưu DB lỗi thì xóa ngay secret vừa ghi; lưu thành công mà secret bị thay thì xóa bản cũ. Đơn giản, loại được gần hết các tình huống; nhưng secret nằm trong form tới lúc Save (dính D1), và nếu server chết đúng giữa "ghi Secret Store" và "lưu DB" thì vẫn có thể sót.

**B. Lưu tạm có hạn dùng** (hướng của `88585cc`): lúc nhập, secret vào Secret Store dưới dạng tạm, tự hết hạn; Save thành công thì chuyển thành chính thức; không Save thì tự biến mất. Secret rời trình duyệt ngay, không có rác lâu dài; nhưng phức tạp hơn (tạm/chính thức/thu hồi/hết hạn) và có kẽ hở khi DB đã lưu mà chưa kịp chuyển thành chính thức thì bản tạm hết hạn.

**C. Tác vụ dọn rác định kỳ:** định kỳ so danh sách secret trong Secret Store với reference trong DB, xóa cái không ai trỏ tới. Bắt được mọi trường hợp sót, kể cả server chết giữa chừng; nhưng giữa hai lần dọn, secret orphan vẫn tồn tại.

Đề xuất lúc bàn: A, kèm C làm lưới an toàn.

### Câu hỏi cần chốt khi giải quyết

1. Chọn hướng nào (A, B, C hoặc kết hợp)?
2. Khi đổi secret: xóa bản cũ ngay sau khi lưu thành công, hay giữ lại một thời gian để có thể quay lại?
3. Có giải quyết cùng lúc với D1 (bản nháp) không?

### Điều kiện đóng

Sequence UC-02, contract 3 và VOPC (Secret Store) thể hiện rõ thời điểm ghi secret, cách xử lý khi validate/lưu DB thất bại và khi secret bị thay; không tình huống nào trong bảng trên để lại secret không ai quản lý mà không có cơ chế dọn.
