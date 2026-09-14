# Step 2: Persistence Classification

Phân loại dưới đây bao phủ toàn bộ domain object trong `domain_model.puml`. Với các object thuộc application lifecycle, `PERSISTENT` nghĩa là state cần tồn tại qua nhiều request/deployment execution và được quản lý bởi đúng một trong sáu repository boundary đã thống nhất (năm repository ban đầu cộng **Workload Instance Repository** được bổ sung cho việc triển khai theo tầng). `Resource Definition` cũng là dữ liệu bền vững, nhưng là reference data thuộc catalog do platform quản lý và nằm ngoài phạm vi sáu repository này. `TRANSIENT` nghĩa là object chỉ được dựng/resolve trong một deployment execution và không được lưu như một domain record độc lập.

| Domain Object | Persistent/Transient | Repository (nếu persistent) | Lý do |
|---|---|---|---|
| Application Definition | PERSISTENT | Application Repository | Là identity bền vững của application và sở hữu các phiên bản định nghĩa. |
| Application Definition Version | PERSISTENT | Application Repository | Mỗi lần lưu tạo một phiên bản bất biến; deployment trỏ tới phiên bản đã dùng nên phiên bản cũ phải được giữ nguyên, không sửa, không xóa. |
| Workload | PERSISTENT | Application Repository | Được lưu theo từng phiên bản với ID cố định qua phiên bản; image repository và configuration requirements phải tồn tại qua các lần deploy. |
| Resource Requirement | PERSISTENT | Application Repository | Được lưu theo từng phiên bản với ID cố định; được dùng lại để dựng graph, resolve resource và làm khóa chủ sở hữu của Resource Instance. |
| Environment Variable Definition | PERSISTENT | Application Repository | Requirement khai báo ở UC-01 (theo phiên bản, ID cố định) phải được tải lại trong UC-02 và UC-03. |
| Secret Definition | PERSISTENT | Application Repository | Tên/requirement của Secret là metadata của Workload (theo phiên bản, ID cố định); không chứa secret plaintext. |
| Dependency | PERSISTENT | Application Repository | Topology `depends on` là một phần bền vững của từng phiên bản. |
| Application Specification | PERSISTENT | Specification Repository / Config Repo Service | Generated specification cần được lưu hoặc version hóa như đã quy định ở Step 1. |
| Environment Configuration | PERSISTENT | Environment Configuration Repository | Configuration được quản lý và đọc lại riêng theo application + environment (`STAGING`, `PRODUCTION`); không có phiên bản, tham chiếu thành phần qua ID cố định. |
| Environment Variable | PERSISTENT | Environment Configuration Repository | Binding variable theo workload/environment phải được dùng lại khi tạo deployment. |
| Secret | PERSISTENT | Environment Configuration Repository | Chỉ metadata và `secretReference`/output reference được lưu; plaintext nằm ngoài repository này. |
| Configuration Value | PERSISTENT | Environment Configuration Repository | Persist discriminator của nguồn giá trị và payload subtype trong Environment Configuration. |
| Direct Configuration Value | PERSISTENT | Environment Configuration Repository | Direct value của Environment Variable thông thường cần được dùng lại cho deployment sau; không chứa plaintext Secret. |
| Resource Output Reference | PERSISTENT | Environment Configuration Repository | Lưu logical reference như `DB_HOST -> postgresql.host`, không lưu resolved value. |
| Workload Output Reference | PERSISTENT | Environment Configuration Repository | Lưu logical reference như `BACKEND_URL -> backend.endpoint`, không lưu runtime value. |
| Resource Definition | PERSISTENT (platform-managed) | Platform Resource Definition catalog (ngoài phạm vi sáu application-lifecycle repository) | Là reference data do platform quản lý, dùng qua nhiều deployment để resolve logical resource; khai báo loại `MANAGED` (IDP quản lý hạ tầng) hoặc `EXISTING` (trỏ tới resource dùng chung có sẵn, kèm điều kiện áp dụng). Không thuộc Specification Repository / Config Repo Service, vì repository đó chỉ lưu/version hóa Application Specification được sinh. |
| Deployment | PERSISTENT | Deployment Repository | Deployment cần identity, phiên bản Application Definition đã deploy và lifecycle status bền vững để xác nhận, theo dõi và truy vấn lịch sử. |
| Workload Deployment | PERSISTENT | Deployment Repository | Phải lưu chính xác image repository/version thực tế, lý do có mặt (được chọn hay tự động deploy lại) và tầng của từng workload trong mỗi deployment. |
| Deployment Execution Job | PERSISTENT | Deployment Repository | Job được tạo cùng transaction với việc xác nhận deployment và giữ các giá trị override đã chọn, để Deployment Worker chạy nền sau khi request xác nhận đã kết thúc và không mất khi server khởi động lại. |
| Deployment Context | PERSISTENT | Deployment Repository | Environment, target và context đã dùng phải được giữ để audit/reproduce kết quả deployment. |
| Deployment Graph | TRANSIENT | — | Được dựng lại từ phiên bản Application Definition, Environment Configuration, images và context cho một execution (gồm phạm vi, các tầng và danh sách có thể deploy lại); không phải source of truth. |
| Resource Resolution | TRANSIENT | — | Là quyết định trung gian của resolver trong execution hiện tại; durable outcome là Resource Instance/reference. |
| Resource Instance | PERSISTENT | Resource Instance Repository | Durable infrastructure identity theo chủ sở hữu (application + environment + resource requirement + target), state/reference, status và dấu vân tay output cần cho reconcile/update/reuse, gỡ/hủy, lan truyền thay đổi output và UC-04. Dòng được giữ lại với status `DESTROYED`/`UNLINKED` sau khi gỡ để lịch sử vẫn tham chiếu được. |
| Resource Output | TRANSIENT | — | Object ở đây là output đã được collector nạp vào memory sau khi resource ready để resolve configuration; nó không được lưu như domain record độc lập. Durable infrastructure/provider reference vẫn nằm trong Resource Instance; chỉ dấu vân tay output được lưu trên Resource Instance. |
| Workload Instance | PERSISTENT | Workload Instance Repository | Trạng thái hiện hành của từng workload theo environment + target (workload deployment đang chạy, status, dấu vân tay output) cần cho deploy một phần, tự động deploy lại khi output đổi, gỡ workload và UC-04. Dòng được giữ lại với status `REMOVED` sau khi gỡ. |
| Workload Output | TRANSIENT | — | Output được Workload Output Collector đọc từ workload đã healthy hoặc đang chạy để resolve configuration trong một execution; không lưu giá trị, chỉ dấu vân tay output được lưu trên Workload Instance. |
| Resolved Configuration | TRANSIENT | — | Chỉ là snapshot giá trị đã resolve cho một deployment execution; source of truth vẫn là Environment Configuration references và Secret Store. |
| Resolved Specification | TRANSIENT | — | Được sinh cho execution hiện tại làm input cho `score-k8s`; application specification chưa resolve mới là artifact được version hóa. |
| Deployment Record | PERSISTENT | Deployment Repository | Cung cấp history/detail, final status, infrastructure/delivery references, các thành phần đã gỡ/hủy/gỡ liên kết và error summary cho UC-04. |
| Deployment Step | PERSISTENT | Deployment Repository | Progress theo tầng/thành phần, failed step và error detail phải còn lại để xem kết quả sau khi execution kết thúc (ai ghi và ghi lúc nào: D4). |

