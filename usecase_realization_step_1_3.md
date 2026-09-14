# Usecase
UC-01 – Create / Configure Application

## Mục tiêu

Cho phép Developer tạo mới hoặc chỉnh sửa application thông qua giao diện **IDP**.

Developer khai báo:

Workload.

Resource mà application cần.

Quan hệ dependency.

Environment Variable và Secret mà từng workload cần.

**IDP** lưu Application Definition và sinh application specification tương ứng, ví dụ score.yaml.

2. Actor

Primary Actor: Developer

## Tiền điều kiện

Developer đã đăng nhập.

Developer có quyền tạo hoặc chỉnh sửa application.

## Hậu điều kiện

Một phiên bản mới của Application Definition được tạo; các phiên bản cũ giữ nguyên, không bị sửa hay xóa.

Workload, resource, dependency và configuration requirement được lưu trong phiên bản mới; mỗi thành phần giữ ID cố định qua các phiên bản.

Application specification được tạo hoặc cập nhật theo phiên bản mới.

Deployment hiện tại không tự động bị thay đổi; phiên bản mới chỉ có hiệu lực ở một environment khi được deploy vào environment đó trong UC-03.

## Luồng chính

Developer chọn Create Application hoặc mở application hiện có để chỉnh sửa.

Developer khai báo:

Application name.

Description.

Developer chọn Add Resource để khai báo resource application cần.

Với mỗi resource:

Resource name.

Resource type.

Ví dụ:

Name: postgresql Type: PostgreSQL

Developer chọn Add Workload.

Với mỗi workload, Developer khai báo:

Workload name.

Workload type.

Image repository.

Application port nếu cần.

Output mà workload cung cấp cho thành phần khác nếu có, ví dụ endpoint.

Ví dụ:

Workload: backend Type: Backend Service Image Repository: registry.company.local/shop-backend Port: **8080** Outputs: endpoint

Trong workload, Developer khai báo các Environment Variable mà workload cần bằng Add Environment Variable.

Ví dụ:

LOG_LEVEL PAYMENT_API_URL DB_HOST DB_PORT

Developer khai báo các Secret mà workload cần bằng Add Secret.

Ví dụ:

DB_USERNAME DB_PASSWORD PAYMENT_API_KEY

Developer khai báo dependency giữa các thành phần bằng quan hệ depends on.

Ví dụ:

frontend → backend backend  → postgresql

Quan hệ depends on quyết định thứ tự triển khai trong UC-03 và giới hạn những output mà workload được phép tham chiếu trong UC-02.

**IDP** hiển thị topology và cấu hình tổng quan của application.

Developer chọn Save Application.

**IDP** kiểm tra dữ liệu, lưu Application Definition thành một phiên bản mới và sinh/cập nhật application specification.

Nếu Developer đổi tên một thành phần, thành phần đó vẫn giữ ID cũ trong phiên bản mới. Nếu Developer bỏ một thành phần, phiên bản mới không còn thành phần đó; phiên bản cũ vẫn giữ nguyên.

## Luồng ngoại lệ

A1 – Dữ liệu không hợp lệ

Nếu dữ liệu không hợp lệ, **IDP** hiển thị lỗi để Developer chỉnh sửa.

Ví dụ:

Workload name bị trùng.

Image repository không hợp lệ.

Port không hợp lệ.

Dependency tham chiếu tới thành phần không tồn tại.

Các quan hệ depends on tạo thành vòng, ví dụ frontend → backend và backend → frontend.

A1 chỉ xét lỗi bên trong phiên bản đang lưu. Việc Environment Configuration của từng environment có khớp với phiên bản hay không được kiểm tra khi deploy ở UC-03.

## Dữ liệu chính

Nhóm

Dữ liệu

Application

Name, description

Workload

Name, type, image repository, port, outputs

Resource

Name, resource type

### Environment Variable Definition

Name

### Secret Definition

Name

Dependency

Source → target

## Quy tắc nghiệp vụ

Một application có thể có một hoặc nhiều workload.

Application có thể yêu cầu không hoặc nhiều resource.

Environment Variable và Secret thuộc về một workload cụ thể.

UC-01 chỉ khai báo workload cần configuration gì, chưa khai báo giá trị cụ thể theo environment.

Image repository có thể được khai báo trong Workload Definition.

Image tag/version không thuộc UC-01.

Image tag/version được xác định khi tạo deployment trong UC-03 – Deploy Application hoặc được cung cấp thông qua CI/External Delivery Integration.

Secret được phân biệt với Environment Variable thông thường.

Developer không khai báo Kubernetes ConfigMap hoặc Kubernetes Secret.

Developer chỉ mô tả resource ở mức logic, ví dụ PostgreSQL, Redis.

Developer không khai báo cách resource được provision.

Developer không cần thao tác trực tiếp với score.yaml.

Environment và deployment target không được lựa chọn trong UC-01.

Hai thành phần không có quan hệ depends on được coi là độc lập với nhau.

Workload chỉ được dùng output của thành phần mà nó depends on (xem UC-02).

Các quan hệ depends on không được tạo thành vòng.

Mỗi lần lưu tạo một phiên bản mới của Application Definition; phiên bản đã lưu không bao giờ bị sửa hay xóa.

Workload, resource, Environment Variable Definition và Secret Definition giữ ID cố định qua các phiên bản; đổi tên không làm đổi ID.

Bỏ một thành phần khỏi application nghĩa là phiên bản mới không còn thành phần đó. Workload hoặc hạ tầng đang chạy chỉ bị gỡ khi environment được deploy phiên bản mới trong UC-03.

Mọi application có đúng hai environment cố định: staging và production.

9. Ví dụ

Application: shop-app

Resources ────────────────────────────

postgresql Type: PostgreSQL

Workload: backend ────────────────────────────

Type: Backend Service Image Repository: registry.company.local/shop-backend

Port: **8080**

Outputs: endpoint

Environment Variables:
    LOG_LEVEL
    DB_HOST
    DB_PORT

Secrets:
    DB_USERNAME
    DB_PASSWORD

Depends on: postgresql

Workload: frontend ────────────────────────────

Type: Frontend Image Repository: registry.company.local/shop-frontend

Port: **3000**

Environment Variables: BACKEND_URL

Depends on: backend

Không có image version cụ thể trong UC-01:

✗ backend:v1.4.2 ✗ frontend:v2.1.0

Các version cụ thể sẽ được lựa chọn hoặc nhận từ CI khi thực hiện deployment trong UC-03.

UC-02 – Configure Application Environment

## Mục tiêu

Cho phép Developer cấu hình giá trị Environment Variable và Secret của application cho từng environment.

