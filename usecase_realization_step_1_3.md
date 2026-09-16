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

Cho phép Developer triển khai một phiên bản Application Definition cụ thể, cho toàn bộ hoặc một phần application, lên một environment (staging hoặc production) và một nơi triển khai (deployment target). Thử một phiên bản ở staging xong thì deploy cùng phiên bản đó lên production.

Có hai loại nơi triển khai:

- **Cloud**, ví dụ AWS region ap-southeast-1: IDP dựng VPC và cụm Kubernetes (ví dụ EKS) riêng cho mỗi application + environment.
- **Cụm Kubernetes nội bộ**: cụm đã có sẵn và được platform khai báo trong catalog; IDP chỉ kết nối vào để deploy, không tạo và không xóa cụm.

Image tag/version được xác định tại thời điểm deployment, không phải khi định nghĩa application trong UC-01.

**IDP** chịu trách nhiệm:

- Kiểm tra Environment Configuration của environment khớp với phiên bản Application Definition được deploy.
- Xác định phạm vi deployment: các workload được chọn, các resource mà chúng depends on trực tiếp, và cụm Kubernetes/network mà các resource đó cần.
- Xác định dependency và chia các thành phần trong phạm vi thành các tầng triển khai theo thứ tự phụ thuộc.
- Resolve Resource Definition phù hợp trong phiên bản catalog được chọn, theo deployment context.
- Tìm resource để dùng lại theo đúng chủ sở hữu (application + environment + resource requirement + deployment target); resource dùng chung chỉ được dùng khi Resource Definition khai báo trỏ tới resource có sẵn.
- Reconcile infrastructure, gồm cả VPC và cụm Kubernetes khi deploy lên cloud; khi deploy lên cụm nội bộ thì chỉ kết nối vào cụm có sẵn.
- Bảo đảm application có nơi chứa desired state của riêng nó (Delivery Repository) trước khi giao hàng; tạo nơi đó ở lần deploy đầu tiên nếu chưa có.
- Gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản được deploy.
- Nhận yêu cầu deploy và trả lời Developer ngay; việc triển khai chạy nền.
- Triển khai lần lượt từng tầng: resolve Environment Configuration từ output của các tầng trước, sinh Kubernetes manifest, chuyển desired deployment state cho hệ thống Continuous Delivery và chờ workload healthy.
- Thu thập Resource Output và Workload Output sau khi thành phần tương ứng sẵn sàng.
- Tự động làm lại các resource và workload phụ thuộc khi output mà chúng dùng bị thay đổi.

## 2. Actor

**Primary Actor:** Developer

## 3. Tiền điều kiện

- Phiên bản Application Definition được chọn tồn tại, hợp lệ và các quan hệ depends on không tạo thành vòng.
- Workload cần deploy có Image Repository.
- Environment Configuration cần thiết của environment đã được cấu hình và khớp với phiên bản được chọn.
- Phiên bản catalog được chọn tồn tại và có Resource Definition, provisioner phù hợp cho các resource, cụm Kubernetes và network (nếu cần) của nơi triển khai.
- Nơi triển khai được hỗ trợ: phiên bản catalog được chọn có công thức `k8s-cluster` phù hợp. Với cụm nội bộ, cụm đã có sẵn.
- Nếu chỉ deploy một phần application, phiên bản Application Definition và phiên bản catalog được chọn phải trùng bản đang chạy trên environment và deployment target đó.
- Mọi workload mà các workload được chọn depends on nhưng không nằm trong phạm vi deployment đang chạy healthy trên cùng environment và deployment target.

## 4. Hậu điều kiện

- Deployment ghi nhận phiên bản Application Definition và phiên bản catalog đã được deploy.
- Deployment ghi nhận chính xác image version của từng workload được triển khai, gồm cả workload được triển khai lại tự động.
- Infrastructure trong phạm vi deployment, gồm cả cụm Kubernetes và network, đã được reconcile (hoặc liên kết tới cụm nội bộ có sẵn) và sẵn sàng.
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
    * Nơi triển khai: cloud (ví dụ AWS) hoặc một cụm Kubernetes nội bộ mà platform đã khai báo.
    * Phiên bản Application Definition cần deploy.
    * Phiên bản catalog.

   IDP chọn sẵn phiên bản catalog đang chạy trên environment/nơi triển khai đã chọn và báo nếu có phiên bản mới hơn; nếu chưa có gì chạy thì chọn sẵn phiên bản mới nhất. Khi promote lên production, Developer dùng cả phiên bản Application Definition lẫn phiên bản catalog đang chạy ở staging.

## IDP hiển thị phiên bản Application Definition và phiên bản catalog đang chạy trên environment/target đã chọn, các workload của phiên bản được chọn, Image Repository tương ứng và image version đang chạy nếu có.

   Ví dụ:

    ```text
    Environment: production    Đang chạy: phiên bản 4, catalog v2    Chọn deploy: phiên bản 5, catalog v2

    backend
    registry.company.local/shop-backend       (đang chạy: v1.4.2)

    frontend
    registry.company.local/shop-frontend      (đang chạy: v2.0.9)
    ```

## Developer chọn các workload cần deploy và chọn hoặc xác nhận image tag/version cho từng workload được chọn.

   Nếu phiên bản Application Definition và phiên bản catalog được chọn đều trùng bản đang chạy, Developer có thể chọn toàn bộ hoặc chỉ một phần workload của application. Nếu một trong hai khác bản đang chạy, Developer phải deploy toàn bộ application.

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

## Developer chọn các deployment context còn lại nếu cần. Deploy lên cụm nội bộ thì không cần chọn thêm.

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
    * Phiên bản catalog.

   Ngoài các thành phần Developer khai báo ở UC-01, IDP tự thêm hạ tầng của nơi triển khai vào graph:

    * Cụm Kubernetes (`k8s-cluster`): mọi workload chạy trên cụm này.
    * Những gì Resource Definition khai báo là cần thêm (`requires`), ví dụ trên AWS cụm EKS và Aurora cần `network` (VPC).

