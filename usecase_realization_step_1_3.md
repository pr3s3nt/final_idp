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

Application Definition được tạo hoặc cập nhật.

Workload, resource, dependency và configuration requirement được lưu.

Application specification được tạo hoặc cập nhật.

Deployment hiện tại không tự động bị thay đổi.

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

**IDP** kiểm tra dữ liệu, lưu Application Definition và sinh/cập nhật application specification.

## Luồng ngoại lệ

A1 – Dữ liệu không hợp lệ

Nếu dữ liệu không hợp lệ, **IDP** hiển thị lỗi để Developer chỉnh sửa.

Ví dụ:

Workload name bị trùng.

Image repository không hợp lệ.

Port không hợp lệ.

Dependency tham chiếu tới thành phần không tồn tại.

Các quan hệ depends on tạo thành vòng, ví dụ frontend → backend và backend → frontend.

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

Developer chọn environment cần cấu hình.

Ví dụ:

dev staging production

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

Configuration được quản lý riêng theo từng environment.

Cùng một Environment Variable có thể có giá trị khác nhau giữa các environment.

**IDP** chỉ cho Developer chọn những output mà Resource Definition hoặc workload expose.

Workload chỉ được tham chiếu output của resource hoặc workload mà nó đã khai báo depends on trong UC-01. Nếu cần output của thành phần khác, Developer phải bổ sung dependency ở UC-01 trước.

Resource Output và Workload Output chỉ lưu reference; giá trị thực tế được resolve trong quá trình deployment.

Giá trị Secret không được hiển thị lại dưới dạng plaintext.

Thay đổi Environment Configuration không tự động làm thay đổi deployment đang chạy.

# UC-03 – Deploy Application

## 1. Mục tiêu

Cho phép Developer triển khai một phiên bản cụ thể của application, hoặc một phần application, lên một environment và Kubernetes deployment target.

Image tag/version được xác định tại thời điểm deployment, không phải khi định nghĩa application trong UC-01.

**IDP** chịu trách nhiệm:

- Xác định phạm vi deployment: các workload được chọn và các resource mà chúng depends on trực tiếp.
- Xác định dependency và chia các thành phần trong phạm vi thành các tầng triển khai theo thứ tự phụ thuộc.
- Resolve Resource Definition phù hợp theo deployment context.
- Reconcile infrastructure.
- Triển khai lần lượt từng tầng: resolve Environment Configuration từ output của các tầng trước, sinh Kubernetes manifest, chuyển desired deployment state cho hệ thống Continuous Delivery và chờ workload healthy.
- Thu thập Resource Output và Workload Output sau khi thành phần tương ứng sẵn sàng.
- Tự động triển khai lại các thành phần phụ thuộc khi output mà chúng dùng bị thay đổi.

## 2. Actor

**Primary Actor:** Developer

## 3. Tiền điều kiện

- Application Definition hợp lệ và các quan hệ depends on không tạo thành vòng.
- Workload cần deploy có Image Repository.
- Environment Configuration cần thiết đã được cấu hình.
- Platform đã có Resource Definition và provisioner phù hợp.
- Deployment target được hỗ trợ.
- Mọi workload mà các workload được chọn depends on nhưng không nằm trong phạm vi deployment đang chạy healthy trên cùng environment và deployment target.

## 4. Hậu điều kiện

- Deployment ghi nhận chính xác image version của từng workload được triển khai, gồm cả workload được triển khai lại tự động.
- Infrastructure trong phạm vi deployment đã được reconcile và sẵn sàng.
- Environment Configuration và các dependency reference được resolve.
- Kubernetes manifest được sinh với đúng image và configuration.
- Mọi workload được triển khai đều healthy.
- Trạng thái hiện hành của từng workload và resource đã triển khai (Workload Instance, Resource Instance), gồm dấu vân tay output, được cập nhật.
- Deployment Record được lưu.

## 5. Luồng chính

## Developer chọn **Deploy Application**.

## Developer chọn:

    * Environment.
    * Deployment target.

## IDP hiển thị các workload, Image Repository tương ứng và image version đang chạy trên environment/target đã chọn nếu có.

   Ví dụ:

    ```text
    backend
    registry.company.local/shop-backend       (đang chạy: v1.4.2)

    frontend
    registry.company.local/shop-frontend      (đang chạy: v2.0.9)
    ```

## Developer chọn các workload cần deploy và chọn hoặc xác nhận image tag/version cho từng workload được chọn.

   Developer có thể chọn toàn bộ hoặc chỉ một phần workload của application.

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

## IDP xác định các infrastructure resource cần:

    * Tạo mới.
    * Cập nhật.
    * Hoặc tái sử dụng.

## IDP hiển thị các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị triển khai lại và các infrastructure parameter mà Developer được phép override.

## Developer xác nhận và chọn **Deploy**.