Các giá trị có thể:

Được nhập trực tiếp.

Lấy từ output của một resource.

Lấy từ output của một workload khác.

2. Actor

Primary Actor: Developer

## Tiền điều kiện

Application đã được khai báo trong UC-01.

Application có Environment Variable hoặc Secret cần cấu hình.

## Hậu điều kiện

Configuration của application cho environment được lưu.

Các reference tới Resource Output hoặc Workload Output được lưu.

Việc thay đổi configuration chưa tự động deploy application.

## Luồng chính

Developer mở application.

Developer chọn trang Configuration.

Developer chọn environment cần cấu hình. Mọi application có đúng hai environment:

staging production

**IDP** hiển thị Environment Variable và Secret đã được khai báo cho từng workload trong UC-01.

Với mỗi Environment Variable, Developer chọn nguồn giá trị.

### Environment Value

Developer nhập trực tiếp giá trị.

Ví dụ:

LOG_LEVEL = **INFO**

### Resource Output

Developer chọn:

Resource.

Output do resource cung cấp.

Ví dụ:

DB_HOST

Source: ### Resource Output

Resource: postgresql

Output: host

**IDP** chỉ hiển thị các resource mà workload chứa biến này depends on, cùng danh sách output hợp lệ của từng resource để Developer lựa chọn.

### Workload Output

Developer chọn workload và output tương ứng.

**IDP** chỉ hiển thị các workload mà workload chứa biến này depends on, ví dụ frontend depends on backend nên frontend được chọn output của backend.

Ví dụ:

BACKEND_URL

Source: ### Workload Output

Workload: backend

Output: endpoint

Với Secret, Developer có thể:

Nhập giá trị secret theo cơ chế bảo mật của **IDP**.

Hoặc chọn output nhạy cảm của resource mà workload depends on.

Developer chọn Save Configuration.

**IDP** kiểm tra và lưu Environment Configuration.

## Luồng ngoại lệ

A1 – Configuration không hợp lệ

**IDP** yêu cầu Developer chỉnh sửa nếu:

Thiếu giá trị bắt buộc.

Resource không tồn tại.

Output được chọn không tồn tại.

Workload output không hợp lệ.

Output thuộc resource hoặc workload mà workload chứa biến không depends on.

## Dữ liệu chính

Nhóm

Dữ liệu

### Environment Configuration

Application, environment

### Environment Variable

Workload, variable name, value source

Secret

Workload, secret name, value source

### Resource Output Reference

Resource + output

### Workload Output Reference

Workload + output

## Quy tắc nghiệp vụ

Configuration được quản lý riêng theo từng environment (staging, production).

Cùng một Environment Variable có thể có giá trị khác nhau giữa các environment.

Configuration không có phiên bản. Khi lưu, **IDP** kiểm tra configuration theo phiên bản Application Definition mới nhất; khi deploy một phiên bản vào environment, UC-03 kiểm tra lại configuration của environment đó có khớp với phiên bản được deploy hay không.

Configuration tham chiếu workload, resource, Environment Variable Definition và Secret Definition qua ID cố định, nên vẫn đúng khi các thành phần đó đổi tên ở phiên bản sau.

**IDP** chỉ cho Developer chọn những output mà Resource Definition hoặc workload expose.

Workload chỉ được tham chiếu output của resource hoặc workload mà nó đã khai báo depends on trong UC-01. Nếu cần output của thành phần khác, Developer phải bổ sung dependency ở UC-01 trước.

Resource Output và Workload Output chỉ lưu reference; giá trị thực tế được resolve trong quá trình deployment.

Giá trị Secret không được hiển thị lại dưới dạng plaintext.

Thay đổi Environment Configuration không tự động làm thay đổi deployment đang chạy.

# UC-03 – Deploy Application

## 1. Mục tiêu

Cho phép Developer triển khai một phiên bản Application Definition cụ thể, cho toàn bộ hoặc một phần application, lên một environment (staging hoặc production) và Kubernetes deployment target. Thử một phiên bản ở staging xong thì deploy cùng phiên bản đó lên production.

Image tag/version được xác định tại thời điểm deployment, không phải khi định nghĩa application trong UC-01.

**IDP** chịu trách nhiệm:

- Kiểm tra Environment Configuration của environment khớp với phiên bản Application Definition được deploy.
- Xác định phạm vi deployment: các workload được chọn và các resource mà chúng depends on trực tiếp.
- Xác định dependency và chia các thành phần trong phạm vi thành các tầng triển khai theo thứ tự phụ thuộc.
- Resolve Resource Definition phù hợp theo deployment context.
- Tìm resource để dùng lại theo đúng chủ sở hữu (application + environment + resource requirement + deployment target); resource dùng chung chỉ được dùng khi Resource Definition khai báo trỏ tới resource có sẵn.
- Reconcile infrastructure.
- Gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản được deploy.
- Nhận yêu cầu deploy và trả lời Developer ngay; việc triển khai chạy nền.
- Triển khai lần lượt từng tầng: resolve Environment Configuration từ output của các tầng trước, sinh Kubernetes manifest, chuyển desired deployment state cho hệ thống Continuous Delivery và chờ workload healthy.
- Thu thập Resource Output và Workload Output sau khi thành phần tương ứng sẵn sàng.
- Tự động triển khai lại các thành phần phụ thuộc khi output mà chúng dùng bị thay đổi.

## 2. Actor

**Primary Actor:** Developer

## 3. Tiền điều kiện

- Phiên bản Application Definition được chọn tồn tại, hợp lệ và các quan hệ depends on không tạo thành vòng.
- Workload cần deploy có Image Repository.
- Environment Configuration cần thiết của environment đã được cấu hình và khớp với phiên bản được chọn.
- Platform đã có Resource Definition và provisioner phù hợp.
- Deployment target được hỗ trợ.
- Nếu chỉ deploy một phần application, phiên bản được chọn phải trùng phiên bản đang chạy trên environment và deployment target đó.
- Mọi workload mà các workload được chọn depends on nhưng không nằm trong phạm vi deployment đang chạy healthy trên cùng environment và deployment target.

## 4. Hậu điều kiện

- Deployment ghi nhận phiên bản Application Definition đã được deploy.
- Deployment ghi nhận chính xác image version của từng workload được triển khai, gồm cả workload được triển khai lại tự động.
- Infrastructure trong phạm vi deployment đã được reconcile và sẵn sàng.
- Khi deploy một phiên bản mới, workload và resource không còn trong phiên bản đó đã được gỡ, hủy hoặc gỡ liên kết trên environment và deployment target đó.
- Environment Configuration và các dependency reference được resolve.
- Kubernetes manifest được sinh với đúng image và configuration.
- Mọi workload được triển khai đều healthy.
- Trạng thái hiện hành của từng workload và resource đã triển khai (Workload Instance, Resource Instance), gồm dấu vân tay output, được cập nhật.
- Deployment Record được lưu.