## IDP resolve Resource Definition phù hợp trong phiên bản catalog được chọn cho từng resource, cụm Kubernetes và network trong graph.

   Resource Definition có hai loại: loại do IDP quản lý hạ tầng (tạo, sửa, hủy) và loại trỏ tới thứ có sẵn do platform khai báo (IDP chỉ dùng, không đụng tới hạ tầng thật), ví dụ database dùng chung hoặc cụm nội bộ.

   Ví dụ:

    ```text
    Cloud (AWS):   k8s-cluster → eks-cluster        (IDP quản lý, cần network)
                   network     → aws-network        (IDP quản lý)
                   postgresql  → aurora-postgresql  (IDP quản lý, cần network)

    Cụm nội bộ:    k8s-cluster → cụm nội bộ có sẵn  (trỏ tới cụm có sẵn)
                   postgresql  → postgres-k8s       (IDP quản lý, chạy trong cụm)
    ```

## IDP xác định phạm vi deployment và chia tầng triển khai.

   Phạm vi gồm các workload được chọn, các resource mà chúng depends on trực tiếp, và cụm Kubernetes/network mà các resource đó cần. Workload mà chúng depends on nhưng không được chọn thì không được triển khai lại; output của các workload đó được lấy từ bản đang chạy.

   Tầng 0 gồm các thành phần trong phạm vi không phụ thuộc thành phần nào khác trong phạm vi; mỗi tầng sau gồm các thành phần chỉ phụ thuộc vào thành phần ở các tầng trước.

   Ví dụ deploy toàn bộ `shop-app` lên AWS:

    ```text
    Tầng 0: network      (VPC)
    Tầng 1: k8s-cluster  (EKS, cần network)
            postgresql   (Aurora, cần network)
    Tầng 2: backend      (depends on postgresql, chạy trên k8s-cluster)
    Tầng 3: frontend     (depends on backend)
    ```

   Ví dụ chỉ deploy frontend: phạm vi gồm frontend, cụm Kubernetes và network mà cụm cần; `backend.endpoint` được lấy từ backend đang chạy. Cụm và network không có gì thay đổi thì chỉ được dùng lại.

## IDP tìm Resource Instance hiện có của từng resource, cụm Kubernetes và network theo đúng chủ sở hữu: application + environment + resource requirement + deployment target.

## IDP xác định các infrastructure resource cần:

    * Tạo mới.
    * Cập nhật.
    * Tái sử dụng.
    * Hoặc liên kết tới thứ có sẵn (resource dùng chung, cụm nội bộ).

   Resource cần cập nhật khi đầu vào hiện tại khác đầu vào lần apply gần nhất: công thức trong phiên bản catalog, tham số, hoặc output của những gì resource đó cần.

   Khi deploy một phiên bản mới, IDP đồng thời xác định các thành phần đang chạy trên environment/target nhưng không còn trong phiên bản:

    * Workload cần gỡ.
    * Resource do IDP quản lý cần hủy.
    * Resource dùng chung cần gỡ liên kết (không hủy hạ tầng thật).

## IDP xác định các thành phần có thể bị làm lại tự động nếu output mà chúng dùng thay đổi.

   Chỉ tính từ các thành phần sẽ được tạo, cập nhật hoặc deploy; thành phần được tái sử dụng không đổi output.

   Ví dụ: `network` sẽ được cập nhật thì `k8s-cluster`, `postgresql` và các workload chạy trên cụm có thể bị làm lại.

## IDP hiển thị các tầng triển khai, infrastructure plan (kể cả các thành phần sẽ bị gỡ, hủy hoặc gỡ liên kết, kèm cảnh báo mất dữ liệu khi hủy resource), danh sách thành phần có thể bị làm lại và các infrastructure parameter mà Developer được phép override.

## Developer xác nhận và chọn **Deploy**.

## IDP ghi nhận xác nhận, lưu các giá trị override Developer đã chọn cùng một job triển khai, rồi trả lời Developer ngay.

   Developer không phải chờ việc triển khai hoàn tất; Developer theo dõi tiến trình và kết quả ở UC-04 – View Deployment Result.

## Deployment Worker của IDP nhận job và triển khai lần lượt từng tầng ở chế độ chạy nền. Với mỗi tầng:

   Với mỗi resource trong tầng (kể cả VPC và cụm Kubernetes):

    * Nếu Resource Definition thuộc loại do IDP quản lý: IDP reconcile infrastructure; input gồm output của những gì resource đó cần, ví dụ cụm EKS nhận subnet của network.
    * Nếu Resource Definition thuộc loại trỏ tới thứ có sẵn (resource dùng chung hoặc cụm nội bộ): IDP không tạo hay sửa hạ tầng, chỉ liên kết Resource Instance của application tới thứ đó.
    * Khi resource sẵn sàng, IDP thu thập Resource Output tương ứng.

   Với mỗi workload trong tầng:

    * IDP resolve Environment Configuration từ direct value, output của các tầng trước và output của các thành phần đang chạy ngoài phạm vi.
    * IDP tạo resolved application specification với image version, configuration và các dependency đã được resolve.
    * IDP sử dụng `score-k8s` để sinh base Kubernetes manifest.
    * IDP áp dụng target-specific manifest adaptation/patch nếu deployment target yêu cầu cấu hình riêng.
    * Environment Variable được chuyển thành ConfigMap hoặc Kubernetes configuration tương ứng.
    * Secret được chuyển thành Kubernetes Secret hoặc secret reference phù hợp.
    * IDP publish desired deployment state tới CD Integration; CD Integration chuyển desired state tới concrete CD implementation để triển khai xuống cụm Kubernetes của nơi triển khai. Thông tin kết nối cụm lấy từ output của `k8s-cluster`.
    * Desired state được ghi vào **Delivery Repository của chính application đó**. Lần đầu application được deploy, IDP tạo nơi chứa này, sinh một cặp khóa riêng cho application (khóa ghi cho IDP, khóa đọc cho CD system) và ghi nhận lại để các lần sau dùng tiếp.
    * IDP chờ tới khi pod của workload healthy.
    * IDP thu thập Workload Output tương ứng.

   Ví dụ deploy toàn bộ `shop-app` lên AWS:

