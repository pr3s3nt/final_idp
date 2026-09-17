---
id: UC-03
artifact: use-case-specification
status: current
delivery_status: implemented-and-verified
last_reviewed: 2026-09-17
---

# UC-03 — Deploy Application


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

### UC03-MF-01 — Developer chọn Deploy Application

### UC03-MF-02 — Developer chọn phạm vi phiên bản và target

    * Environment: staging hoặc production.
    * Nơi triển khai: cloud (ví dụ AWS) hoặc một cụm Kubernetes nội bộ mà platform đã khai báo.
    * Phiên bản Application Definition cần deploy.
    * Phiên bản catalog.

   IDP chọn sẵn phiên bản catalog đang chạy trên environment/nơi triển khai đã chọn và báo nếu có phiên bản mới hơn; nếu chưa có gì chạy thì chọn sẵn phiên bản mới nhất. Khi promote lên production, Developer dùng cả phiên bản Application Definition lẫn phiên bản catalog đang chạy ở staging.

### UC03-MF-03 — IDP hiển thị trạng thái và lựa chọn hiện tại

   Ví dụ:

    ```text
    Environment: production    Đang chạy: phiên bản 4, catalog v2    Chọn deploy: phiên bản 5, catalog v2

    backend
    registry.company.local/shop-backend       (đang chạy: v1.4.2)

    frontend
    registry.company.local/shop-frontend      (đang chạy: v2.0.9)
    ```

### UC03-MF-04 — Developer chọn workload và image version

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

### UC03-MF-05 — Developer chọn deployment context

   Ví dụ khi deploy lên Cloud:

    ```text
    Cloud Provider: **AWS**
    Region: ap-southeast-1
    ```

### UC03-MF-06 — IDP kiểm tra input và Environment Configuration

   IDP kiểm tra Environment Configuration của environment đã chọn khớp với phiên bản được deploy: mọi configuration bắt buộc của phiên bản đã có giá trị, và không biến nào còn tham chiếu output của thành phần không có trong phiên bản hoặc không được depends on trong phiên bản.

### UC03-MF-07 — IDP dựng dependency/resource graph

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

### UC03-MF-08 — IDP resolve Resource Definition

   Resource Definition có hai loại: loại do IDP quản lý hạ tầng (tạo, sửa, hủy) và loại trỏ tới thứ có sẵn do platform khai báo (IDP chỉ dùng, không đụng tới hạ tầng thật), ví dụ database dùng chung hoặc cụm nội bộ.

   Ví dụ:

    ```text
    Cloud (AWS):   k8s-cluster → eks-cluster        (IDP quản lý, cần network)
                   network     → aws-network        (IDP quản lý)
                   postgresql  → aurora-postgresql  (IDP quản lý, cần network)

    Cụm nội bộ:    k8s-cluster → cụm nội bộ có sẵn  (trỏ tới cụm có sẵn)
                   postgresql  → postgres-k8s       (IDP quản lý, chạy trong cụm)
    ```

### UC03-MF-09 — IDP xác định phạm vi và chia tầng

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

### UC03-MF-10 — IDP tìm Resource Instance theo chủ sở hữu

### UC03-MF-11 — IDP lập infrastructure plan

    * Tạo mới.
    * Cập nhật.
    * Tái sử dụng.
    * Hoặc liên kết tới thứ có sẵn (resource dùng chung, cụm nội bộ).

   Resource cần cập nhật khi đầu vào hiện tại khác đầu vào lần apply gần nhất: công thức trong phiên bản catalog, tham số, hoặc output của những gì resource đó cần.

   Khi deploy một phiên bản mới, IDP đồng thời xác định các thành phần đang chạy trên environment/target nhưng không còn trong phiên bản:

    * Workload cần gỡ.
    * Resource do IDP quản lý cần hủy.
    * Resource dùng chung cần gỡ liên kết (không hủy hạ tầng thật).

### UC03-MF-12 — IDP xác định phạm vi cascade tiềm năng

   Chỉ tính từ các thành phần sẽ được tạo, cập nhật hoặc deploy; thành phần được tái sử dụng không đổi output.

   Ví dụ: `network` sẽ được cập nhật thì `k8s-cluster`, `postgresql` và các workload chạy trên cụm có thể bị làm lại.

### UC03-MF-13 — IDP hiển thị plan và cảnh báo

### UC03-MF-14 — Developer xác nhận deployment

### UC03-MF-15 — IDP ghi nhận và tạo job

   Developer không phải chờ việc triển khai hoàn tất; Developer theo dõi tiến trình và kết quả ở UC-04 – View Deployment Result.

