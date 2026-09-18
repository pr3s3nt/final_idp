---
id: UC-01-UI-DESIGN
artifact: user-interface-design
status: current
last_reviewed: 2026-09-18
related: UC-01, ADR-016, ADR-017
---

# UC-01 — Thiết kế giao diện Application Builder

## Mục đích và ranh giới

Tài liệu này định nghĩa thiết kế màn hình và tương tác đã được chấp nhận cho
React editor của UC-01. [Đặc tả UC-01](../specification.md) vẫn là tài liệu sở
hữu hành vi sản phẩm bắt buộc; tài liệu này chuyển hành vi đó thành hợp đồng
giao diện mà con người và AI agent có thể review.

Editor là một **Application Builder có cấu trúc**. Developer chỉnh sửa từng
component trong form tập trung và dùng topology chỉ đọc ở `Review` để kiểm tra
kết quả. Topology không phải bề mặt chỉnh sửa kéo-thả.

UC-01 chỉ nhắm đến trình duyệt web trên desktop. Ứng dụng mobile, React Native,
layout riêng cho điện thoại/máy tính bảng và luồng mobile riêng đều nằm ngoài
phạm vi.

## Nguyên tắc thiết kế

1. Tổ chức editor theo các component mà Developer dùng để hình dung
   application: application, workload và resource.
2. Giữ mọi thuộc tính của một workload ở cùng một nơi: định danh, runtime,
   output, configuration requirement và dependency.
3. Luôn cho thấy vị trí đang chỉnh sửa, trạng thái validation và trạng thái
   chưa lưu mà không bắt Developer phải quét một form dài.
4. Dùng `Review` làm bước kiểm tra cuối có chủ đích. Nó tổng hợp draft và hiển
   thị topology mà không tạo thêm một biểu diễn có thể chỉnh sửa.
5. Giữ nguyên draft do browser sở hữu, cơ chế Save toàn bộ draft và optimistic
   concurrency theo ADR-016.
6. Ưu tiên control gốc của trình duyệt, label dễ đọc và điều hướng bằng bàn
   phím. Không bắt buộc UI framework hoặc thư viện icon.

## Kiến trúc thông tin

Editor có ba vùng cố định trên giao diện desktop:

- **Header** — điều hướng quay lại, định danh application, trạng thái
  version/draft, `Discard` và `Save application`.
- **Builder navigation** — `Overview`, một mục cho từng Workload, một mục cho
  từng Resource, các action thêm component và `Review`. Mỗi mục hiển thị số lỗi
  khi các field thuộc nó có lỗi validation.
- **Workspace** — form đang được tập trung hoặc màn `Review` chỉ đọc.

`Overview` chỉnh sửa application name, description và giải thích ranh giới của
UC-01. Workspace của Workload sở hữu các field chung, output, Environment
Variable definition, Secret definition và dependency. Workspace của Resource
sở hữu resource name và type. `Review` sở hữu topology, validation summary và
phần tổng hợp component/configuration.

Đổi tên component không được làm thay đổi định danh điều hướng: React key và
selection dùng component ID cố định, còn label hiển thị được cập nhật ngay.

## Sơ đồ giao diện web desktop

- [Workspace chỉnh sửa component](application-builder.puml) định nghĩa header
  cố định, navigation theo component và workspace đang được tập trung.
- [Workspace Review](review.puml) định nghĩa validation, topology chỉ đọc và
  component summary.
- [Trạng thái editor và phục hồi](states.puml) định nghĩa các chuyển tiếp khi
  load, chỉnh draft, validation, Save, lỗi và conflict.

Topology trong `Review` có thể dùng layout CSS/HTML xác định trước, không cần
là graph canvas tổng quát. Mọi quan hệ cũng phải được biểu diễn bằng text để
người dùng công nghệ hỗ trợ và application graph lớn vẫn hiểu được nội dung.

## Ranh giới desktop web

Header, navigation và workspace tạo thành một bề mặt duy nhất trên trình duyệt
desktop. Khi cửa sổ desktop bị thu hẹp, control có thể xuống dòng hoặc table có
thể cuộn trong vùng chứa như một cơ chế phòng vệ; đây không phải layout mobile
riêng và không phải mục tiêu acceptance cho mobile. Không được làm biến mất
field hoặc action khi cửa sổ đổi kích thước.

## Quy tắc tương tác

### Điều hướng và chỉnh sửa

- Application mới mở tại `Overview`. Application hiện có cũng mở tại
  `Overview` sau khi tải version mới nhất và phục hồi draft của tab nếu có.
- `Add workload` hoặc `Add resource` tạo component trong client draft và mở
  ngay workspace của component đó.
- Khi xóa component đang chọn, UI phải yêu cầu xác nhận trước. Sau khi xác nhận,
  draft reducer xóa component cùng các dependency liên quan rồi chọn component
  gần nhất còn lại hoặc `Overview`.
- Trong workspace Workload, nguồn của dependency được hiểu ngầm là workload
  hiện tại. Developer chỉ chọn component mà workload phụ thuộc vào. Quan hệ đã
  tồn tại được hiển thị và có thể xóa tại đây.
- `Review` luôn có thể truy cập và không làm thay đổi draft.

### Validation

- Local validation chạy trên draft hiện tại. Lỗi field xuất hiện cạnh control
  tương ứng sau lần Save đầu tiên hoặc khi backend trả về lỗi cho field đó.
- Badge trong navigation hiển thị số lỗi của `Overview`, từng component và
  `Review`. Khi chọn một mục, Developer nhìn thấy lỗi thuộc mục đó; `Review`
  cung cấp link quay lại đúng workspace hoặc field bị ảnh hưởng.