```text
Tầng 0: reconcile network
        → network.subnet_ids

Tầng 1: reconcile k8s-cluster (dùng network.subnet_ids)
        → thông tin kết nối cụm
        reconcile postgresql (dùng network.subnet_ids)
        → postgresql.host, postgresql.password

Tầng 2: backend.DB_HOST     → postgresql.host
        backend.DB_PASSWORD → postgresql.password
        deploy backend lên cụm, chờ healthy
        → backend.endpoint

Tầng 3: frontend.BACKEND_URL → backend.endpoint
        deploy frontend, chờ healthy
```

## Sau mỗi tầng, IDP so sánh dấu vân tay output mới với lần triển khai trước của từng thành phần.

   Nếu output thay đổi, mọi thành phần dựa trên thành phần đó được làm lại trong chính deployment này, kể cả khi chưa nằm trong phạm vi:

    * Resource cần thành phần đó (qua `requires`) được reconcile lại với output mới.
    * Workload depends on thành phần đó, hoặc chạy trên cụm đó, được thêm vào các tầng sau với image version đang chạy.

   Việc lan truyền tiếp tục cho tới khi không còn output thay đổi.

   Ví dụ chỉ deploy backend làm `backend.endpoint` thay đổi: frontend được thêm vào tầng kế tiếp với image đang chạy. Ví dụ network đổi subnet: cụm EKS và Aurora được reconcile lại ở tầng kế tiếp, rồi các workload dựa trên chúng được triển khai lại.

## Khi deploy một phiên bản mới, sau khi triển khai xong các tầng, IDP xử lý các thành phần không còn trong phiên bản.

    * Gỡ các workload không còn trong phiên bản.
    * Sau đó hủy resource do IDP quản lý, hoặc gỡ liên kết resource dùng chung, không còn trong phiên bản.

   Ví dụ phiên bản 5 không còn `worker` và `redis`: sau khi backend và frontend của phiên bản 5 đã healthy, IDP gỡ `worker`, rồi hủy `redis`.

## IDP lưu Deployment Record, bao gồm:

- Environment.
- Deployment target.
- Phiên bản Application Definition đã deploy.
- Phiên bản catalog đã dùng.
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
- Phiên bản catalog được chọn không tồn tại.
- Chọn deploy một phần application nhưng phiên bản Application Definition hoặc phiên bản catalog được chọn khác bản đang chạy.
- Dependency không thể resolve hoặc các quan hệ tạo thành vòng (kể cả vòng do `requires` trong catalog).
- Workload phụ thuộc không nằm trong phạm vi deployment và chưa chạy healthy trên environment/target đã chọn.
- Phiên bản catalog được chọn không có Resource Definition phù hợp, kể cả công thức cụm Kubernetes cho nơi triển khai đã chọn.
- Resource Output hoặc Workload Output được tham chiếu không hợp lệ.

**IDP** hiển thị lỗi để Developer chỉnh sửa.

### A2 – Provisioning, delivery hoặc workload thất bại

Nếu infrastructure provisioning (kể cả VPC, cụm Kubernetes), việc chuẩn bị Delivery Repository của application, manifest generation, CD delivery thất bại, workload không healthy tại một tầng, hoặc việc gỡ workload / hủy resource / gỡ liên kết resource thất bại:

- **IDP** ghi nhận deployment thất bại và không triển khai các tầng sau.
- **IDP** lưu tầng, thành phần liên quan, failed step và error summary.
- Developer có thể xem chi tiết trong UC-04 – View Deployment Result.

## 7. Dữ liệu chính

| Nhóm                   | Dữ liệu                                                 |
| ---------------------- | ------------------------------------------------------- |
| Deployment             | Application, phiên bản Application Definition, phiên bản catalog, environment, deployment target |
| Deployment Execution Job | Deployment, giá trị override đã chọn, trạng thái job  |
| Workload Deployment    | Workload, image repository, image version, được chọn hay triển khai lại tự động |
| Deployment Context     | Cloud provider, region, target-specific input           |
| Catalog Version        | Số phiên bản, các Resource Definition của phiên bản     |
| Platform Requirement   | Application, loại (ví dụ `k8s-cluster`, `network`)      |
| Deployment Graph       | Workload, resource, cụm Kubernetes/network do IDP thêm, dependency, configuration reference, phạm vi, tầng triển khai |
| Resource Resolution    | Resource, cụm Kubernetes hoặc network; Resource Definition trong phiên bản catalog |
| Resource Instance      | Application, environment, resource requirement hoặc platform requirement, deployment target, trạng thái, dấu vân tay output, dấu vân tay đầu vào lần apply gần nhất |
| Resource Output        | Resource + output                                       |
| Workload Output        | Workload + output                                       |
| Resolved Configuration | Workload, variable/secret, resolved source              |
| Workload Instance      | Workload, environment, deployment target, image đang chạy, trạng thái, dấu vân tay output |
| Delivery Repository    | Application, nơi chứa desired state, nhánh, tham chiếu tới cặp khóa của application |
| Deployment Record      | Image version, target, phiên bản catalog, status, infrastructure reference, tiến trình theo tầng/thành phần |

## 8. Quy tắc nghiệp vụ