## IDP triển khai lần lượt từng tầng. Với mỗi tầng:

   Với mỗi resource trong tầng:

    * IDP reconcile infrastructure.
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

## IDP lưu Deployment Record, bao gồm:

- Environment.
- Deployment target.
- Image version thực tế của từng workload và workload nào được triển khai lại tự động.
- Infrastructure reference.
- Tiến trình theo từng tầng và từng thành phần.
- Trạng thái deployment.

## 6. Luồng ngoại lệ

### A1 – Deployment input hoặc dependency không hợp lệ

Deployment dừng nếu:

- Image version không hợp lệ.
- Configuration bắt buộc chưa được cấu hình.
- Dependency không thể resolve hoặc các quan hệ depends on tạo thành vòng.
- Workload phụ thuộc không nằm trong phạm vi deployment và chưa chạy healthy trên environment/target đã chọn.
- Resource Definition phù hợp không tồn tại.
- Resource Output hoặc Workload Output được tham chiếu không hợp lệ.

**IDP** hiển thị lỗi để Developer chỉnh sửa.

### A2 – Provisioning, delivery hoặc workload thất bại

Nếu infrastructure provisioning, manifest generation, CD delivery thất bại hoặc workload không healthy tại một tầng:

- **IDP** ghi nhận deployment thất bại và không triển khai các tầng sau.
- **IDP** lưu tầng, thành phần liên quan, failed step và error summary.
- Developer có thể xem chi tiết trong UC-04 – View Deployment Result.

## 7. Dữ liệu chính

| Nhóm                   | Dữ liệu                                                 |
| ---------------------- | ------------------------------------------------------- |
| Deployment             | Application, environment, deployment target             |
| Workload Deployment    | Workload, image repository, image version, được chọn hay triển khai lại tự động |
| Deployment Context     | Cloud provider, region, target-specific input           |
| Deployment Graph       | Workload, resource, dependency, configuration reference, phạm vi, tầng triển khai |
| Resource Resolution    | Resource, Resource Definition                           |
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
- Developer có thể deploy toàn bộ hoặc một phần workload của application.
- Phạm vi deployment gồm workload được chọn và resource mà chúng depends on trực tiếp; resource ngoài phạm vi không bị reconcile.
- Dependency/resource graph được xây dựng tại thời điểm deployment dựa trên Application Definition, Environment Configuration và deployment context.
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

Image repository và version của từng workload, và workload nào được triển khai lại tự động.

Deployment status.

Tiến trình theo từng tầng và từng thành phần.

Infrastructure status.

CD status.

Workload health.

Endpoint nếu có.

Ví dụ:

Deployment #42 Production — Succeeded

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

Chịu trách nhiệm khai báo cấu trúc logic của application: workload, resource requirement, dependency và các Environment Variable/Secret mà workload cần. Lưu Application Definition và sinh/cập nhật application specification.

## UC 02 Configure Application Environment

Chịu trách nhiệm khai báo giá trị cấu hình theo từng environment, bao gồm giá trị trực tiếp hoặc reference tới Resource Output / Workload Output. Không thực hiện deployment.

## UC 03 Deploy Application

Chịu trách nhiệm biến Application Definition + Environment Configuration + deployment context + image version của các workload được chọn thành một deployment thực tế: dựng dependency/resource graph, xác định phạm vi và chia tầng theo thứ tự phụ thuộc, rồi với từng tầng reconcile infrastructure, resolve configuration, sinh manifest, gửi desired state sang CD system, chờ workload healthy và thu output; tự động triển khai lại thành phần phụ thuộc khi output thay đổi.

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

- **saveApplicationDefinition()** - Lưu Application Definition.

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

- **createDeployment()** - Tạo deployment mới từ application, environment, deployment target và image version của các workload được chọn.

- **validateDeploymentInput()** - Kiểm tra image version, Environment Configuration và deployment context.

- **buildDeploymentGraph()** - Dựng dependency/resource graph từ workload, resource, dependency, configuration reference và deployment context; phát hiện các quan hệ depends on tạo thành vòng.

- **planDeploymentWaves()** - Xác định phạm vi deployment (workload được chọn và resource mà chúng depends on trực tiếp), chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy và liệt kê thành phần có thể bị triển khai lại.

- **resolveResourceDefinitions()** - Chọn Resource Definition phù hợp cho từng resource trong phạm vi deployment.

- **planInfrastructureChanges()** - Xác định infrastructure resource nào cần tạo mới, cập nhật hoặc tái sử dụng.

- **loadInfrastructureOverrides()** - Lấy các infrastructure parameter mà Developer được phép override.

- **applyInfrastructureOverrides()** - Ghi nhận các giá trị override mà Developer lựa chọn.

- **confirmDeployment()** - Xác nhận deployment sau khi Developer kiểm tra các tầng triển khai, infrastructure plan, danh sách thành phần có thể bị triển khai lại và các override.