- `Save application` kiểm tra toàn bộ draft. Nếu có lỗi local, editor mở
  `Review`, focus validation summary và không gửi API request.
- Validation của backend là nguồn có thẩm quyền và được hợp nhất vào cùng cách
  trình bày ở navigation, field và `Review`.

### Trạng thái draft và action

- Header phân biệt `Saved`, `Unsaved changes`, `Restored draft` và `Saving…`.
  Với definition đã tồn tại, header cũng hiển thị base version và version sẽ
  được tạo nếu Save thành công.
- `Save application` và `Discard` luôn nằm trên header ở mọi workspace. Save
  gửi toàn bộ draft đúng một lần; thay đổi field hoặc navigation không gọi API.
- `Discard` chỉ yêu cầu xác nhận khi có thay đổi. Action này xóa local draft và
  tải lại durable definition hoặc khởi tạo definition mới rỗng.
- Save thành công xác nhận version đã lưu và xóa local draft theo ADR-016.

### Conflict khi Save

`DRAFT_CONFLICT` không được thay thế hoặc xóa draft của Developer. Editor hiển
thị conflict panel chặn luồng với:

1. giải thích rằng một version mới hơn đã được lưu trước;
2. **`Copy draft JSON`** để Developer giữ công việc bên ngoài tab;
3. **`Download draft JSON`** khi trình duyệt hỗ trợ download;
4. **`Load latest version`** với xác nhận rõ rằng local draft sẽ bị bỏ.

Tự động merge và diff song song nằm ngoài UC-01. JSON được export chỉ là công
cụ phục hồi, không phải import contract hoặc public API contract.

## Trạng thái màn hình

| Trạng thái | Cách hiển thị và phục hồi |
|---|---|
| Bắt đầu load | Hiển thị trạng thái/skeleton workspace, không nháy form rỗng có thể chỉnh sửa |
| Load thất bại | Giải thích lỗi, `Try again` và `Back to applications` |
| Không tìm thấy application | Giải thích không tìm thấy và `Back to applications` |
| Draft sạch | Trạng thái `Saved`; `Discard` bị disable |
| Draft có thay đổi | `Unsaved changes · kept in this tab`; cho phép Save và Discard |
| Draft được phục hồi | Thông báo và trạng thái header `Restored draft` |
| Đang lưu | Disable Save với `Saving…`; nội dung hiện tại vẫn hiển thị |
| Validation thất bại | Chọn `Review`, focus summary, hiển thị badge và lỗi field |
| Save thất bại | Lỗi không phá hủy draft; thử lại bằng action Save thông thường |
| Conflict | Giữ draft; cho copy/download và chỉ load bản mới nhất sau xác nhận |
| Save thành công | Thông báo version đã lưu, draft sạch và durable version được tải lại |

## Ánh xạ action sang state/API

| Action của người dùng | Trạng thái client draft/session | HTTP request |
|---|---|---|
| Mở application mới | Phục hồi draft `new` hoặc tạo draft rỗng | Không |
| Mở application hiện có | Tải version mới nhất, sau đó phục hồi draft của tab khi có | Một `GET /api/application-definitions/{applicationId}` |
| Chọn workspace | Chỉ thay đổi selection | Không |
| Thêm, sửa, đổi tên hoặc xóa component/field | Cập nhật reducer và `sessionStorage` | Không |
| Mở `Review` | Chỉ thay đổi selection; validate draft hiện tại để hiển thị | Không |
| `Discard` | Xóa `sessionStorage`; khởi tạo hoặc tải lại | Application hiện có thực hiện load GET thông thường |
| Save application mới hợp lệ | Giữ draft đến khi thành công rồi xóa | Một `POST /api/application-definitions` |
| Save application hiện có hợp lệ | Giữ draft đến khi thành công rồi xóa | Một `POST /api/application-definitions/{applicationId}/versions` |
| Copy/download draft conflict | Không thay đổi state | Không |
| `Load latest version` sau conflict | Xác nhận, xóa local draft và tải version mới nhất | Một load GET |

## Hợp đồng accessibility

- Shell có đúng một `h1`; các section trong workspace dùng thứ tự heading hợp
  lý.
- Builder navigation là một navigation region có label. Workspace đang chọn
  dùng `aria-current`; badge lỗi có text cho screen reader.
- Mọi control có label cố định. Không truyền đạt ý nghĩa chỉ bằng màu hoặc icon.
- Validation summary và lỗi save/load dùng live-region role phù hợp; focus chỉ
  di chuyển sau action Save hoặc phục hồi do người dùng chủ động thực hiện.
- Accessible name của action thêm/xóa chứa tên component hoặc requirement bị
  ảnh hưởng.
- Mọi action chỉnh sửa, điều hướng, xác nhận và phục hồi conflict hoạt động bằng
  bàn phím.
- Giao diện tôn trọng thiết lập giảm chuyển động, duy trì focus rõ ràng và độ
  tương phản phù hợp.

## Ngôn ngữ thị giác

Dùng nền trung tính nhẹ, bề mặt làm việc màu trắng, chữ xanh navy đậm và một
màu xanh dương cho action. Loại component có thể dùng hình dạng/label tiết chế
và dư thừa thông tin (hình tròn cho workload, hình thoi cho resource), nhưng
hình dạng và màu không thay thế text. Border và khoảng cách tạo phân cấp; tránh
card trang trí lồng nhau, hero lớn và dashboard metric không hỗ trợ hoàn thành
UC-01.

Kết quả cần mang cảm giác của một công cụ kỹ thuật: đủ gọn để so sánh cấu hình,
nhưng không dày đặc đến mức che label, lỗi hoặc hướng dẫn phục hồi.