- Image Repository thuộc Workload Definition.
- Image tag/version thuộc Deployment.
- Mỗi deployment phải lưu chính xác image được sử dụng cho từng workload.
- Image version có thể do Developer chọn hoặc được CI cung cấp.
- Mỗi deployment deploy đúng một phiên bản Application Definition, với đúng một phiên bản catalog, vào đúng một environment (staging hoặc production) và một nơi triển khai; environment khác không bị ảnh hưởng.
- Có hai loại nơi triển khai: cloud, nơi IDP dựng VPC và cụm Kubernetes riêng cho mỗi application + environment; và cụm Kubernetes nội bộ có sẵn, nơi IDP chỉ kết nối vào, không tạo và không xóa cụm. Platform có thể khai báo nhiều cụm nội bộ; catalog quyết định application và environment nào dùng cụm nào.
- Developer không khai báo cụm Kubernetes hay network. IDP tự thêm cụm (`k8s-cluster`) mà mọi workload chạy trên đó, và những gì Resource Definition khai báo là cần thêm (`requires`). Công thức cần gì là do platform team quyết định trong catalog.
- Database và các resource khác vẫn do IDP tạo cho mỗi application + environment, kể cả trên cụm nội bộ; muốn dùng thứ có sẵn thì platform khai báo Resource Definition trỏ tới resource có sẵn.
- Catalog có phiên bản; sửa catalog là tạo phiên bản mới, phiên bản cũ giữ nguyên. Application đang chạy không bị ảnh hưởng cho tới khi Developer deploy với phiên bản catalog mới. Developer được chọn phiên bản catalog cũ hơn phiên bản đang chạy.
- Đổi sang phiên bản Application Definition hoặc phiên bản catalog khác bản đang chạy thì phải deploy toàn bộ application; deploy một phần chỉ dùng bản đang chạy.
- Promote staging → production dùng cả phiên bản Application Definition lẫn phiên bản catalog đang chạy ở staging.
- Developer có thể deploy toàn bộ hoặc một phần workload của application.
- Phạm vi deployment gồm workload được chọn, resource mà chúng depends on trực tiếp, và cụm Kubernetes/network mà các resource đó cần; resource khác ngoài phạm vi không bị reconcile. Thành phần trong phạm vi không có gì thay đổi thì được dùng lại.
- Environment Configuration được kiểm tra khớp với phiên bản được deploy trước khi lập plan.
- Resource Instance thuộc về đúng một chủ: application + environment + resource requirement + deployment target. Resource chỉ được dùng lại khi khớp đủ bộ này.
- Resource dùng chung giữa các application hoặc environment chỉ được khai báo ở Resource Definition do platform quản lý; với loại này, IDP chỉ đọc output, không tạo, sửa hay hủy hạ tầng thật.
- Workload và resource không còn trong phiên bản được deploy bị gỡ, hủy hoặc gỡ liên kết trên environment/target đó; việc này hiện trong plan và cần Developer xác nhận. Resource dùng chung chỉ được gỡ liên kết.
- Sau khi Developer xác nhận, IDP trả lời ngay; việc triển khai do Deployment Worker chạy nền dựa trên job đã lưu.
- Dependency/resource graph được xây dựng tại thời điểm deployment dựa trên phiên bản Application Definition được chọn, Environment Configuration và deployment context.
- Các thành phần được triển khai theo thứ tự phụ thuộc: một thành phần chỉ được triển khai sau khi mọi thành phần nó depends on trong phạm vi đã sẵn sàng. Các thành phần không có quan hệ depends on với nhau là độc lập.
- Resource Definition được resolve theo resource requirement (hoặc cụm Kubernetes, network), phiên bản catalog được chọn và deployment context.
- Infrastructure được reconcile thay vì luôn tạo mới.
- Resource Output chỉ được sử dụng sau khi resource tương ứng đã được resolve và sẵn sàng.
- Workload Output chỉ được sử dụng sau khi workload tương ứng healthy; với workload ngoài phạm vi, output được lấy từ bản đang chạy.
- Environment Configuration có thể phụ thuộc vào Resource Output hoặc Workload Output của thành phần mà workload depends on.
- Khi output của một thành phần thay đổi, mọi thành phần dựa trên nó được tự động làm lại trong cùng deployment: resource cần nó được reconcile lại, workload depends on nó được triển khai lại với image version đang chạy. **IDP** chỉ lưu dấu vân tay output để so sánh, không lưu giá trị output.
- Developer không trực tiếp thao tác với Terraform module, Kubernetes ConfigMap, Kubernetes Secret hoặc Kubernetes manifest.
- `score-k8s` được sử dụng để sinh base Kubernetes manifest từ resolved application specification.
- Target-specific manifest adaptation/patch được áp dụng sau bước sinh base manifest khi cần.
- Mỗi application có một Delivery Repository riêng; desired state của application này không bao giờ được ghi vào nơi chứa của application khác. Trong một Delivery Repository, desired state được tách theo nơi triển khai và environment.
- IDP tạo Delivery Repository ở lần deploy đầu tiên của application, theo quy ước đặt tên do platform cấu hình; nơi chứa đã tồn tại đúng tên thì được dùng lại.
- Mỗi application có cặp khóa truy cập riêng cho nơi chứa của mình; khóa và thông tin đăng nhập hệ thống lưu trữ Git chỉ nằm trong Secret Store, không nằm trong database, log hay Deployment Record.
- Gỡ application khỏi một environment và nơi triển khai chỉ xóa phần desired state của environment đó; Delivery Repository và khóa của application được giữ lại.
- UC-03 sử dụng CD abstraction và không phụ thuộc trực tiếp vào Fleet, Argo CD, Flux hoặc một sản phẩm CD cụ thể; việc tạo nơi chứa desired state cũng đi qua một abstraction, không phụ thuộc vào một hệ thống lưu trữ Git cụ thể.

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

Phiên bản catalog đã dùng.

Image repository và version của từng workload, và workload nào được triển khai lại tự động.

Các thành phần đã được gỡ, hủy hoặc gỡ liên kết trong deployment, nếu có.

Deployment status.

Tiến trình theo từng tầng và từng thành phần.

Infrastructure status.

CD status.

Workload health.

Endpoint nếu có.

Ví dụ:

Deployment #42 Production — Phiên bản 5 — Catalog v2 — Succeeded

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

# UC-05 – Remove Application from Environment

## Mục tiêu