## 5. Luồng chính

## Developer chọn **Deploy Application**.

## Developer chọn:

    * Environment: staging hoặc production.
    * Deployment target.
    * Phiên bản Application Definition cần deploy.

## IDP hiển thị phiên bản đang chạy trên environment/target đã chọn, các workload của phiên bản được chọn, Image Repository tương ứng và image version đang chạy nếu có.

   Ví dụ:

    ```text
    Environment: production    Đang chạy: phiên bản 4    Chọn deploy: phiên bản 5

    backend
    registry.company.local/shop-backend       (đang chạy: v1.4.2)

    frontend
    registry.company.local/shop-frontend      (đang chạy: v2.0.9)
    ```

## Developer chọn các workload cần deploy và chọn hoặc xác nhận image tag/version cho từng workload được chọn.

   Nếu phiên bản được chọn trùng phiên bản đang chạy, Developer có thể chọn toàn bộ hoặc chỉ một phần workload của application. Nếu chọn phiên bản khác phiên bản đang chạy, Developer phải deploy toàn bộ application.

   Ví dụ deploy toàn bộ:

    ```text
    backend
    registry.company.local/shop-backend:v1.4.3

    frontend
    registry.company.local/shop-frontend:v2.1.0
    ```

   Ví dụ chỉ deploy frontend:

    ```text
    frontend
    registry.company.local/shop-frontend:v2.1.0
    ```

   Image version cũng có thể đã được cung cấp từ CI/External Delivery Integration.

## Developer chọn các deployment context còn lại nếu cần.

   Ví dụ khi deploy lên Cloud:

    ```text
    Cloud Provider: **AWS**
    Region: ap-southeast-1
    ```

## IDP kiểm tra deployment input và Environment Configuration.

   IDP kiểm tra Environment Configuration của environment đã chọn khớp với phiên bản được deploy: mọi configuration bắt buộc của phiên bản đã có giá trị, và không biến nào còn tham chiếu output của thành phần không có trong phiên bản hoặc không được depends on trong phiên bản.

## IDP xây dựng dependency/resource graph cho deployment dựa trên:

    * Workload.
    * Resource.
    * Dependency.
    * Environment Configuration.
    * Resource Output Reference.
    * Workload Output Reference.
    * Deployment context.

## IDP xác định phạm vi deployment và chia tầng triển khai.

   Phạm vi gồm các workload được chọn và các resource mà chúng depends on trực tiếp. Workload mà chúng depends on nhưng không được chọn thì không được triển khai lại; output của các workload đó được lấy từ bản đang chạy.

   Tầng 0 gồm các thành phần trong phạm vi không phụ thuộc thành phần nào khác trong phạm vi; mỗi tầng sau gồm các thành phần chỉ phụ thuộc vào thành phần ở các tầng trước.

   Ví dụ deploy toàn bộ `shop-app`:

    ```text
    Tầng 0: postgresql
    Tầng 1: backend      (depends on postgresql)
    Tầng 2: frontend     (depends on backend)
    ```

   Ví dụ chỉ deploy frontend: phạm vi chỉ gồm frontend ở tầng 0; `backend.endpoint` được lấy từ backend đang chạy.

## IDP xác định các thành phần ngoài phạm vi có thể bị triển khai lại tự động nếu output mà chúng dùng thay đổi.

## IDP resolve Resource Definition phù hợp cho các resource trong phạm vi.

   Resource Definition có hai loại: loại do IDP quản lý hạ tầng (tạo, sửa, hủy) và loại trỏ tới resource có sẵn do platform khai báo để dùng chung (IDP chỉ đọc output, không đụng tới hạ tầng thật).

## IDP tìm Resource Instance hiện có của từng resource theo đúng chủ sở hữu: application + environment + resource requirement + deployment target.

## IDP xác định các infrastructure resource cần:

    * Tạo mới.
    * Cập nhật.
    * Hoặc tái sử dụng.

   Khi deploy một phiên bản mới, IDP đồng thời xác định các thành phần đang chạy trên environment/target nhưng không còn trong phiên bản:

    * Workload cần gỡ.
    * Resource do IDP quản lý cần hủy.
    * Resource dùng chung cần gỡ liên kết (không hủy hạ tầng thật).

## IDP hiển thị các tầng triển khai, infrastructure plan (kể cả các thành phần sẽ bị gỡ, hủy hoặc gỡ liên kết, kèm cảnh báo mất dữ liệu khi hủy resource), danh sách thành phần có thể bị triển khai lại và các infrastructure parameter mà Developer được phép override.

## Developer xác nhận và chọn **Deploy**.

## IDP ghi nhận xác nhận, lưu các giá trị override Developer đã chọn cùng một job triển khai, rồi trả lời Developer ngay.

   Developer không phải chờ việc triển khai hoàn tất; Developer theo dõi tiến trình và kết quả ở UC-04 – View Deployment Result.

## Deployment Worker của IDP nhận job và triển khai lần lượt từng tầng ở chế độ chạy nền. Với mỗi tầng:

   Với mỗi resource trong tầng:

    * Nếu Resource Definition thuộc loại do IDP quản lý: IDP reconcile infrastructure.
    * Nếu Resource Definition thuộc loại trỏ tới resource có sẵn: IDP không tạo hay sửa hạ tầng, chỉ liên kết Resource Instance của application tới resource đó.
    * Khi resource sẵn sàng, IDP thu thập Resource Output tương ứng.

   Với mỗi workload trong tầng:

    * IDP resolve Environment Configuration từ direct value, output của các tầng trước và output của các thành phần đang chạy ngoài phạm vi.
    * IDP tạo resolved application specification với image version, configuration và các dependency đã được resolve.
    * IDP sử dụng `score-k8s` để sinh base Kubernetes manifest.
    * IDP áp dụng target-specific manifest adaptation/patch nếu deployment target yêu cầu cấu hình riêng.
    * Environment Variable được chuyển thành ConfigMap hoặc Kubernetes configuration tương ứng.
    * Secret được chuyển thành Kubernetes Secret hoặc secret reference phù hợp.
    * IDP publish desired deployment state tới CD Integration; CD Integration chuyển desired state tới concrete CD implementation để triển khai xuống Kubernetes cluster.
    * IDP chờ tới khi pod của workload healthy.
    * IDP thu thập Workload Output tương ứng.

   Ví dụ:

