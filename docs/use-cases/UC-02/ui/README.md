---
id: UC-02-UI-DESIGN
artifact: user-interface-design
status: current
last_reviewed: 2026-09-18
related: UC-02, ADR-016, ADR-017, ADR-018, ADR-020, D07, D15
---

# UC-02 — Thiết kế giao diện Configuration

## Mục đích và ranh giới

Tài liệu này định nghĩa thiết kế màn hình và tương tác cho trang Configuration
của UC-02. [Đặc tả UC-02](../specification.md) vẫn là tài liệu sở hữu hành vi
sản phẩm bắt buộc; tài liệu này chuyển hành vi đó thành hợp đồng giao diện mà
con người và AI agent có thể review.

Trang Configuration trả lời đúng một câu hỏi: **mỗi Environment Variable và
Secret mà UC-01 đã khai báo sẽ lấy giá trị từ đâu, trong environment này.** Nó
không tạo, sửa hay xóa workload và resource — việc đó thuộc UC-01. Nó không
deploy, không hiển thị trạng thái runtime, không hiện giá trị thật của
Resource Output; giá trị thật chỉ được resolve ở UC-03.

UC-02 chỉ nhắm tới trình duyệt web trên desktop, giống UC-01. Mobile, React
Native và layout riêng cho điện thoại/máy tính bảng nằm ngoài phạm vi.

Trang là feature React trong frontend hiện tại, tại route
`/ui/applications/{applicationId}/configuration`.

## Nguyên tắc thiết kế

1. Tổ chức theo workload, giống [UI design của UC-01](../../UC-01/ui/README.md).
   Developer nghĩ theo "backend cần gì", không nghĩ theo một danh sách biến
   phẳng.
2. Với mỗi requirement, chỉ hỏi đúng một câu: *giá trị này đến từ đâu?* Các
   control tiếp theo chỉ xuất hiện sau khi đã trả lời câu đó.
3. Chỉ cho chọn những thứ hợp lệ. Danh sách resource và workload đã được lọc
   theo dependency khai báo ở UC-01; danh sách output đã được lọc theo Resource
   Definition. Developer không phải tự nhớ luật.
4. Không bao giờ hiển thị lại giá trị Secret. Sau khi lưu, Secret chỉ còn là
   một tham chiếu.
5. Giữ nguyên draft do browser sở hữu và cơ chế Save toàn bộ draft với hai
   concurrency base theo [ADR-016](../../../decisions/ADR-016-client-owned-drafts.md).
6. Ưu tiên control gốc của trình duyệt, label cố định và điều hướng bằng bàn
   phím.

## Kiến trúc thông tin

Trang có bốn vùng cố định trên desktop:

- **Header** — link quay lại application, tên application, trạng thái draft,
  `Discard` và `Save configuration`.
- **Thanh lựa chọn phạm vi** — `Environment`, `Platform catalog version`,
  `Deployment target`. Ba lựa chọn này quyết định nội dung phía dưới, nên đặt
  trên cùng, ngay dưới header.
- **Navigation** — một mục cho mỗi workload có requirement, và mục `Review`.
  Mỗi mục hiển thị số vấn đề đang chặn Save của workload đó.
- **Workspace** — các requirement của workload đang chọn, hoặc bảng tổng hợp
  `Review`.

Workload không có Environment Variable lẫn Secret thì không xuất hiện trong
navigation: không có gì để cấu hình.

### Ba lựa chọn phạm vi khác nhau thế nào

| Lựa chọn | Ảnh hưởng | Có được lưu không |
|---|---|---|
| `Environment` | Chọn bản ghi Environment Configuration đang sửa. Đổi nó là mở một cấu hình khác. | Có — là khóa của configuration |
| `Platform catalog version` | Quyết định tập Resource Definition dùng để liệt kê output | Không ([ADR-020](../../../decisions/ADR-020-uc02-catalog-version-and-target.md)) |
| `Deployment target` | Cùng với catalog version, chọn ra đúng một Resource Definition cho mỗi resource | Không (ADR-020) |

Giao diện phải nói rõ sự khác nhau đó. Dưới thanh lựa chọn có một dòng giải
thích: phiên bản catalog và deployment target chỉ dùng để biết resource có
những output nào, không được lưu cùng configuration, và UC-03 sẽ kiểm tra lại
khi deploy.

Đổi `Environment` là đổi hẳn cấu hình đang sửa, nên khi draft có thay đổi chưa
lưu, UI phải hỏi xác nhận trước.

Đổi `Platform catalog version` hoặc `Deployment target` có thể làm một output
đang được tham chiếu không còn tồn tại. Khi đó UI xóa phần resource và output
của các binding dạng Resource Output, giữ nguyên nguồn giá trị đã chọn, và báo
cho Developer biết những binding nào cần chọn lại.

## Sơ đồ giao diện web desktop

- [Workspace cấu hình một workload](configuration.puml) định nghĩa header,
  thanh lựa chọn phạm vi, navigation và workspace.