Cho phép Developer gỡ một application ra khỏi **một** environment và một nơi triển khai: workload ngừng chạy, hạ tầng do IDP tạo riêng cho application ở đó được hủy, hạ tầng dùng chung chỉ bị gỡ liên kết. Application Definition, các phiên bản, Environment Configuration, Delivery Repository và cặp khóa của application đều được giữ lại, nên deploy lại lúc nào cũng được.

Use case này tách khỏi UC-03 vì nó không triển khai phiên bản nào: Developer không chọn phiên bản Application Definition, không chọn phiên bản catalog và không chọn image.

## Actor

Primary Actor: Developer

## Tiền điều kiện

Application đã từng được deploy thành công lên đúng environment và nơi triển khai được chọn.

Không có deployment nào của cùng application + environment + nơi triển khai đang chờ xác nhận hoặc đang chạy.

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

**IDP** dựng dependency/resource graph của lần deploy đó và lập plan gỡ bỏ theo **thứ tự ngược với thứ tự triển khai**:

Tầng gỡ đầu tiên: mọi workload của application trên environment và nơi triển khai đó.

Các tầng sau: resource mà workload phụ thuộc, rồi tới cụm Kubernetes và network, mỗi thứ là **hủy** nếu do IDP tạo và **gỡ liên kết** nếu là thứ có sẵn dùng chung.

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

**IDP** từ chối yêu cầu và không tạo deployment nào khi application chưa từng được deploy lên environment và nơi triển khai đã chọn, khi không còn workload hay resource nào của application ở đó, hoặc khi đang có một deployment của cùng application + environment + nơi triển khai chờ xác nhận hoặc đang chạy.

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

Chịu trách nhiệm biến một phiên bản Application Definition + phiên bản catalog + Environment Configuration của một environment (staging hoặc production) + nơi triển khai (cloud hoặc cụm Kubernetes nội bộ) và deployment context + image version của các workload được chọn thành một deployment thực tế. Trách nhiệm chia làm hai phần:

- **Nhận yêu cầu deploy:** kiểm tra configuration khớp phiên bản, dựng dependency/resource graph (kể cả cụm Kubernetes và network của nơi triển khai), xác định phạm vi và chia tầng theo thứ tự phụ thuộc, tìm resource để dùng lại theo đúng chủ sở hữu, lập plan (gồm cả thành phần sẽ bị gỡ, hủy hoặc gỡ liên kết) cho Developer xem; khi Developer xác nhận thì lưu job triển khai và trả lời ngay.
- **Thực thi chạy nền (Deployment Worker):** với từng tầng reconcile infrastructure, kể cả VPC và cụm trên cloud (hoặc chỉ liên kết cụm nội bộ và resource dùng chung), resolve configuration, sinh manifest, bảo đảm Delivery Repository của application tồn tại rồi gửi desired state sang CD system, chờ workload healthy và thu output; tự động làm lại resource và workload phụ thuộc khi output thay đổi; gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản.

## UC 04 View Deployment Result

Chịu trách nhiệm đọc và hiển thị trạng thái/kết quả của deployment, gồm infrastructure, configuration resolution, CD status, workload health, endpoint và image version thực tế. Không thay đổi application hay infrastructure.

## UC 05 Remove Application from Environment

Chịu trách nhiệm gỡ application khỏi **một** environment và một nơi triển khai: lập plan gỡ bỏ theo thứ tự ngược với thứ tự triển khai dựa trên đúng phiên bản đang chạy (Developer không chọn lại phiên bản hay image), cho Developer xem kèm cảnh báo mất dữ liệu, rồi khi được xác nhận thì chạy nền để gỡ workload, hủy resource do IDP tạo và gỡ liên kết resource dùng chung. Không xóa Application Definition, phiên bản, Environment Configuration hay Delivery Repository, và không đụng tới environment khác.

**Ranh giới giữa năm Use Case**: UC-01 định nghĩa cần gì → UC-02 định nghĩa giá trị theo environment là gì → UC-03 quyết định triển khai chúng như thế nào → UC-04 cho biết kết quả ra sao → UC-05 thu hồi những gì UC-03 đã tạo ra trên một environment.

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

- **createDeployment()** - Tạo deployment mới từ application, phiên bản Application Definition và phiên bản catalog được chọn, environment (staging hoặc production), nơi triển khai và image version của các workload được chọn.

- **validateDeploymentInput()** - Kiểm tra image version, deployment context, Environment Configuration khớp với phiên bản được chọn, và luật deploy một phần chỉ dùng phiên bản Application Definition và phiên bản catalog đang chạy.

- **buildDeploymentGraph()** - Dựng dependency/resource graph từ workload, resource, dependency, configuration reference, phiên bản catalog và deployment context; tự thêm cụm Kubernetes và những gì Resource Definition khai báo là cần thêm (`requires`), ví dụ network; phát hiện các quan hệ tạo thành vòng.

- **planDeploymentWaves()** - Xác định phạm vi deployment (workload được chọn, resource mà chúng depends on trực tiếp, cụm Kubernetes/network mà các resource đó cần), chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy; sau khi có infrastructure plan, liệt kê thành phần có thể bị làm lại (chỉ tính từ thành phần sẽ được tạo, cập nhật hoặc deploy).

- **resolveResourceDefinitions()** - Chọn Resource Definition phù hợp trong phiên bản catalog được chọn cho từng resource, cụm Kubernetes và network trong graph (được gọi trong lúc dựng graph), gồm cả việc thuộc loại do IDP quản lý hạ tầng hay loại trỏ tới thứ có sẵn (resource dùng chung, cụm nội bộ).

- **planInfrastructureChanges()** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target) và xác định resource nào cần tạo mới, cập nhật, tái sử dụng hoặc liên kết tới thứ có sẵn; resource cần cập nhật khi đầu vào hiện tại (công thức trong phiên bản catalog, tham số, output của những gì nó cần) khác đầu vào lần apply gần nhất; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **loadInfrastructureOverrides()** - Lấy các infrastructure parameter mà Developer được phép override.