```text
Tầng 0: reconcile postgresql
        → postgresql.host, postgresql.password

Tầng 1: backend.DB_HOST     → postgresql.host
        backend.DB_PASSWORD → postgresql.password
        deploy backend, chờ healthy
        → backend.endpoint

Tầng 2: frontend.BACKEND_URL → backend.endpoint
        deploy frontend, chờ healthy
```

## Sau mỗi tầng, IDP so sánh dấu vân tay output mới với lần triển khai trước của từng thành phần.

   Nếu output thay đổi, các thành phần depends on thành phần đó nhưng chưa nằm trong phạm vi được tự động thêm vào các tầng sau của chính deployment này, dùng image version đang chạy của chúng. Việc lan truyền tiếp tục cho tới khi không còn output thay đổi.

   Ví dụ chỉ deploy backend làm `backend.endpoint` thay đổi: frontend được thêm vào tầng kế tiếp với image đang chạy.

## Khi deploy một phiên bản mới, sau khi triển khai xong các tầng, IDP xử lý các thành phần không còn trong phiên bản.

    * Gỡ các workload không còn trong phiên bản.
    * Sau đó hủy resource do IDP quản lý, hoặc gỡ liên kết resource dùng chung, không còn trong phiên bản.

   Ví dụ phiên bản 5 không còn `worker` và `redis`: sau khi backend và frontend của phiên bản 5 đã healthy, IDP gỡ `worker`, rồi hủy `redis`.

## IDP lưu Deployment Record, bao gồm:

- Environment.
- Deployment target.
- Phiên bản Application Definition đã deploy.
- Image version thực tế của từng workload và workload nào được triển khai lại tự động.
- Các thành phần đã được gỡ, hủy hoặc gỡ liên kết.
- Infrastructure reference.
- Tiến trình theo từng tầng và từng thành phần.
- Trạng thái deployment.

## 6. Luồng ngoại lệ

### A1 – Deployment input hoặc dependency không hợp lệ

Deployment dừng nếu:

- Image version không hợp lệ.
- Configuration bắt buộc chưa được cấu hình.
- Environment Configuration không khớp với phiên bản được deploy, ví dụ biến còn tham chiếu output của thành phần không có trong phiên bản.
- Chọn deploy một phần application nhưng phiên bản được chọn khác phiên bản đang chạy.
- Dependency không thể resolve hoặc các quan hệ depends on tạo thành vòng.
- Workload phụ thuộc không nằm trong phạm vi deployment và chưa chạy healthy trên environment/target đã chọn.
- Resource Definition phù hợp không tồn tại.
- Resource Output hoặc Workload Output được tham chiếu không hợp lệ.

**IDP** hiển thị lỗi để Developer chỉnh sửa.

### A2 – Provisioning, delivery hoặc workload thất bại

Nếu infrastructure provisioning, manifest generation, CD delivery thất bại, workload không healthy tại một tầng, hoặc việc gỡ workload / hủy resource / gỡ liên kết resource thất bại:

- **IDP** ghi nhận deployment thất bại và không triển khai các tầng sau.
- **IDP** lưu tầng, thành phần liên quan, failed step và error summary.
- Developer có thể xem chi tiết trong UC-04 – View Deployment Result.

## 7. Dữ liệu chính

| Nhóm                   | Dữ liệu                                                 |
| ---------------------- | ------------------------------------------------------- |
| Deployment             | Application, phiên bản Application Definition, environment, deployment target |
| Deployment Execution Job | Deployment, giá trị override đã chọn, trạng thái job  |
| Workload Deployment    | Workload, image repository, image version, được chọn hay triển khai lại tự động |
| Deployment Context     | Cloud provider, region, target-specific input           |
| Deployment Graph       | Workload, resource, dependency, configuration reference, phạm vi, tầng triển khai |
| Resource Resolution    | Resource, Resource Definition                           |
| Resource Instance      | Application, environment, resource requirement, deployment target, trạng thái, dấu vân tay output |
| Resource Output        | Resource + output                                       |
| Workload Output        | Workload + output                                       |
| Resolved Configuration | Workload, variable/secret, resolved source              |
| Workload Instance      | Workload, environment, deployment target, image đang chạy, trạng thái, dấu vân tay output |
| Deployment Record      | Image version, target, status, infrastructure reference, tiến trình theo tầng/thành phần |

## 8. Quy tắc nghiệp vụ

- Image Repository thuộc Workload Definition.
- Image tag/version thuộc Deployment.
- Mỗi deployment phải lưu chính xác image được sử dụng cho từng workload.
- Image version có thể do Developer chọn hoặc được CI cung cấp.
- Mỗi deployment deploy đúng một phiên bản Application Definition vào đúng một environment (staging hoặc production) và một deployment target; environment khác không bị ảnh hưởng.
- Đổi sang phiên bản khác phiên bản đang chạy thì phải deploy toàn bộ application; deploy một phần chỉ dùng phiên bản đang chạy.
- Developer có thể deploy toàn bộ hoặc một phần workload của application.
- Phạm vi deployment gồm workload được chọn và resource mà chúng depends on trực tiếp; resource ngoài phạm vi không bị reconcile.
- Environment Configuration được kiểm tra khớp với phiên bản được deploy trước khi lập plan.
- Resource Instance thuộc về đúng một chủ: application + environment + resource requirement + deployment target. Resource chỉ được dùng lại khi khớp đủ bộ này.
- Resource dùng chung giữa các application hoặc environment chỉ được khai báo ở Resource Definition do platform quản lý; với loại này, IDP chỉ đọc output, không tạo, sửa hay hủy hạ tầng thật.
- Workload và resource không còn trong phiên bản được deploy bị gỡ, hủy hoặc gỡ liên kết trên environment/target đó; việc này hiện trong plan và cần Developer xác nhận. Resource dùng chung chỉ được gỡ liên kết.
- Sau khi Developer xác nhận, IDP trả lời ngay; việc triển khai do Deployment Worker chạy nền dựa trên job đã lưu.
- Dependency/resource graph được xây dựng tại thời điểm deployment dựa trên phiên bản Application Definition được chọn, Environment Configuration và deployment context.
- Các thành phần được triển khai theo thứ tự phụ thuộc: một thành phần chỉ được triển khai sau khi mọi thành phần nó depends on trong phạm vi đã sẵn sàng. Các thành phần không có quan hệ depends on với nhau là độc lập.
- Resource Definition được resolve theo resource requirement và deployment context.
- Infrastructure được reconcile thay vì luôn tạo mới.
- Resource Output chỉ được sử dụng sau khi resource tương ứng đã được resolve và sẵn sàng.
- Workload Output chỉ được sử dụng sau khi workload tương ứng healthy; với workload ngoài phạm vi, output được lấy từ bản đang chạy.
- Environment Configuration có thể phụ thuộc vào Resource Output hoặc Workload Output của thành phần mà workload depends on.
- Khi output của một thành phần thay đổi, các thành phần depends on nó được tự động triển khai lại trong cùng deployment với image version đang chạy. **IDP** chỉ lưu dấu vân tay output để so sánh, không lưu giá trị output.
- Developer không trực tiếp thao tác với Terraform module, Kubernetes ConfigMap, Kubernetes Secret hoặc Kubernetes manifest.
- `score-k8s` được sử dụng để sinh base Kubernetes manifest từ resolved application specification.
- Target-specific manifest adaptation/patch được áp dụng sau bước sinh base manifest khi cần.
- UC-03 sử dụng CD abstraction và không phụ thuộc trực tiếp vào Argo CD, Flux hoặc một sản phẩm CD cụ thể.