## Ba business constraint ảnh hưởng trực tiếp tới persistence

### 1. Persist references, không persist resolved values

`Environment Configuration Repository` lưu value source. Với output-based configuration, dữ liệu bền vững là reference, ví dụ:

```text
DB_HOST -> postgresql.host
BACKEND_URL -> backend.endpoint
```

Repository không thay các reference này bằng host, port, endpoint hoặc credential đã resolve. `Resource Output`, `Workload Output` và `Resolved Configuration` chỉ tồn tại trong execution của UC-03; nhờ vậy deployment sau luôn resolve theo Resource Instance, Workload Instance và context hiện hành, còn việc thay configuration không tự động sửa deployment đang chạy.

Để phát hiện output thay đổi (và tự động deploy lại thành phần phụ thuộc), hệ thống chỉ lưu **dấu vân tay (hash) output** trên Resource Instance và Workload Instance, không lưu giá trị output.

### 2. Persist Secret reference, không persist plaintext

Khi Developer nhập Secret trực tiếp, `Secret Store / Secret Management Adapter` lưu secret value và trả về một `secretReference`. `Environment Configuration Repository` chỉ persist tên Secret, workload association và reference này. Nếu Secret lấy từ sensitive Resource Output, repository chỉ persist `Resource Output Reference`. Plaintext Secret không xuất hiện trong `Secret`, `Direct Configuration Value`, `Environment Configuration`, log hay Deployment Record.

### 3. Persist phiên bản bất biến, không ghi đè hay xóa định nghĩa

`Application Repository` lưu mỗi lần Save ở UC-01 thành một Application Definition Version mới; phiên bản đã lưu không bị sửa hay xóa. Workload, Resource Requirement, Environment Variable Definition và Secret Definition được lưu theo từng phiên bản nhưng giữ ID cố định qua các phiên bản. Nhờ vậy:

- Deployment cũ luôn trỏ được tới đúng phiên bản đã deploy, lịch sử không bị hỏng khi một thành phần bị bỏ ở phiên bản sau.
- Environment Configuration (không có phiên bản), Resource Instance và Workload Instance tham chiếu thành phần qua ID cố định, nên không bị tạo mới hay mất liên kết khi thành phần đổi tên.
- Thành phần không còn trong phiên bản được deploy chỉ bị gỡ ở hạ tầng/cluster; dòng Resource Instance và Workload Instance được giữ lại với status kết thúc (`DESTROYED`, `UNLINKED`, `REMOVED`).