- **applyInfrastructureOverrides()** - Ghi nhận các giá trị override mà Developer lựa chọn.

- **confirmDeployment()** - Xác nhận deployment sau khi Developer kiểm tra các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị làm lại và các override. Operation chỉ đổi trạng thái deployment sang đã xác nhận và tạo job triển khai (kèm các giá trị override đã chọn) trong cùng một lần lưu, rồi trả lời ngay; không tự thực thi việc triển khai.

- **reconcileInfrastructure()** - Do Deployment Worker gọi. Thực thi việc tạo/cập nhật infrastructure của các resource do IDP quản lý trong một tầng (kể cả VPC, cụm Kubernetes trên cloud) thông qua provisioner phù hợp, với input gồm output của những gì resource cần; hoặc liên kết Resource Instance tới thứ có sẵn (resource dùng chung, cụm nội bộ) mà không đụng hạ tầng thật; với thành phần không còn trong phiên bản, thực thi việc hủy resource hoặc gỡ liên kết resource dùng chung.

- **collectResourceOutputs()** - Thu thập Resource Output sau khi infrastructure resource sẵn sàng.

- **resolveEnvironmentConfiguration()** - Resolve configuration của các workload trong một tầng từ direct value, Resource Output và Workload Output.

- **generateResolvedApplicationSpecification()** - Tạo resolved application specification chứa image version, configuration và dependency đã resolve.

- **generateKubernetesManifest()** - Sinh base Kubernetes manifest từ resolved specification bằng score-k8s.

- **adaptManifestForTarget()** - Áp dụng target-specific patch/adaptation cho Kubernetes manifest nếu deployment target yêu cầu.

- **materializeEnvironmentConfiguration()** - Chuyển Environment Variable đã resolve thành Kubernetes configuration, ví dụ ConfigMap hoặc cấu hình tương ứng.

- **materializeSecretConfiguration()** - Chuyển Secret đã resolve thành Kubernetes Secret hoặc secret reference phù hợp.

- **publishDesiredDeploymentState()** - Bảo đảm application có Delivery Repository của riêng nó (tạo nơi chứa và cặp khóa ở lần đầu, ghi nhận lại), rồi gửi desired deployment state của các workload trong một tầng sang CD abstraction; khi deploy phiên bản mới, desired state không còn các workload bị gỡ để CD system gỡ chúng khỏi cluster.

- **waitForWorkloadsHealthy()** - Chờ tới khi pod của các workload trong tầng healthy trên deployment target.

- **collectWorkloadOutputs()** - Thu thập Workload Output từ workload vừa healthy, hoặc từ workload đang chạy ngoài phạm vi deployment.

- **propagateOutputChanges()** - So sánh dấu vân tay output mới với lần triển khai trước; nếu thay đổi, reconcile lại các resource cần thành phần đó và thêm các workload dựa trên nó vào các tầng sau với image version đang chạy.

- **saveDeploymentRecord()** - Lưu Deployment Record, image version, infrastructure reference, tiến trình theo tầng/thành phần và trạng thái thực thi.

Chuỗi chính gồm hai phần:

- **Trong request của Developer:** tạo deployment (chọn phiên bản Application Definition và phiên bản catalog) → kiểm tra input và configuration khớp phiên bản → dựng graph (resolve Resource Definition, thêm cụm/network) → chia tầng → plan/override infra (gồm gỡ/hủy/gỡ liên kết) → liệt kê thành phần có thể bị làm lại → xác nhận: lưu job và trả lời ngay.
- **Deployment Worker chạy nền:** với mỗi tầng: reconcile infra (kể cả VPC, cụm) hoặc liên kết thứ có sẵn → thu Resource Output → resolve config → sinh resolved spec → sinh base manifest → adapt theo target → materialize config/secret → publish sang CD → chờ healthy → thu Workload Output → lan truyền thay đổi output (reconcile lại resource, triển khai lại workload); sau các tầng: gỡ workload, hủy resource hoặc gỡ liên kết resource dùng chung không còn trong phiên bản → lưu deployment record.

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

## UC 05 Remove Application from Environment

- **createTeardown()** - Nhận application, environment và nơi triển khai; lấy lại phiên bản Application Definition, phiên bản catalog và Environment Configuration của lần deploy gần nhất, dựng plan gỡ bỏ theo thứ tự ngược và lưu deployment loại gỡ bỏ ở trạng thái chờ xác nhận.

- **confirmDeployment()** - Dùng lại operation của UC-03: kiểm tra plan chưa đổi, ghi xác nhận và tạo job trong một transaction, rồi trả lời ngay.

- **removeWorkloadsAndInfrastructure()** - Deployment Worker chạy các tầng gỡ theo thứ tự ngược: publish desired state không còn workload, xác minh workload đã biến mất, gỡ đối tượng khỏi CD system và xóa desired state của environment, xóa không gian tên, rồi hủy hoặc gỡ liên kết từng resource.

- **saveDeploymentRecord()** - Dùng lại operation của UC-03 để lưu kết quả lần gỡ.

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

- **Web UI** - Cho Developer chọn environment, nơi triển khai (cloud hoặc cụm nội bộ), phiên bản Application Definition, phiên bản catalog, workload cần deploy và image version, xem các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị làm lại, nhập override và xác nhận deploy.

- **Deployment API / Controller** - Nhận request từ UI, validate ở mức request và chuyển sang Deployment Orchestrator.

### Application services

- **Deployment Orchestrator** - Điều phối phần xử lý trong request của Developer: tạo deployment theo phiên bản Application Definition và phiên bản catalog được chọn, kiểm tra input, dựng graph, chia tầng, lập plan; khi Developer xác nhận thì đổi trạng thái deployment và lưu job triển khai trong cùng một lần lưu rồi trả lời ngay. Không tự thực thi việc triển khai.