UC-04 – View Deployment Result

## Mục tiêu

Cho phép Developer xem tiến trình và kết quả của deployment, bao gồm image version thực tế đang được triển khai.

2. Actor

Primary Actor: Developer

## Tiền điều kiện

Application có ít nhất một deployment.

## Hậu điều kiện

Developer xem được trạng thái deployment.

Không làm thay đổi application hoặc infrastructure.

## Luồng chính

Developer mở application và chọn Deployments.

**IDP** hiển thị lịch sử deployment.

Developer chọn một deployment.

**IDP** hiển thị:

Environment.

Deployment target.

Phiên bản Application Definition đã deploy.

Image repository và version của từng workload, và workload nào được triển khai lại tự động.

Các thành phần đã được gỡ, hủy hoặc gỡ liên kết trong deployment, nếu có.

Deployment status.

Tiến trình theo từng tầng và từng thành phần.

Infrastructure status.

CD status.

Workload health.

Endpoint nếu có.

Ví dụ:

Deployment #42 Production — Phiên bản 5 — Succeeded

Tầng 0 postgresql ✓ Infrastructure Ready

Tầng 1 backend registry.company.local/shop-backend:v1.4.3 Healthy ✓ Configuration Resolved ✓ Manifest Generated ✓ CD Synced ✓ Application Ready

Tầng 2 frontend registry.company.local/shop-frontend:v2.1.0 Healthy (triển khai lại tự động) ✓ Configuration Resolved ✓ Manifest Generated ✓ CD Synced ✓ Application Ready

## Luồng ngoại lệ

A1 – Deployment thất bại

**IDP** hiển thị tầng, failed step, workload/resource liên quan và error summary.

A2 – Workload chưa Healthy

**IDP** hiển thị workload và image version gặp vấn đề.

## Quy tắc nghiệp vụ

UC-04 chỉ cung cấp trạng thái và kết quả deployment.

Phải hiển thị image/version thực tế đã sử dụng trong deployment.

Phải phân biệt workload được Developer chọn với workload được triển khai lại tự động.

Logs, metrics, traces và diagnostics sâu nằm ngoài phạm vi UC
# Use Case Realization cho Dev Portal

**Kết quả đã thống nhất cho Bước 1 đến Bước 3**

Tài liệu ghi lại đúng các nội dung đã được thống nhất trong quá trình thực hiện Use Case Realization. Tài liệu chỉ bao gồm ba bước đã hoàn thành.

# Quy trình Use Case Realization đã thực hiện

| **Bước** | **Nội dung**                                          |
|----------|-------------------------------------------------------|
| Bước 1   | Chốt responsibility của từng Use Case                 |
| Bước 2   | Xác định các System Operation chính của từng Use Case |
| Bước 3   | Xác định các thành phần tham gia                      |

# Bước 1 Chốt responsibility của từng Use Case

## UC 01 Create Configure Application

Chịu trách nhiệm khai báo cấu trúc logic của application: workload, resource requirement, dependency và các Environment Variable/Secret mà workload cần. Mỗi lần lưu, ghi Application Definition thành một phiên bản mới (không ghi đè phiên bản cũ, các thành phần giữ ID cố định qua phiên bản) và sinh/cập nhật application specification. Không deploy và không làm thay đổi environment nào.

## UC 02 Configure Application Environment

Chịu trách nhiệm khai báo giá trị cấu hình theo từng environment, bao gồm giá trị trực tiếp hoặc reference tới Resource Output / Workload Output. Không thực hiện deployment.

## UC 03 Deploy Application

Chịu trách nhiệm biến một phiên bản Application Definition + Environment Configuration của một environment (staging hoặc production) + deployment context + image version của các workload được chọn thành một deployment thực tế. Trách nhiệm chia làm hai phần:

- **Nhận yêu cầu deploy:** kiểm tra configuration khớp phiên bản, dựng dependency/resource graph, xác định phạm vi và chia tầng theo thứ tự phụ thuộc, tìm resource để dùng lại theo đúng chủ sở hữu, lập plan (gồm cả thành phần sẽ bị gỡ, hủy hoặc gỡ liên kết) cho Developer xem; khi Developer xác nhận thì lưu job triển khai và trả lời ngay.
- **Thực thi chạy nền (Deployment Worker):** với từng tầng reconcile infrastructure (hoặc chỉ liên kết resource dùng chung), resolve configuration, sinh manifest, gửi desired state sang CD system, chờ workload healthy và thu output; tự động triển khai lại thành phần phụ thuộc khi output thay đổi; gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản.

## UC 04 View Deployment Result

Chịu trách nhiệm đọc và hiển thị trạng thái/kết quả của deployment, gồm infrastructure, configuration resolution, CD status, workload health, endpoint và image version thực tế. Không thay đổi application hay infrastructure.

**Ranh giới giữa bốn Use Case**: UC-01 định nghĩa cần gì → UC-02 định nghĩa giá trị theo environment là gì → UC-03 quyết định triển khai chúng như thế nào → UC-04 cho biết kết quả ra sao.

# Bước 2 Xác định các System Operation chính

## UC 01 Create Configure Application

- **createApplication()** - Tạo một Application Definition mới.

- **updateApplication()** - Cập nhật thông tin application đã tồn tại.

- **addResourceRequirement()** - Thêm resource logic mà application cần, ví dụ PostgreSQL hoặc Redis.

- **addWorkload()** - Thêm workload và các thông tin như type, image repository, port.

- **defineConfigurationRequirement()** - Khai báo Environment Variable và Secret mà workload cần.

- **defineDependency()** - Khai báo quan hệ depends on giữa workload và resource/workload khác.

