---
id: UC-05
artifact: use-case-specification
status: current
delivery_status: implemented-and-verified
last_reviewed: 2026-09-18
---

# UC-05 — Remove Application from Environment


## Mục tiêu

Cho phép Developer gỡ một application ra khỏi **một** environment và một nơi triển khai: workload ngừng chạy, hạ tầng do IDP tạo riêng cho application ở đó được hủy, hạ tầng dùng chung chỉ bị gỡ liên kết. Application Definition, các phiên bản, Environment Configuration, Delivery Repository và cặp khóa của application đều được giữ lại, nên deploy lại lúc nào cũng được.

Use case này tách khỏi UC-03 vì nó không triển khai phiên bản nào: Developer không chọn phiên bản Application Definition, không chọn phiên bản catalog và không chọn image.

## Actor

**Primary Actor:** Developer

## Tiền điều kiện

Developer có Auth Session hợp lệ và Local User Account còn `ACTIVE` theo UC-06.

Application đã từng được deploy thành công lên đúng environment và nơi triển khai được chọn.

Không có deployment nào của cùng application + environment + nơi triển khai **đã được xác nhận hoặc đang chạy**. Một plan chưa được xác nhận thì không giữ chỗ và không chặn yêu cầu gỡ; nó chỉ trở thành lỗi thời khi IDP dựng lại plan lúc xác nhận.

## Hậu điều kiện

Mọi Workload Instance của application trên environment và nơi triển khai đó ở trạng thái đã gỡ.

Mọi Resource Instance thuộc quyền sở hữu của application trên environment và nơi triển khai đó đã bị hủy (resource do IDP tạo) hoặc đã bị gỡ liên kết (resource có sẵn, dùng chung).

Không gian tên của application trên cụm không còn; desired state của environment đó không còn trong Delivery Repository.

Application Definition, mọi phiên bản, Environment Configuration của mọi environment, Delivery Repository và cặp khóa của application **không** bị xóa. Các environment và nơi triển khai khác không bị ảnh hưởng.

Lịch sử deployment được giữ lại và ghi thêm một Deployment Record cho lần gỡ này.

## Luồng chính

Developer mở application và chọn **Remove from Environment**.

Developer chọn environment và nơi triển khai cần gỡ.

**IDP** lấy lại phiên bản Application Definition, phiên bản catalog và Environment Configuration của lần deploy gần nhất trên đúng environment và nơi triển khai đó. Developer không chọn lại các thứ này: gỡ bỏ phải dựa trên đúng thứ đang chạy.

**IDP** liệt kê các thành phần đã khai báo trong phiên bản đó cùng mọi Resource Instance đang tồn tại của chủ sở hữu, rồi lập plan gỡ bỏ theo **thứ tự ngược với thứ tự triển khai**:

Tầng gỡ đầu tiên: mọi workload của application trên environment và nơi triển khai đó.

Các tầng sau: các resource, trong đó một resource chỉ được gỡ khi **không còn resource nào chưa gỡ cần tới loại của nó**. Nhờ quy tắc này, thứ mà nhiều thứ khác dựa vào — cụm Kubernetes, network — tự nhiên nằm ở tầng cuối. Mỗi resource là **hủy** nếu do IDP tạo và **gỡ liên kết** nếu là thứ có sẵn dùng chung.

Việc lập plan gỡ bỏ không cần tới giá trị cấu hình của environment: IDP gỡ những gì đang tồn tại, không phải những gì cấu hình mô tả.

**IDP** hiển thị plan gỡ bỏ, trong đó nêu rõ thành phần nào bị hủy kèm **cảnh báo mất dữ liệu**, thành phần nào chỉ bị gỡ liên kết và thành phần nào không bị đụng tới.

Developer xác nhận và chọn **Remove**.

**IDP** ghi nhận xác nhận cùng một job, rồi trả lời Developer ngay.