- **Deployment Worker** - Tiến trình chạy nền nhận job triển khai đã lưu và điều phối phần thực thi: triển khai lần lượt từng tầng tới khi mọi workload trong phạm vi healthy, lan truyền khi output thay đổi (reconcile lại resource, triển khai lại workload), gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản, và cập nhật trạng thái deployment.

### Domain components

- **Deployment Graph Builder** - Dựng dependency/resource graph từ Application Definition, Environment Configuration, phiên bản catalog và deployment context; tự thêm cụm Kubernetes và những gì Resource Definition `requires` (gọi Resource Definition Resolver trong lúc dựng); phát hiện các quan hệ tạo thành vòng.

- **Deployment Wave Planner** - Xác định phạm vi deployment, chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy, liệt kê thành phần có thể bị làm lại, và lan truyền khi output thay đổi bằng cách reconcile lại resource phụ thuộc và thêm workload phụ thuộc vào các tầng sau.

- **Resource Definition Resolver** - Chọn Resource Definition phù hợp trong phiên bản catalog được chọn cho từng logical resource, cụm Kubernetes và network, dựa trên type và deployment context; phân biệt definition do IDP quản lý hạ tầng với definition trỏ tới thứ có sẵn (resource dùng chung, cụm nội bộ).

- **Infrastructure Planner** - Tìm Resource Instance hiện có theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target), so sánh đầu vào hiện tại (công thức trong phiên bản catalog, tham số, output của những gì resource cần) với đầu vào lần apply gần nhất để xác định cần create, update, reuse hay liên kết tới thứ có sẵn; khi deploy phiên bản mới, xác định thêm workload cần gỡ, resource cần hủy và resource dùng chung cần gỡ liên kết.

- **Infrastructure Reconciler** - Điều phối việc reconcile infrastructure của từng tầng theo plan đã xác định: tạo/sửa resource do IDP quản lý (kể cả VPC, cụm Kubernetes trên cloud) với input từ output của những gì resource cần, chỉ liên kết (không đụng hạ tầng thật) với thứ có sẵn (resource dùng chung, cụm nội bộ), và thực hiện hủy resource hoặc gỡ liên kết resource không còn trong phiên bản.

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

- **CD Integration / CD Provider Interface** - Abstraction để publish desired deployment state mà không phụ thuộc trực tiếp vào Fleet, Argo CD, Flux hay implementation cụ thể.

- **Delivery Repository Provider** - Abstraction để bảo đảm nơi chứa desired state của một application tồn tại: tạo nơi chứa theo quy ước đặt tên, sinh và gắn cặp khóa của application. Không phụ thuộc vào một hệ thống lưu trữ Git cụ thể.

- **Workload Status Provider / Kubernetes Adapter** - Kiểm tra workload đã healthy chưa và đọc dữ liệu runtime của workload phục vụ Workload Output Collector.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể của CD abstraction, ví dụ Fleet Adapter, Argo CD Adapter hoặc Flux Adapter.

- **Concrete Git Hosting Provider** - Implementation cụ thể của Delivery Repository Provider cho một hệ thống lưu trữ Git, ví dụ GitHub Adapter; đọc thông tin đăng nhập từ Secret Store.

### External systems

- **Terraform/OpenTofu Runner** - Thực thi Terraform/OpenTofu module để tạo hoặc cập nhật infrastructure theo yêu cầu từ Provisioner Adapter.

- **score-k8s** - Sinh base Kubernetes manifest từ resolved application specification.

- **CD System** - Hệ thống CD bên ngoài, ví dụ Fleet, Argo CD hoặc Flux, nhận desired deployment state và đồng bộ xuống Kubernetes.

- **Kubernetes Cluster** - Nơi workload thực sự chạy. Trên cloud, cụm do IDP dựng qua Provisioner; với cụm nội bộ, cụm có sẵn và được platform khai báo trong catalog. Thông tin kết nối cụm lấy từ output của `k8s-cluster`. Cung cấp trạng thái health và dữ liệu runtime của workload.

### Persistence

- **Application Repository** - Đọc phiên bản Application Definition được chọn (đã lưu từ UC-01); tạo Platform Requirement (cụm Kubernetes, network) của application khi cần lần đầu.

- **Environment Configuration Repository** - Đọc configuration/reference đã lưu từ UC-02.

- **Resource Definition Catalog** - Đọc các phiên bản catalog và Resource Definition của phiên bản được chọn. Catalog do platform quản lý; IDP chỉ đọc.

- **Delivery Repository Registry** - Lưu/đọc Delivery Repository của từng application: nơi chứa desired state, nhánh và tham chiếu tới cặp khóa trong Secret Store.

- **Resource Instance Repository** - Lưu/đọc Resource Instance theo đúng chủ sở hữu (application + environment + resource requirement hoặc cụm Kubernetes/network + deployment target): trạng thái, reference, liên kết tới thứ có sẵn, dấu vân tay output và dấu vân tay đầu vào lần apply gần nhất, phục vụ reconcile/reuse, gỡ/hủy và lan truyền thay đổi output.

- **Workload Instance Repository** - Lưu/đọc trạng thái hiện hành của từng workload theo environment và deployment target: image đang chạy, trạng thái health và dấu vân tay output.

- **Deployment Repository** - Lưu deployment (kèm phiên bản Application Definition và phiên bản catalog), job triển khai cùng giá trị override đã chọn, Deployment Record, image version, target, infrastructure reference, tiến trình theo tầng/thành phần, status và lỗi nếu có.

Luồng responsibility:

- **Trong request:** Web UI → Deployment API → Deployment Orchestrator → Graph Builder (→ Resource Definition Resolver → Resource Definition Catalog) → Wave Planner → Infrastructure Planner → Wave Planner (thành phần có thể bị làm lại) → Deployment Repository (lưu deployment; khi xác nhận thì lưu job) → Web UI.
- **Chạy nền:** Deployment Worker → (với mỗi tầng) Infrastructure Reconciler → Provisioner → Terraform/OpenTofu Runner → Resource Output Collector → Configuration Resolver → Resolved Spec Generator → Score Renderer → score-k8s → Target Adapter → Config/Secret Materializer → Delivery Repository Provider (bảo đảm nơi chứa của application) → CD Integration → Concrete CD Provider → CD System → Kubernetes → Workload Status Provider → Workload Output Collector → Wave Planner (lan truyền sang resource và workload) → (sau các tầng) gỡ/hủy/gỡ liên kết thành phần không còn trong phiên bản → Resource/Workload Instance Repository → Deployment Repository.

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