- **validateApplicationDefinition()** - Kiểm tra tính hợp lệ của workload, resource, dependency và configuration requirement.

- **saveApplicationDefinition()** - Lưu Application Definition thành một phiên bản mới; phiên bản cũ giữ nguyên, các thành phần giữ ID cố định qua phiên bản.

- **generateApplicationSpecification()** - Sinh hoặc cập nhật application specification, ví dụ score.yaml, từ Application Definition đã lưu.

Ở mức Use Case Realization, không tách nhỏ hơn thành các operation như addEnvironmentVariable(), addSecret() hoặc validatePort() để tránh làm sequence diagram quá vụn.

## UC 02 Configure Application Environment

- **selectEnvironment()** - Chọn environment cần cấu hình cho application.

- **loadConfigurationRequirements()** - Lấy danh sách Environment Variable và Secret đã được khai báo từ UC-01.

- **setDirectConfigurationValue()** - Gán giá trị trực tiếp cho Environment Variable hoặc Secret.

- **bindResourceOutput()** - Gán configuration vào một Resource Output của resource mà workload depends on, ví dụ DB_HOST → postgresql.host.

- **bindWorkloadOutput()** - Gán configuration vào một Workload Output của workload mà workload chứa biến depends on, ví dụ BACKEND_URL → backend.endpoint.

- **validateEnvironmentConfiguration()** - Kiểm tra giá trị, resource, workload và output reference có hợp lệ hay không, gồm việc output được tham chiếu thuộc thành phần mà workload depends on.

- **saveEnvironmentConfiguration()** - Lưu configuration riêng cho environment đã chọn.

UC-02 chỉ lưu value hoặc reference, chưa resolve giá trị thật của Resource Output / Workload Output. Việc resolve thuộc UC-03 khi deployment thực sự diễn ra.

## UC 03 Deploy Application

- **createDeployment()** - Tạo deployment mới từ application, phiên bản Application Definition được chọn, environment (staging hoặc production), deployment target và image version của các workload được chọn.

- **validateDeploymentInput()** - Kiểm tra image version, deployment context, Environment Configuration khớp với phiên bản được chọn, và luật deploy một phần chỉ dùng phiên bản đang chạy.

- **buildDeploymentGraph()** - Dựng dependency/resource graph từ workload, resource, dependency, configuration reference và deployment context; phát hiện các quan hệ depends on tạo thành vòng.

- **planDeploymentWaves()** - Xác định phạm vi deployment (workload được chọn và resource mà chúng depends on trực tiếp), chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy và liệt kê thành phần có thể bị triển khai lại.

- **resolveResourceDefinitions()** - Chọn Resource Definition phù hợp cho từng resource trong phạm vi deployment, gồm cả việc resource đó thuộc loại do IDP quản lý hạ tầng hay loại trỏ tới resource dùng chung có sẵn.

- **planInfrastructureChanges()** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement + deployment target) và xác định resource nào cần tạo mới, cập nhật, tái sử dụng hoặc liên kết tới resource dùng chung; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **loadInfrastructureOverrides()** - Lấy các infrastructure parameter mà Developer được phép override.

- **applyInfrastructureOverrides()** - Ghi nhận các giá trị override mà Developer lựa chọn.

- **confirmDeployment()** - Xác nhận deployment sau khi Developer kiểm tra các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị triển khai lại và các override. Operation chỉ đổi trạng thái deployment sang đã xác nhận và tạo job triển khai (kèm các giá trị override đã chọn) trong cùng một lần lưu, rồi trả lời ngay; không tự thực thi việc triển khai.

- **reconcileInfrastructure()** - Do Deployment Worker gọi. Thực thi việc tạo/cập nhật infrastructure của các resource do IDP quản lý trong một tầng thông qua provisioner phù hợp, hoặc liên kết Resource Instance tới resource dùng chung có sẵn mà không đụng hạ tầng thật; với thành phần không còn trong phiên bản, thực thi việc hủy resource hoặc gỡ liên kết resource dùng chung.

- **collectResourceOutputs()** - Thu thập Resource Output sau khi infrastructure resource sẵn sàng.

- **resolveEnvironmentConfiguration()** - Resolve configuration của các workload trong một tầng từ direct value, Resource Output và Workload Output.

- **generateResolvedApplicationSpecification()** - Tạo resolved application specification chứa image version, configuration và dependency đã resolve.

- **generateKubernetesManifest()** - Sinh base Kubernetes manifest từ resolved specification bằng score-k8s.

- **adaptManifestForTarget()** - Áp dụng target-specific patch/adaptation cho Kubernetes manifest nếu deployment target yêu cầu.

- **materializeEnvironmentConfiguration()** - Chuyển Environment Variable đã resolve thành Kubernetes configuration, ví dụ ConfigMap hoặc cấu hình tương ứng.

- **materializeSecretConfiguration()** - Chuyển Secret đã resolve thành Kubernetes Secret hoặc secret reference phù hợp.

- **publishDesiredDeploymentState()** - Gửi desired deployment state của các workload trong một tầng sang CD abstraction; khi deploy phiên bản mới, desired state không còn các workload bị gỡ để CD system gỡ chúng khỏi cluster.

- **waitForWorkloadsHealthy()** - Chờ tới khi pod của các workload trong tầng healthy trên deployment target.

- **collectWorkloadOutputs()** - Thu thập Workload Output từ workload vừa healthy, hoặc từ workload đang chạy ngoài phạm vi deployment.

- **propagateOutputChanges()** - So sánh dấu vân tay output mới với lần triển khai trước; nếu thay đổi, thêm các thành phần depends on vào các tầng sau với image version đang chạy.

- **saveDeploymentRecord()** - Lưu Deployment Record, image version, infrastructure reference, tiến trình theo tầng/thành phần và trạng thái thực thi.

Chuỗi chính gồm hai phần:

- **Trong request của Developer:** tạo deployment (chọn phiên bản) → kiểm tra input và configuration khớp phiên bản → dựng graph → chia tầng → resolve resource → plan/override infra (gồm gỡ/hủy/gỡ liên kết) → xác nhận: lưu job và trả lời ngay.
- **Deployment Worker chạy nền:** với mỗi tầng: reconcile infra hoặc liên kết resource dùng chung → thu Resource Output → resolve config → sinh resolved spec → sinh base manifest → adapt theo target → materialize config/secret → publish sang CD → chờ healthy → thu Workload Output → lan truyền thay đổi output; sau các tầng: gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản → lưu deployment record.

## UC 04 View Deployment Result

- **listDeployments()** - Lấy danh sách lịch sử deployment của application.