### UC03-MF-16 — Deployment Worker thực thi theo tầng

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

### UC03-MF-17 — IDP phát hiện và lan truyền thay đổi output

   Nếu output thay đổi, mọi thành phần dựa trên thành phần đó được làm lại trong chính deployment này, kể cả khi chưa nằm trong phạm vi:

    * Resource cần thành phần đó (qua `requires`) được reconcile lại với output mới.
    * Workload depends on thành phần đó, hoặc chạy trên cụm đó, được thêm vào các tầng sau với image version đang chạy.

   Việc lan truyền tiếp tục cho tới khi không còn output thay đổi.

   Ví dụ chỉ deploy backend làm `backend.endpoint` thay đổi: frontend được thêm vào tầng kế tiếp với image đang chạy. Ví dụ network đổi subnet: cụm EKS và Aurora được reconcile lại ở tầng kế tiếp, rồi các workload dựa trên chúng được triển khai lại.

### UC03-MF-18 — IDP xử lý thành phần bị loại khỏi phiên bản

    * Gỡ các workload không còn trong phiên bản.
    * Sau đó hủy resource do IDP quản lý, hoặc gỡ liên kết resource dùng chung, không còn trong phiên bản.

   Ví dụ phiên bản 5 không còn `worker` và `redis`: sau khi backend và frontend của phiên bản 5 đã healthy, IDP gỡ `worker`, rồi hủy `redis`.

### UC03-MF-19 — IDP lưu Deployment Record

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

Deployment dừng theo danh mục ổn định sau:

- **A1-1 — Invalid image:** image version không hợp lệ hoặc không tồn tại trong registry đích.
- **A1-2 — Missing configuration:** thiếu Environment Variable hoặc Secret bắt buộc.
- **A1-3 — Configuration/version mismatch:** Environment Configuration không khớp phiên bản Application Definition được deploy, ví dụ binding còn trỏ tới thành phần không có trong phiên bản.
- **A1-4 — Invalid partial deployment:** deploy một phần nhưng phiên bản Application Definition hoặc Catalog Version khác bản đang chạy, hoặc trạng thái đang chạy không thống nhất để xác định baseline.
- **A1-5 — Invalid dependency graph:** dependency hay `requires` không resolve được hoặc tạo thành vòng.
- **A1-6 — Dependency outside scope not ready:** workload/resource cần đọc ngoài phạm vi chưa ở trạng thái `HEALTHY`/`READY` trên đúng environment và target.
- **A1-7 — Invalid Resource Definition:** không có hoặc có nhiều Resource Definition cùng mức ưu tiên; management mode không tương thích; hoặc definition của một Resource Instance đang chạy đổi sang definition khác (`RESOURCE_DEFINITION_CHANGED`). IDP không tự thay thế resource đang chạy.
- **A1-8 — Invalid output reference:** Resource Output hoặc Workload Output reference không tồn tại, không được expose hoặc không thuộc dependency được phép.
- **A1-9 — Invalid override:** override không được phép, sai kiểu/phạm vi, hoặc cố đổi tham số bất biến. Deployment giữ `AWAITING_CONFIRMATION`; không job nào được tạo.
- **A1-10 — Duplicate confirmation:** xác nhận lại một Deployment đã được xác nhận trả `DEPLOYMENT_ALREADY_CONFIRMED`; deployment đó vẫn chỉ có đúng một execution job.
- **A1-11 — Unsupported target/context:** target hoặc context không được hỗ trợ, đặc biệt không có Resource Definition `k8s-cluster` phù hợp.
- **A1-12 — Overlapping deployment:** đã có Deployment cùng application, environment và target ở `CONFIRMED` hoặc `DEPLOYING`; yêu cầu mới trả `DEPLOYMENT_IN_PROGRESS`.

Với lỗi phát hiện trước khi persist plan (A1-1 đến A1-8, A1-11 và A1-12), attempt không để lại row Deployment hay job và không gây side effect. A1-9 chỉ giữ plan hiện có ở `AWAITING_CONFIRMATION`; A1-10 không tạo job thứ hai. **IDP** hiển thị mã lỗi để Developer chỉnh sửa hoặc tải lại plan.

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
- Override đã apply thành công trở thành baseline của Resource Instance. Lần deploy sau không truyền lại override thì tiếp tục dùng baseline đó; override mới chỉ thay các key được phép và plan hiển thị `UPDATE` nếu effective input thay đổi.
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