- **Concrete CD Provider** - Implementation cụ thể để truy vấn Fleet, Argo CD, Flux hoặc CD system khác.

### External systems

- **CD System** - Hệ thống CD bên ngoài, ví dụ Fleet, Argo CD hoặc Flux, cung cấp trạng thái deployment và trạng thái đồng bộ.

- **Kubernetes Cluster** - Nguồn trạng thái runtime thực tế của workload.

### Persistence

- **Deployment Repository** - Đọc Deployment Record, image version, target, trạng thái từng bước và error summary.

- **Resource Instance Repository** - Đọc thông tin infrastructure reference và trạng thái resource liên quan đến deployment.

- **Workload Instance Repository** - Đọc image version đang chạy của từng workload theo environment và deployment target, để cho biết deployment đang xem có còn là bản đang chạy hay đã được thay thế.

Luồng responsibility: Web UI → Deployment Query API → Deployment Query Service → Deployment Repository + Resource Repository + Workload Instance Repository + CD Status Provider → Concrete CD Provider → CD System + Kubernetes Adapter → Kubernetes Cluster → Result Aggregator → Web UI.

UC-04 không dùng Deployment Orchestrator để thực hiện hành động. UC-04 đi theo query path riêng vì chỉ đọc và tổng hợp trạng thái, không reconcile infrastructure hoặc trigger deployment.

## UC 05 Remove Application from Environment

### Boundary/UI

- **Web UI** - Cho Developer chọn environment và nơi triển khai cần gỡ, hiển thị plan gỡ bỏ kèm cảnh báo mất dữ liệu, rồi theo dõi tiến trình gỡ.

- **Deployment API / Controller** - Nhận yêu cầu lập plan gỡ bỏ và yêu cầu xác nhận, chuyển sang orchestrator. Dùng chung controller với UC-03.

### Application services

- **Deployment Orchestrator** - Lấy lại phiên bản, phiên bản catalog và Environment Configuration của lần deploy gần nhất trên đúng chủ sở hữu, dựng plan gỡ bỏ, lưu deployment loại gỡ bỏ, và khi được xác nhận thì tạo job rồi trả lời ngay. Không tự thực thi.

- **Deployment Worker** - Chạy nền các tầng gỡ theo thứ tự ngược: gỡ workload, gỡ đối tượng khỏi CD system, xóa không gian tên, hủy hoặc gỡ liên kết resource, rồi lưu Deployment Record.

### Domain components

- **Deployment Graph Builder** - Dựng lại graph của phiên bản đang chạy để biết thứ tự phụ thuộc cần đảo ngược khi gỡ.

- **Infrastructure Planner** - Lập plan gỡ bỏ: tầng đầu là workload, các tầng sau là resource với hành động hủy (do IDP tạo) hoặc gỡ liên kết (có sẵn, dùng chung), kèm cảnh báo mất dữ liệu.

- **Infrastructure Reconciler** - Hủy hạ tầng do IDP tạo theo đúng thứ tự ngược.

### Integration abstractions

- **CD Integration / CD Provider Interface** - Gửi desired state không còn workload, rồi gỡ đối tượng của application khỏi CD system và xóa phần desired state của environment khỏi Delivery Repository.

- **Workload Status Provider / Kubernetes Adapter** - Xác minh workload đã biến mất khỏi cụm và xóa không gian tên của application.

- **Provisioner Adapter / Provisioner Interface** - Hủy hạ tầng qua provisioner tương ứng của từng Resource Definition.

### Integration implementations

- **Concrete CD Provider** - Hiện thực cụ thể cho Fleet, Argo CD hoặc CD system khác.

### External systems

- **CD System**, **Kubernetes Cluster**, **Terraform/OpenTofu Runner** - Nơi thao tác gỡ bỏ thực sự diễn ra.

### Persistence

- **Deployment Repository** - Tìm deployment gần nhất của chủ sở hữu, lưu deployment loại gỡ bỏ, tạo job, lưu Deployment Record của lần gỡ.

- **Resource Instance Repository** - Đọc Resource Instance thuộc chủ sở hữu và cập nhật sang trạng thái đã hủy hoặc đã gỡ liên kết.

- **Workload Instance Repository** - Đọc workload đang chạy và cập nhật sang trạng thái đã gỡ sau khi đã xác minh.

- **Delivery Repository Registry** - Đọc nơi chứa desired state của application để xóa đúng phần của environment bị gỡ; row đăng ký không bị xóa.

- **Application Repository**, **Environment Configuration Repository**, **Resource Definition Catalog** - Chỉ đọc, để dựng lại graph và plan của phiên bản đang chạy.

Luồng responsibility: Web UI → Deployment API → Deployment Orchestrator → Deployment Repository + Application Repository + Environment Configuration Repository + Resource Definition Catalog + Graph Builder + Infrastructure Planner → (Developer xác nhận) → Deployment Worker → CD Integration → Concrete CD Provider → CD System, Kubernetes Adapter → Kubernetes Cluster, Infrastructure Reconciler → Provisioner → Terraform Runner → Resource/Workload Instance Repository → Deployment Repository.

UC-05 không thêm class mới nào so với UC-03: nó dùng lại đúng các thành phần đó, chỉ khác ở chỗ plan chỉ gồm các tầng gỡ bỏ và thứ tự là ngược lại. Application Repository và Environment Configuration Repository chỉ được đọc; không use case nào trong bốn use case còn lại, và cả UC-05, xóa Application Definition hay Environment Configuration.