- **getDeploymentDetail()** - Lấy thông tin chi tiết của một deployment cụ thể.

- **getDeploymentProgress()** - Lấy tiến trình thực thi của deployment theo từng tầng, từng thành phần và từng bước.

- **getInfrastructureStatus()** - Lấy trạng thái của infrastructure liên quan đến deployment.

- **getCDStatus()** - Lấy trạng thái đồng bộ/triển khai từ CD system thông qua CD abstraction.

- **getWorkloadStatus()** - Lấy trạng thái health của từng workload trong deployment.

- **getDeploymentEndpoints()** - Lấy endpoint được expose sau deployment nếu có.

- **getDeploymentImages()** - Lấy image repository và image version thực tế đã dùng cho từng workload.

- **getDeploymentFailureDetail()** - Lấy failed step, workload/resource liên quan và error summary khi deployment thất bại.

UC-04 là read-only: đọc dữ liệu từ Deployment Record và các nguồn trạng thái liên quan, sau đó tổng hợp để hiển thị cho Developer. UC-04 không trigger reconcile, không sửa infrastructure và không redeploy.

# Bước 3 Xác định các thành phần tham gia

Các thành phần được phân theo bảy nhóm: Boundary/UI, Application services, Domain components, Integration abstractions, Integration implementations, External systems và Persistence. Trong từng Use Case, tài liệu chỉ hiển thị những nhóm có thành phần tham gia.

## UC 01 Create Configure Application

### Boundary/UI

- **Web UI** - Cho Developer khai báo application, workload, resource, dependency và configuration requirement.

- **Application API / Controller** - Nhận request từ UI, validate ở mức request và điều phối sang application service.

### Application services

- **Application Service** - Chịu trách nhiệm xử lý use case tạo/cập nhật Application Definition.

### Domain components

- **Application Definition Validator** - Kiểm tra tính hợp lệ của workload, resource, dependency, port, image repository và configuration requirement.

- **Application Specification Generator** - Chuyển Application Definition thành application specification tương ứng, ví dụ score.yaml.

### Persistence

- **Application Repository** - Lưu và đọc Application Definition theo phiên bản: mỗi lần lưu thêm một phiên bản mới, không sửa hay xóa phiên bản cũ; giữ ID cố định của các thành phần qua phiên bản.

- **Specification Repository / Config Repo Service** - Lưu application specification đã sinh nếu hệ thống cần persist hoặc version hóa artifact này.

Luồng trách nhiệm: Web UI → API/Controller → Application Service → Validator → Repository + Specification Generator.

UC-01 chưa cần Deployment Orchestrator, Resource Definition Resolver, Infrastructure Reconciler, CD Integration hoặc Kubernetes Cluster vì use case chỉ dừng ở việc định nghĩa application và sinh specification, chưa deploy.

## UC 02 Configure Application Environment

### Boundary/UI

- **Web UI** - Cho Developer chọn environment, nhập giá trị configuration và chọn nguồn từ Resource Output hoặc Workload Output.

- **Environment Configuration API / Controller** - Nhận request từ UI, kiểm tra request cơ bản và chuyển sang application service tương ứng.

### Application services

- **Environment Configuration Service** - Điều phối toàn bộ use case cấu hình application theo environment.

### Domain components

- **Resource Output Catalog / Resource Definition Query** - Cung cấp danh sách output hợp lệ mà một resource có thể expose để Developer lựa chọn.

- **Workload Output Catalog** - Cung cấp danh sách output mà workload có thể expose, ví dụ endpoint.

- **Environment Configuration Validator** - Kiểm tra direct value, resource reference, workload reference và output được chọn có hợp lệ hay không, gồm việc output thuộc thành phần mà workload depends on.

### Integration abstractions

- **Secret Store / Secret Management Adapter** - Lưu giá trị Secret theo cơ chế bảo mật, tránh lưu plaintext trực tiếp trong configuration database.

Secret backend implementation cụ thể chưa được chốt, vì vậy tài liệu chưa liệt kê thành phần thuộc nhóm Integration implementations cho UC-02.

### Persistence

- **Application Query / Application Repository** - Lấy danh sách workload, Environment Variable, Secret và dependency đã được khai báo từ UC-01.

- **Environment Configuration Repository** - Lưu configuration và các reference riêng theo từng environment.

Luồng trách nhiệm: Web UI → Configuration API → Environment Configuration Service → Application/Output Catalog → Validator → Secret Store + Environment Configuration Repository.

UC-02 chưa cần Configuration Resolver. Hệ thống chỉ lưu reference như DB_HOST → postgresql.host; giá trị thật chỉ được resolve trong UC-03 khi deploy.

## UC 03 Deploy Application

### Boundary/UI

- **Web UI** - Cho Developer chọn environment, deployment target, workload cần deploy và image version, xem các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị triển khai lại, nhập override và xác nhận deploy.

- **Deployment API / Controller** - Nhận request từ UI, validate ở mức request và chuyển sang Deployment Orchestrator.

### Application services

- **Deployment Orchestrator** - Điều phối phần xử lý trong request của Developer: tạo deployment theo phiên bản được chọn, kiểm tra input, dựng graph, chia tầng, lập plan; khi Developer xác nhận thì đổi trạng thái deployment và lưu job triển khai trong cùng một lần lưu rồi trả lời ngay. Không tự thực thi việc triển khai.

- **Deployment Worker** - Tiến trình chạy nền nhận job triển khai đã lưu và điều phối phần thực thi: triển khai lần lượt từng tầng tới khi mọi workload trong phạm vi healthy, lan truyền khi output thay đổi, gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản, và cập nhật trạng thái deployment.

### Domain components

- **Deployment Graph Builder** - Dựng dependency/resource graph từ Application Definition, Environment Configuration và deployment context; phát hiện các quan hệ depends on tạo thành vòng.

- **Deployment Wave Planner** - Xác định phạm vi deployment, chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy, và lan truyền khi output thay đổi bằng cách thêm thành phần phụ thuộc vào các tầng sau.

- **Resource Definition Resolver** - Chọn Resource Definition phù hợp cho từng logical resource dựa trên type và deployment context; phân biệt definition do IDP quản lý hạ tầng với definition trỏ tới resource dùng chung có sẵn.

- **Infrastructure Planner** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement + deployment target), so sánh desired state với resource hiện tại để xác định cần create, update, reuse hay liên kết resource dùng chung; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **Infrastructure Reconciler** - Điều phối việc reconcile infrastructure của từng tầng theo plan đã xác định: tạo/sửa resource do IDP quản lý, chỉ liên kết (không đụng hạ tầng thật) với resource dùng chung, và thực hiện hủy resource hoặc gỡ liên kết resource không còn trong phiên bản.