- [Workspace Review](review.puml) định nghĩa bảng tổng hợp và validation
  summary.
- [Trạng thái trang và phục hồi](states.puml) định nghĩa các chuyển tiếp khi
  load, chỉnh draft, staging Secret, Save, lỗi và conflict.

## Quy tắc tương tác

### Chọn nguồn giá trị

Mỗi requirement là một khối gồm tên, dấu bắt buộc, và một select `Source`.

Environment Variable có ba nguồn: `Environment value`, `Resource output`,
`Workload output`. Secret có hai: `Secret value`, `Resource output`.

Sau khi chọn nguồn, khối hiển thị đúng control cần cho nguồn đó:

| Nguồn | Control tiếp theo |
|---|---|
| `Environment value` | Một ô nhập text. Ví dụ `LOG_LEVEL = INFO` |
| `Resource output` | Select resource, rồi select output. Ví dụ `DB_HOST ← postgresql.host` |
| `Workload output` | Select workload, rồi select output. Ví dụ `BACKEND_URL ← backend.endpoint` |
| `Secret value` | Ô nhập dạng password và nút `Store secret` |

Đổi nguồn sẽ xóa dữ liệu của nguồn cũ. Giá trị vừa nhập cho nguồn cũ không được
âm thầm giữ lại rồi gửi đi lúc Save.

Select resource và select workload chỉ chứa thành phần mà workload này depends
on trong phiên bản mới nhất. Khi không có thành phần nào, select hiển thị câu
giải thích thay vì một danh sách rỗng: *"backend không phụ thuộc vào workload
nào; khai báo dependency ở UC-01 trước."*

Danh sách output chỉ được tải khi Developer đã chọn resource hoặc workload. Khi
đang tải, select output hiển thị `Loading outputs…` và chưa cho chọn.

Environment Variable chỉ được chọn output thường. Secret chỉ được chọn output
nhạy cảm. Hai danh sách không giao nhau, nên Developer không thể gán
`password` vào một biến thường.

### Secret

Giá trị Secret không đi cùng draft. Developer nhập giá trị rồi bấm
`Store secret`; giá trị được gửi thẳng tới Secret Store và đổi lấy một tham
chiếu. Từ lúc đó UI chỉ giữ tham chiếu.

- Trước khi bấm: khối hiển thị *"Giá trị sẽ được gửi tới Secret Store và thay
  bằng một tham chiếu."*
- Sau khi bấm: ô nhập được xóa rỗng và khối hiển thị *"Đã lưu. IDP chỉ giữ một
  tham chiếu; giá trị không được hiển thị lại."*
- Nhập giá trị nhưng chưa bấm: khối hiển thị cảnh báo *"Đã nhập nhưng chưa
  lưu"*, và Save bị chặn cho tới khi Developer bấm `Store secret` hoặc xóa
  trống ô nhập.

Secret đã lưu ở lần cấu hình trước hiển thị là đã có giá trị, kèm nút
`Replace` để nhập giá trị mới. Không có nút xem lại giá trị.

Vòng đời và việc dọn dẹp Secret đã staging nhưng không được Save thuộc
[D07](../../../backlog/D07-orphaned-secret.md) và nằm ngoài thiết kế này.

### Validation

- Validation local chạy trên draft hiện tại: requirement bắt buộc chưa có
  nguồn, `Environment value` rỗng, chọn resource nhưng chưa chọn output, Secret
  đã nhập mà chưa staging.
- Badge trong navigation hiển thị số vấn đề của từng workload và của `Review`.
- `Save configuration` bị disable khi còn vấn đề local. Nút disable phải luôn
  đi kèm câu giải thích, không để Developer đoán vì sao không bấm được.
- Validation của backend là nguồn có thẩm quyền. Khi backend từ chối, UI hiển
  thị từng vấn đề trong `Review` và chuyển tới workspace tương ứng khi vấn đề
  nêu tên một workload.

### Trạng thái draft và action

- Header phân biệt `Saved`, `Unsaved changes · kept in this tab`,
  `Restored draft` và `Saving…`, giống UC-01.
- `Save configuration` và `Discard` luôn nằm trên header. Save gửi toàn bộ
  draft đúng một lần.
- `Discard` chỉ hỏi xác nhận khi có thay đổi; sau đó xóa draft trong
  `sessionStorage` và tải lại cấu hình bền vững.
- Save thành công hiển thị xác nhận, nêu rõ rằng lưu cấu hình **không** deploy
  application, và xóa draft theo ADR-016.

### Conflict khi Save

`DRAFT_CONFLICT` không được thay thế hoặc xóa draft của Developer. UI hiển thị
panel chặn luồng, giải thích rằng Application Definition hoặc cấu hình đã thay
đổi kể từ lúc tải, và cung cấp `Reload configuration` có xác nhận rõ rằng draft
hiện tại sẽ bị bỏ. IDP không tự động merge.

## Trạng thái màn hình