Deployment Worker của IDP nhận job và chạy lần lượt từng tầng gỡ bỏ ở chế độ chạy nền:

Gửi desired state mới, trong đó workload cần gỡ không còn, sang CD system và chờ CD system nhận.

Xác minh workload đã thật sự biến mất khỏi cụm rồi mới đánh dấu đã gỡ; cấu hình nhạy cảm của workload trên cụm cũng được xóa.

Gỡ đối tượng của application khỏi CD system và xóa phần desired state của environment đó khỏi Delivery Repository; repository và cặp khóa của application được giữ lại.

Xóa không gian tên của application trên cụm.

Hủy từng resource do IDP tạo, gỡ liên kết từng resource có sẵn, theo đúng thứ tự ngược của graph.

**IDP** lưu Deployment Record cho lần gỡ: environment, nơi triển khai, danh sách thành phần đã gỡ, đã hủy, đã gỡ liên kết, tiến trình theo tầng và kết quả.

## Luồng ngoại lệ

### A1 – Không có gì để gỡ hoặc đang có deployment khác chạy dở

**IDP** từ chối yêu cầu và không tạo deployment nào khi application chưa từng được deploy lên environment và nơi triển khai đã chọn, khi không còn workload hay resource nào của application ở đó, hoặc khi đang có một deployment của cùng application + environment + nơi triển khai đã được xác nhận hoặc đang chạy.

Không có hạ tầng nào bị đụng tới trong các trường hợp này.

### A2 – Gỡ bỏ hoặc hủy thất bại

Khi gỡ workload, hủy resource hoặc gỡ liên kết thất bại, **IDP** dừng lại: các thành phần ở tầng gỡ sau **không** bị hủy, đánh dấu deployment thất bại và lưu tầng, thành phần liên quan cùng mô tả lỗi.

Thành phần chưa được xác minh là đã gỡ khỏi cụm thì không bị đánh dấu là đã gỡ, và resource mà nó phụ thuộc không bị hủy. Developer có thể gỡ lại sau khi xử lý nguyên nhân.

## Dữ liệu chính

| Dữ liệu | Mô tả |
|---|---|
| Deployment | Cùng đối tượng với UC-03, nhưng thuộc loại gỡ bỏ; không có workload nào được chọn để triển khai |
| Plan gỡ bỏ | Các tầng gỡ theo thứ tự ngược, mỗi thành phần kèm hành động gỡ, hủy hoặc gỡ liên kết và cảnh báo mất dữ liệu |
| Deployment Record | Kết quả lần gỡ: thành phần đã gỡ, đã hủy, đã gỡ liên kết, tiến trình và lỗi nếu có |

## Quy tắc nghiệp vụ

Gỡ bỏ chỉ tác động lên đúng một environment và một nơi triển khai; các environment và nơi triển khai khác của cùng application không thay đổi.

Developer không chọn phiên bản Application Definition, phiên bản catalog hay image khi gỡ; IDP dùng lại đúng thứ đang chạy.

Resource có sẵn, dùng chung chỉ bị gỡ liên kết, không bao giờ bị hủy — kể cả khi nó chỉ đang phục vụ application này.

Việc hủy resource phải hiện trong plan kèm cảnh báo mất dữ liệu và phải được Developer xác nhận trước khi chạy.

Workload chỉ được đánh dấu đã gỡ sau khi IDP xác minh nó đã biến mất khỏi cụm; resource chỉ bị hủy sau khi các workload phụ thuộc nó đã được gỡ.

Delivery Repository của application và cặp khóa của nó được giữ lại; chỉ phần desired state của environment bị gỡ là bị xóa.

Application Definition, các phiên bản và Environment Configuration không bị xóa; sau khi gỡ, Developer deploy lại được bằng UC-03 mà không phải khai báo lại gì.

IDP chỉ gỡ những thứ chính nó tạo ra cho application trên environment và nơi triển khai đó; hạ tầng của application khác và hạ tầng dùng chung không bị đụng tới.
