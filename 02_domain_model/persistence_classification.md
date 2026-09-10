# Step 2: Persistence Classification

Phân loại dưới đây bao phủ toàn bộ domain object trong `domain_model.puml`. Với các object thuộc application lifecycle, `PERSISTENT` nghĩa là state cần tồn tại qua nhiều request/deployment execution và được quản lý bởi đúng một trong năm repository boundary đã thống nhất. `Resource Definition` cũng là dữ liệu bền vững, nhưng là reference data thuộc catalog do platform quản lý và nằm ngoài phạm vi năm repository này. `TRANSIENT` nghĩa là object chỉ được dựng/resolve trong một deployment execution và không được lưu như một domain record độc lập.

| Domain Object | Persistent/Transient | Repository (nếu persistent) | Lý do |
|---|---|---|---|
| Application Definition | PERSISTENT | Application Repository | Là source of truth của cấu trúc logic application và phải đọc lại khi configure/deploy. |
| Workload | PERSISTENT | Application Repository | Là thành phần được Application Definition sở hữu; image repository và configuration requirements phải tồn tại qua các lần deploy. |
| Resource Requirement | PERSISTENT | Application Repository | Logical resource requirement là một phần của Application Definition, được dùng lại để dựng graph và resolve resource. |
| Environment Variable Definition | PERSISTENT | Application Repository | Requirement khai báo ở UC-01 phải được tải lại trong UC-02 và UC-03. |
| Secret Definition | PERSISTENT | Application Repository | Tên/requirement của Secret là metadata của Workload; không chứa secret plaintext. |
| Dependency | PERSISTENT | Application Repository | Topology `depends on` là một phần bền vững của Application Definition. |
| Application Specification | PERSISTENT | Specification Repository / Config Repo Service | Generated specification cần được lưu hoặc version hóa như đã quy định ở Step 1. |
| Environment Configuration | PERSISTENT | Environment Configuration Repository | Configuration được quản lý và đọc lại riêng theo application + environment. |
| Environment Variable | PERSISTENT | Environment Configuration Repository | Binding variable theo workload/environment phải được dùng lại khi tạo deployment. |
| Secret | PERSISTENT | Environment Configuration Repository | Chỉ metadata và `secretReference`/output reference được lưu; plaintext nằm ngoài repository này. |
| Configuration Value | PERSISTENT | Environment Configuration Repository | Persist discriminator của nguồn giá trị và payload subtype trong Environment Configuration. |
| Direct Configuration Value | PERSISTENT | Environment Configuration Repository | Direct value của Environment Variable thông thường cần được dùng lại cho deployment sau; không chứa plaintext Secret. |
| Resource Output Reference | PERSISTENT | Environment Configuration Repository | Lưu logical reference như `DB_HOST -> postgresql.host`, không lưu resolved value. |
| Workload Output Reference | PERSISTENT | Environment Configuration Repository | Lưu logical reference như `BACKEND_URL -> backend.endpoint`, không lưu runtime value. |
| Resource Definition | PERSISTENT (platform-managed) | Platform Resource Definition catalog (ngoài phạm vi năm application-lifecycle repository) | Là reference data do platform quản lý, dùng qua nhiều deployment để resolve logical resource. Không thuộc Specification Repository / Config Repo Service, vì repository đó chỉ lưu/version hóa Application Specification được sinh. |
| Deployment | PERSISTENT | Deployment Repository | Deployment cần identity và lifecycle status bền vững để xác nhận, theo dõi và truy vấn lịch sử. |
| Workload Deployment | PERSISTENT | Deployment Repository | Phải lưu chính xác image repository/version thực tế cho từng workload của mỗi deployment. |
| Deployment Context | PERSISTENT | Deployment Repository | Environment, target và context đã dùng phải được giữ để audit/reproduce kết quả deployment. |
| Deployment Graph | TRANSIENT | — | Được dựng lại từ Application Definition, Environment Configuration, images và context cho một execution; không phải source of truth. |
| Resource Resolution | TRANSIENT | — | Là quyết định trung gian của resolver trong execution hiện tại; durable outcome là Resource Instance/reference. |
| Resource Instance | PERSISTENT | Resource Instance Repository | Durable infrastructure identity, state/reference và status cần cho reconcile/update/reuse và UC-04. |
| Resource Output | TRANSIENT | — | Object ở đây là output đã được collector nạp vào memory sau khi resource ready để resolve configuration; nó không được lưu như domain record độc lập. Durable infrastructure/provider reference vẫn nằm trong Resource Instance. |
| Resolved Configuration | TRANSIENT | — | Chỉ là snapshot giá trị đã resolve cho một deployment execution; source of truth vẫn là Environment Configuration references và Secret Store. |
| Resolved Specification | TRANSIENT | — | Được sinh cho execution hiện tại làm input cho `score-k8s`; application specification chưa resolve mới là artifact được version hóa. |
| Deployment Record | PERSISTENT | Deployment Repository | Cung cấp history/detail, final status, infrastructure/delivery references và error summary cho UC-04. |
| Deployment Step | PERSISTENT | Deployment Repository | Progress, failed step và error detail phải còn lại để xem kết quả sau khi execution kết thúc. |

## Hai business constraint ảnh hưởng trực tiếp tới persistence

### 1. Persist references, không persist resolved values

`Environment Configuration Repository` lưu value source. Với output-based configuration, dữ liệu bền vững là reference, ví dụ:

```text
DB_HOST -> postgresql.host
BACKEND_URL -> backend.endpoint
```

Repository không thay các reference này bằng host, port, endpoint hoặc credential đã resolve. `Resource Output` và `Resolved Configuration` chỉ tồn tại trong execution của UC-03; nhờ vậy deployment sau luôn resolve theo Resource Instance và context hiện hành, còn việc thay configuration không tự động sửa deployment đang chạy.

### 2. Persist Secret reference, không persist plaintext

Khi Developer nhập Secret trực tiếp, `Secret Store / Secret Management Adapter` lưu secret value và trả về một `secretReference`. `Environment Configuration Repository` chỉ persist tên Secret, workload association và reference này. Nếu Secret lấy từ sensitive Resource Output, repository chỉ persist `Resource Output Reference`. Plaintext Secret không xuất hiện trong `Secret`, `Direct Configuration Value`, `Environment Configuration`, log hay Deployment Record.