- **reconcileInfrastructure()** - Thực thi việc tạo/cập nhật infrastructure của các resource trong một tầng thông qua provisioner phù hợp.

- **collectResourceOutputs()** - Thu thập Resource Output sau khi infrastructure resource sẵn sàng.

- **resolveEnvironmentConfiguration()** - Resolve configuration của các workload trong một tầng từ direct value, Resource Output và Workload Output.

- **generateResolvedApplicationSpecification()** - Tạo resolved application specification chứa image version, configuration và dependency đã resolve.

- **generateKubernetesManifest()** - Sinh base Kubernetes manifest từ resolved specification bằng score-k8s.

- **adaptManifestForTarget()** - Áp dụng target-specific patch/adaptation cho Kubernetes manifest nếu deployment target yêu cầu.

- **materializeEnvironmentConfiguration()** - Chuyển Environment Variable đã resolve thành Kubernetes configuration, ví dụ ConfigMap hoặc cấu hình tương ứng.

- **materializeSecretConfiguration()** - Chuyển Secret đã resolve thành Kubernetes Secret hoặc secret reference phù hợp.

- **publishDesiredDeploymentState()** - Gửi desired deployment state của các workload trong một tầng sang CD abstraction.

- **waitForWorkloadsHealthy()** - Chờ tới khi pod của các workload trong tầng healthy trên deployment target.

- **collectWorkloadOutputs()** - Thu thập Workload Output từ workload vừa healthy, hoặc từ workload đang chạy ngoài phạm vi deployment.

- **propagateOutputChanges()** - So sánh dấu vân tay output mới với lần triển khai trước; nếu thay đổi, thêm các thành phần depends on vào các tầng sau với image version đang chạy.

- **saveDeploymentRecord()** - Lưu Deployment Record, image version, infrastructure reference, tiến trình theo tầng/thành phần và trạng thái thực thi.

Chuỗi chính: tạo deployment → dựng graph → chia tầng → resolve resource → plan/override infra → xác nhận → với mỗi tầng: reconcile infra → thu Resource Output → resolve config → sinh resolved spec → sinh base manifest → adapt theo target → materialize config/secret → publish sang CD → chờ healthy → thu Workload Output → lan truyền thay đổi output → lưu deployment record.

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

- **Application Repository** - Lưu và đọc Application Definition.

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

- **Deployment Orchestrator** - Điều phối toàn bộ luồng deploy từ lúc tạo deployment, qua từng tầng triển khai, đến khi mọi workload trong phạm vi healthy.

### Domain components

- **Deployment Graph Builder** - Dựng dependency/resource graph từ Application Definition, Environment Configuration và deployment context; phát hiện các quan hệ depends on tạo thành vòng.

- **Deployment Wave Planner** - Xác định phạm vi deployment, chia các thành phần trong phạm vi thành các tầng theo thứ tự phụ thuộc, kiểm tra workload phụ thuộc ngoài phạm vi đang chạy healthy, và lan truyền khi output thay đổi bằng cách thêm thành phần phụ thuộc vào các tầng sau.

- **Resource Definition Resolver** - Chọn Resource Definition phù hợp cho từng logical resource dựa trên type và deployment context.

- **Infrastructure Planner** - So sánh desired state với resource hiện tại để xác định cần create, update hay reuse.

- **Infrastructure Reconciler** - Điều phối việc reconcile infrastructure của từng tầng theo plan đã xác định.

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

- **Application Repository** - Đọc Application Definition từ UC-01.

- **Environment Configuration Repository** - Đọc configuration/reference đã lưu từ UC-02.

- **Resource Instance Repository** - Lưu/đọc trạng thái, reference và dấu vân tay output của infrastructure đã provision để phục vụ reconcile/reuse và lan truyền thay đổi output.

- **Workload Instance Repository** - Lưu/đọc trạng thái hiện hành của từng workload theo environment và deployment target: image đang chạy, trạng thái health và dấu vân tay output.

- **Deployment Repository** - Lưu Deployment Record, image version, target, infrastructure reference, tiến trình theo tầng/thành phần, status và lỗi nếu có.

Luồng responsibility: Web UI → Deployment API → Deployment Orchestrator → Graph Builder → Wave Planner → Resource Definition Resolver → Infrastructure Planner → (với mỗi tầng) Infrastructure Reconciler → Provisioner → Terraform/OpenTofu Runner → Resource Output Collector → Configuration Resolver → Resolved Spec Generator → Score Renderer → score-k8s → Target Adapter → Config/Secret Materializer → CD Integration → Concrete CD Provider → CD System → Kubernetes → Workload Status Provider → Workload Output Collector → Wave Planner (lan truyền) → Resource/Workload Instance Repository.

Deployment Orchestrator chỉ điều phối. Các việc dựng graph, chia tầng và lan truyền thay đổi output, resolve Resource Definition, reconcile infrastructure, resolve configuration, sinh manifest, giao tiếp với CD và thu output nằm ở các component riêng.

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