- **Resource Output Resolver / Collector** - Thu thập output từ infrastructure đã provision, ví dụ host, port, username, password.

- **Workload Output Collector** - Thu thập output từ workload đã healthy hoặc đang chạy, ví dụ endpoint. Cơ chế đọc output cụ thể chưa được chốt.

- **Environment Configuration Resolver** - Resolve direct value, Resource Output reference và Workload Output reference thành configuration thực tế cho các workload trong một tầng.

- **Resolved Specification Generator** - Tạo resolved application specification chứa image version, dependency và configuration đã resolve.

- **Manifest Generator / Score Renderer** - Gọi score-k8s để sinh base Kubernetes manifest từ resolved specification.

- **Target Manifest Adapter** - Áp dụng patch/adaptation riêng theo deployment target.

- **Environment Configuration Materializer** - Chuyển Environment Variable đã resolve thành Kubernetes configuration tương ứng, ví dụ ConfigMap.

- **Secret Materializer** - Chuyển Secret thành Kubernetes Secret hoặc secret reference phù hợp mà không làm lộ plaintext.

### Integration abstractions

- **Provisioner Adapter / Provisioner Interface** - Abstraction cho cơ chế provision infrastructure; implementation cụ thể có thể gọi Terraform/OpenTofu/module tương ứng.

- **CD Integration / CD Provider Interface** - Abstraction để publish desired deployment state mà không phụ thuộc trực tiếp vào Argo CD, Flux hay implementation cụ thể.

- **Workload Status Provider / Kubernetes Adapter** - Kiểm tra workload đã healthy chưa và đọc dữ liệu runtime của workload phục vụ Workload Output Collector.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể của CD abstraction, ví dụ Argo CD Adapter hoặc Flux Adapter.

### External systems

- **Terraform/OpenTofu Runner** - Thực thi Terraform/OpenTofu module để tạo hoặc cập nhật infrastructure theo yêu cầu từ Provisioner Adapter.

- **score-k8s** - Sinh base Kubernetes manifest từ resolved application specification.

- **CD System** - Hệ thống CD bên ngoài, ví dụ Argo CD hoặc Flux, nhận desired deployment state và đồng bộ xuống Kubernetes.

- **Kubernetes Cluster** - Deployment target cuối nơi workload thực sự chạy; cung cấp trạng thái health và dữ liệu runtime của workload.

### Persistence

- **Application Repository** - Đọc phiên bản Application Definition được chọn (đã lưu từ UC-01).

- **Environment Configuration Repository** - Đọc configuration/reference đã lưu từ UC-02.

- **Resource Instance Repository** - Lưu/đọc Resource Instance theo đúng chủ sở hữu (application + environment + resource requirement + deployment target): trạng thái, reference, liên kết tới resource dùng chung và dấu vân tay output, phục vụ reconcile/reuse, gỡ/hủy và lan truyền thay đổi output.

- **Workload Instance Repository** - Lưu/đọc trạng thái hiện hành của từng workload theo environment và deployment target: image đang chạy, trạng thái health và dấu vân tay output.

- **Deployment Repository** - Lưu deployment (kèm phiên bản Application Definition), job triển khai cùng giá trị override đã chọn, Deployment Record, image version, target, infrastructure reference, tiến trình theo tầng/thành phần, status và lỗi nếu có.

Luồng responsibility:

- **Trong request:** Web UI → Deployment API → Deployment Orchestrator → Graph Builder → Wave Planner → Resource Definition Resolver → Infrastructure Planner → Deployment Repository (lưu deployment; khi xác nhận thì lưu job) → Web UI.
- **Chạy nền:** Deployment Worker → (với mỗi tầng) Infrastructure Reconciler → Provisioner → Terraform/OpenTofu Runner → Resource Output Collector → Configuration Resolver → Resolved Spec Generator → Score Renderer → score-k8s → Target Adapter → Config/Secret Materializer → CD Integration → Concrete CD Provider → CD System → Kubernetes → Workload Status Provider → Workload Output Collector → Wave Planner (lan truyền) → (sau các tầng) gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản → Resource/Workload Instance Repository → Deployment Repository.

Deployment Orchestrator và Deployment Worker chỉ điều phối. Các việc dựng graph, chia tầng và lan truyền thay đổi output, resolve Resource Definition, reconcile infrastructure, resolve configuration, sinh manifest, giao tiếp với CD và thu output nằm ở các component riêng.

## UC 04 View Deployment Result

### Boundary/UI

- **Web UI** - Hiển thị lịch sử deployment, trạng thái từng bước, workload health, infrastructure status, image version, endpoint và lỗi nếu có.

- **Deployment Query API / Controller** - Nhận request đọc dữ liệu từ UI và chuyển sang query service.

### Application services

- **Deployment Query Service** - Điều phối việc tổng hợp dữ liệu cần hiển thị cho một deployment.

### Domain components

- **Deployment Result Aggregator** - Tổng hợp Deployment Record + infrastructure status + CD status + workload health thành một view model thống nhất cho UI.

### Integration abstractions

- **CD Integration / CD Status Provider** - Lấy trạng thái deployment/sync từ concrete CD implementation thông qua abstraction.

- **Workload Status Provider / Kubernetes Adapter** - Lấy workload health, pod/deployment status và endpoint từ Kubernetes cluster.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể để truy vấn Argo CD, Flux hoặc CD system khác.

### External systems

- **CD System** - Hệ thống CD bên ngoài, ví dụ Argo CD hoặc Flux, cung cấp trạng thái deployment và trạng thái đồng bộ.

- **Kubernetes Cluster** - Nguồn trạng thái runtime thực tế của workload.

### Persistence

- **Deployment Repository** - Đọc Deployment Record, image version, target, trạng thái từng bước và error summary.

- **Resource Instance Repository** - Đọc thông tin infrastructure reference và trạng thái resource liên quan đến deployment.

- **Workload Instance Repository** - Đọc image version đang chạy của từng workload theo environment và deployment target, để cho biết deployment đang xem có còn là bản đang chạy hay đã được thay thế.

Luồng responsibility: Web UI → Deployment Query API → Deployment Query Service → Deployment Repository + Resource Repository + Workload Instance Repository + CD Status Provider → Concrete CD Provider → CD System + Kubernetes Adapter → Kubernetes Cluster → Result Aggregator → Web UI.

UC-04 không dùng Deployment Orchestrator để thực hiện hành động. UC-04 đi theo query path riêng vì chỉ đọc và tổng hợp trạng thái, không reconcile infrastructure hoặc trigger deployment.