| Trạng thái | Cách hiển thị và phục hồi |
|---|---|
| Bắt đầu load | Trạng thái/skeleton workspace, không nháy form rỗng có thể chỉnh sửa |
| Load thất bại | Giải thích lỗi, `Try again` và link quay lại application |
| Application chưa khai báo biến/Secret nào | Giải thích không có gì để cấu hình và link tới UC-01 |
| Draft sạch | Trạng thái `Saved`; `Discard` bị disable |
| Draft có thay đổi | `Unsaved changes · kept in this tab` |
| Draft được phục hồi | Thông báo và trạng thái header `Restored draft` |
| Draft phục hồi nhưng phiên bản đã đổi | Bỏ draft cũ, dùng requirement mới nhất và báo cho Developer |
| Secret đã nhập chưa staging | Cảnh báo tại khối đó, Save bị chặn |
| Đang lưu | Disable Save với `Saving…`; nội dung hiện tại vẫn hiển thị |
| Validation thất bại | Chọn `Review`, focus summary, hiển thị badge và lỗi tại từng khối |
| Save thất bại | Lỗi không phá hủy draft; thử lại bằng Save thông thường |
| Conflict | Giữ draft; chỉ tải lại sau xác nhận |
| Save thành công | Thông báo đã lưu, nêu rõ không deploy, draft sạch |

## Ánh xạ action sang state/API

| Action của người dùng | Trạng thái client draft/session | HTTP request |
|---|---|---|
| Mở trang Configuration | Tải requirement và cấu hình hiện hành, rồi phục hồi draft của tab khi khớp phiên bản | Một `GET /api/environment-configurations/{applicationId}/{environment}` |
| Đổi `Environment` | Xác nhận nếu có thay đổi, rồi tải environment mới | Một `GET` như trên |
| Đổi catalog version hoặc target | Cập nhật reducer; xóa resource/output của binding Resource Output | Không |
| Chọn workload trong navigation | Chỉ đổi selection | Không |
| Chọn nguồn giá trị, nhập giá trị trực tiếp | Cập nhật reducer và `sessionStorage` | Không |
| Chọn resource hoặc workload để tham chiếu | Cập nhật reducer, rồi tải danh sách output | Một `GET .../resources/{id}/outputs` hoặc `GET .../workloads/{id}/outputs` |
| Bấm `Store secret` | Thay giá trị bằng tham chiếu trong draft | Một `POST .../secrets` |
| `Discard` | Xóa `sessionStorage` và tải lại | Một `GET` như trên |
| `Save configuration` | Giữ draft đến khi thành công rồi xóa | Một `PUT /api/environment-configurations/{applicationId}/{environment}` |
| `Reload configuration` sau conflict | Xác nhận, xóa draft và tải lại | Một `GET` như trên |

## Hợp đồng accessibility

- Trang có đúng một `h1` là tên application kèm chữ Configuration. Tên workload
  trong workspace là `h2`.
- Navigation là một navigation region có label. Mục đang chọn dùng
  `aria-current`; badge vấn đề có text cho screen reader.
- Mọi control có label cố định. Select `Source`, select resource/workload và
  select output đều phải có label riêng, không dùng placeholder thay label.
- Ô nhập Secret dùng `type="password"` và `autocomplete="new-password"`.
- Trạng thái staging Secret, validation summary và kết quả Save dùng
  live-region role phù hợp.
- Không truyền đạt ý nghĩa chỉ bằng màu: requirement bắt buộc có dấu `*` kèm
  text, Secret có nhãn `secret` kèm text.
- Mọi action hoạt động bằng bàn phím.

## Ngôn ngữ thị giác

Kế thừa nguyên vẹn hệ token, chữ, khoảng cách, bo góc và quy tắc áp dụng màu
của [UI design UC-01](../../UC-01/ui/README.md#ngôn-ngữ-thị-giác) theo
[ADR-018](../../../decisions/ADR-018-primer-design-tokens.md). Không định nghĩa
lại token ở đây.

Phần riêng của UC-02:

1. Dùng mono cho dữ liệu kỹ thuật: tên biến, tên Secret, tên resource, tên
   workload, tên output, giá trị trực tiếp. Không dùng mono cho câu giải thích.
2. Requirement là Secret có nhãn `secret` dùng cặp `--muted` trên
   `--surface-subtle`. Cảnh báo "đã nhập nhưng chưa lưu" dùng cặp
   `--warning-text` trên `--warning-bg`.
3. Tham chiếu output hiển thị dưới dạng `postgresql.host` để Developer đọc được
   quan hệ mà không cần mở lại select.
4. Mỗi vùng màn hình chỉ có một nút primary: `Save configuration` trên header.
   `Store secret` là nút secondary.
5. Không dùng đổ bóng. Chỉ có chế độ nền sáng.

## Ranh giới đang hoãn

Draft hiện chưa được cô lập theo người dùng đã đăng nhập; vấn đề này thuộc
[D15](../../../backlog/D15-authenticated-draft-isolation.md) và áp dụng cho cả
UC-01 lẫn UC-02.
