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

Draft chỉnh sửa do Web UI sở hữu. Mỗi thao tác chỉnh sửa gửi toàn bộ `ApplicationDefinitionDraft` cần thiết và nhận lại draft mới; backend stateless và không giữ draft xuyên request.

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

Ví dụ:

Workload: backend Type: Backend Service Image Repository: registry.company.local/shop-backend Port: **8080**

Trong workload, Developer khai báo các Environment Variable mà workload cần bằng Add Environment Variable.

Ví dụ:

LOG_LEVEL PAYMENT_API_URL DB_HOST DB_PORT

Developer khai báo các Secret mà workload cần bằng Add Secret.

Ví dụ:

DB_USERNAME DB_PASSWORD PAYMENT_API_KEY

Developer khai báo dependency giữa các thành phần bằng quan hệ depends on.

Ví dụ:

frontend → backend backend  → postgresql

**IDP** hiển thị topology và cấu hình tổng quan của application.

Developer chọn Save Application.

**IDP** kiểm tra dữ liệu, lưu Application Definition và sinh/cập nhật application specification.

Nếu Workload/Resource Requirement/configuration definition đã từng được configuration hoặc deployment history tham chiếu nhưng bị bỏ khỏi draft, **IDP** retire (soft-delete) thay vì hard-delete; history và FK vẫn được giữ.

## Luồng ngoại lệ

A1 – Dữ liệu không hợp lệ

Nếu dữ liệu không hợp lệ, **IDP** hiển thị lỗi để Developer chỉnh sửa.

Ví dụ:

Workload name bị trùng.

Image repository không hợp lệ.

Port không hợp lệ.

Dependency tham chiếu tới thành phần không tồn tại.

## Dữ liệu chính

Nhóm

Dữ liệu

Application

Name, description

Workload

Name, type, image repository, port

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

9. Ví dụ

Application: shop-app

Resources ────────────────────────────

postgresql Type: PostgreSQL

Workload: backend ────────────────────────────

Type: Backend Service Image Repository: registry.company.local/shop-backend

Port: **8080**

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

**IDP** hiển thị danh sách output hợp lệ của resource để Developer lựa chọn.

### Workload Output

Developer chọn workload và output tương ứng.

Ví dụ:

BACKEND_URL

Source: ### Workload Output

Workload: backend

Output: endpoint

Với Secret, Developer có thể:

Nhập giá trị secret theo cơ chế bảo mật của **IDP**.

Hoặc chọn output nhạy cảm của resource.

Khi nhập trực tiếp, Web UI gửi plaintext đúng một lần tới API staging và lập tức thay nó bằng opaque staged reference có TTL/idempotency key. Mọi request chỉnh draft sau đó mang full `ConfigurationDefinitionDraft` và chỉ opaque reference; service không giữ draft hoặc trả plaintext xuyên request.

Developer chọn Save Configuration.

**IDP** kiểm tra và lưu Environment Configuration.

Nếu validation/DB save thất bại, **IDP** gọi compensating revoke idempotent cho staged reference. Nếu save thành công, reference được promote idempotently. Secret Store nằm ngoài DB transaction; TTL, retry và orphan reconciliation giới hạn crash window.

## Luồng ngoại lệ

A1 – Configuration không hợp lệ

**IDP** yêu cầu Developer chỉnh sửa nếu:

Thiếu giá trị bắt buộc.

Resource không tồn tại.

Output được chọn không tồn tại.

Workload output không hợp lệ.

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

Resource Output chỉ lưu reference; giá trị thực tế được resolve trong quá trình deployment.

Giá trị Secret không được hiển thị lại dưới dạng plaintext.

Thay đổi Environment Configuration không tự động làm thay đổi deployment đang chạy.

# UC-03 – Deploy Application

## 1. Mục tiêu

Cho phép Developer triển khai một phiên bản cụ thể của application lên một environment và Kubernetes deployment target.

Image tag/version được xác định tại thời điểm deployment, không phải khi định nghĩa application trong UC-01.

**IDP** chịu trách nhiệm:

- Xác định dependency và resource cần thiết cho deployment.
- Resolve Resource Definition phù hợp theo deployment context.
- Reconcile infrastructure.
- Resolve Environment Configuration, Resource Output và Workload Output.
- Tạo Workload Output plan-time từ graph/workload metadata/deployment context trước khi resolve configuration.
- Sinh Kubernetes manifest.
- Chuyển desired deployment state cho hệ thống Continuous Delivery.

## 2. Actor

**Primary Actor:** Developer

## 3. Tiền điều kiện

- Application Definition hợp lệ.
- Workload cần deploy có Image Repository.
- Environment Configuration cần thiết đã được cấu hình.
- Platform đã có Resource Definition và provisioner phù hợp.
- Deployment target được hỗ trợ.

## 4. Hậu điều kiện

- Deployment ghi nhận chính xác image version của từng workload.
- Infrastructure cần thiết đã được reconcile và sẵn sàng.
- Environment Configuration và các dependency reference được resolve.
- Kubernetes manifest được sinh với đúng image và configuration.
- Desired deployment state được gửi tới CD system.
- Deployment Record được lưu.
- HTTP confirm chỉ trả accepted/tracking sau khi atomically enqueue durable execution job; worker hoàn tất các hậu điều kiện dài hạn.

## 5. Luồng chính

## Developer chọn **Deploy Application**.

## Developer chọn:

    * Environment.
    * Deployment target.

## IDP hiển thị các workload và Image Repository tương ứng.

   Ví dụ:

    ```text
    backend
    registry.company.local/shop-backend

    frontend
    registry.company.local/shop-frontend
    ```

## Developer chọn hoặc xác nhận image tag/version cần deploy cho từng workload.

   Ví dụ:

    ```text
    backend
    registry.company.local/shop-backend:v1.4.3

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

## IDP resolve Resource Definition phù hợp cho các resource trong graph.

## IDP xác định các infrastructure resource cần:

    * Tạo mới.
    * Cập nhật.
    * Hoặc tái sử dụng.

Mọi reusable-instance lookup bắt đầu từ exact active `resource_instance_binding` cho application + environment + logical Resource Requirement + deployment target. Owner columns trên Resource Instance chỉ ghi ownership gốc, không phải read path thay thế. Reuse chéo chỉ hiện diện khi platform đã pre-authorize `SHARED_CONSUMER` binding với policy `EXPLICIT_SHARED`, sharing key và authorization cùng khớp; deployment flow không tự mở rộng sharing scope.

## IDP hiển thị các infrastructure parameter mà Developer được phép override.

## Developer xác nhận và chọn **Deploy**.

**IDP** rebuild typed Infrastructure Plan, kiểm fingerprint và override, rồi CAS/enqueue job trong một DB transaction và trả tracking id. Reconcile, resolve, manifest và publish chạy bất đồng bộ trong worker có lease/retry/idempotency.

Từ đây durable worker thực thi bất đồng bộ.

## IDP reconcile infrastructure theo dependency/resource graph.

## Khi infrastructure resource sẵn sàng, IDP thu thập Resource Output tương ứng.

## IDP dùng Workload Output Resolver tạo output plan-time, ví dụ Kubernetes Service DNS/endpoint, từ deployment graph, workload metadata và target context.

Output chỉ có sau khi workload của chính deployment chạy không được dùng làm input cho deployment đó; vòng tròn bị validation từ chối.

## IDP tiếp tục resolve các dependency và Environment Configuration trong graph.

Ví dụ:

```text
backend.DB_HOST
    → postgresql.host

backend.DB_PASSWORD
    → postgresql.password

frontend.BACKEND_URL
    → backend.endpoint
```

## IDP tạo resolved application specification với image version, configuration và các dependency đã được resolve.

## IDP sử dụng `score-k8s` để sinh base Kubernetes manifest.

## IDP áp dụng target-specific manifest adaptation/patch nếu deployment target yêu cầu cấu hình riêng.

## Environment Variable được chuyển thành ConfigMap hoặc Kubernetes configuration tương ứng.

## Secret được chuyển thành Kubernetes Secret hoặc secret reference phù hợp.

## IDP publish desired deployment state tới CD Integration.

## CD Integration chuyển desired state tới concrete CD implementation để triển khai xuống Kubernetes cluster.

## IDP lưu Deployment Record, bao gồm:

- Environment.
- Deployment target.
- Image version thực tế của từng workload.
- Infrastructure reference.
- Trạng thái deployment.
- External CD delivery reference/status riêng.
- Ba internal progress step: Infrastructure Ready, Configuration Resolved, Manifest Generated.

## 6. Luồng ngoại lệ

### A1 – Deployment input hoặc dependency không hợp lệ

Deployment dừng nếu:

- Image version không hợp lệ.
- Configuration bắt buộc chưa được cấu hình.
- Dependency không thể resolve.
- Resource Definition phù hợp không tồn tại.
- Resource Output hoặc Workload Output được tham chiếu không hợp lệ.

**IDP** hiển thị lỗi để Developer chỉnh sửa.

### A2 – Provisioning hoặc delivery thất bại

Nếu infrastructure provisioning, manifest generation hoặc CD delivery thất bại:

- **IDP** ghi nhận deployment thất bại.
- **IDP** lưu failed internal step cho infrastructure/configuration/manifest; CD publish failure dùng `delivery_status = FAILED` + error summary, không tạo step thứ tư.
- Developer có thể xem chi tiết trong UC-04 – View Deployment Result.

## 7. Dữ liệu chính

| Nhóm                   | Dữ liệu                                                 |
| ---------------------- | ------------------------------------------------------- |
| Deployment             | Application, environment, deployment target             |
| Workload Deployment    | Workload, image repository, image version               |
| Deployment Context     | Cloud provider, region, target-specific input           |
| Deployment Graph       | Workload, resource, dependency, configuration reference |
| Resource Resolution    | Resource, Resource Definition                           |
| Infrastructure Plan    | Typed items/action/parameters/override definitions      |
| Resource Output        | Resource + output                                       |
| Workload Output        | Plan-time DNS/endpoint từ workload metadata/context     |
| Resolved Configuration | Workload, variable/secret, resolved source              |
| Deployment Record      | Lifecycle, delivery reference/status, target, infra refs |
| Deployment Step        | Ba internal step persisted                              |
| Deployment Execution Job | Tracking, overrides, fingerprint, lease/retry          |

## 8. Quy tắc nghiệp vụ

- Image Repository thuộc Workload Definition.
- Image tag/version thuộc Deployment.
- Mỗi deployment phải lưu chính xác image được sử dụng cho từng workload.
- Image version có thể do Developer chọn hoặc được CI cung cấp.
- Dependency/resource graph được xây dựng tại thời điểm deployment dựa trên Application Definition, Environment Configuration và deployment context.
- Resource Definition được resolve theo resource requirement và deployment context.
- Infrastructure được reconcile thay vì luôn tạo mới.
- Resource Output chỉ được sử dụng sau khi resource tương ứng đã được resolve và sẵn sàng.
- Environment Configuration có thể phụ thuộc vào Resource Output hoặc Workload Output.
- Infrastructure Plan, Infrastructure Plan Item và Override Definition là typed transient objects với canonical schema; chỉ fingerprint + algorithm persist, không lưu plan payload.
- `deployment.status` là platform lifecycle `AWAITING_CONFIRMATION/QUEUED/RUNNING/SUBMITTED/FAILED`; CD delivery status có field/enum riêng.
- `CD Synced` và `Application Ready` được UC-04 suy ra live, không persist thành Deployment Step.
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

Image repository và version của từng workload.

Deployment progress.

Infrastructure status.

CD status.

Workload health.

Endpoint nếu có.

Ví dụ:

Deployment #42 Production

frontend registry.company.local/shop-frontend:v2.1.0 Healthy

backend registry.company.local/shop-backend:v1.4.3 Healthy

✓ Infrastructure Ready ✓ Configuration Resolved ✓ Manifest Generated ✓ CD Synced ✓ Application Ready

Ba dấu đầu đọc từ `deployment_step`; hai dấu cuối được suy ra live từ CD/Kubernetes. Nếu reference/prerequisite chưa có, UI hiển thị `PENDING` hoặc `NOT_AVAILABLE`.

## Luồng ngoại lệ

A1 – Deployment thất bại

**IDP** hiển thị failed step, workload/resource liên quan và error summary.

A2 – Workload chưa Healthy

**IDP** hiển thị workload và image version gặp vấn đề.

## Quy tắc nghiệp vụ

UC-04 chỉ cung cấp trạng thái và kết quả deployment.

UC-04 không gọi external provider với reference `NULL`: infrastructure, CD và Kubernetes chỉ được query khi prerequisite tương ứng tồn tại. UC-04 hoàn toàn read-only và không đồng bộ external status vào Deployment lifecycle.

Phải hiển thị image/version thực tế đã sử dụng trong deployment.

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

Chịu trách nhiệm khai báo cấu trúc logic bằng draft do Web UI sở hữu, rồi lưu Application Definition và sinh/cập nhật specification. Backend stateless giữa các request chỉnh draft.

## UC 02 Configure Application Environment

Chịu trách nhiệm khai báo giá trị cấu hình theo environment bằng client-owned draft, gồm direct value hoặc output reference. Secret trực tiếp trở thành opaque staged reference; backend không giữ plaintext/draft qua request.

## UC 03 Deploy Application

Chịu trách nhiệm prepare typed plan và atomically enqueue khi confirm; durable worker biến definition + configuration + context + image thành deployment thực tế qua reconcile, plan-time workload output, manifest và CD publish.

## UC 04 View Deployment Result

Chịu trách nhiệm đọc và hiển thị trạng thái/kết quả của deployment, gồm infrastructure, configuration resolution, CD status, workload health, endpoint và image version thực tế. Không thay đổi application hay infrastructure.

**Ranh giới giữa bốn Use Case**: UC-01 định nghĩa cần gì → UC-02 định nghĩa giá trị theo environment là gì → UC-03 quyết định triển khai chúng như thế nào → UC-04 cho biết kết quả ra sao.

# Bước 2 Xác định các System Operation chính

## UC 01 Create Configure Application

- **createApplication(name, description)** - Tạo client-owned Application Definition draft mới.

- **updateApplication(applicationId)** - Tải definition hiện có thành client-owned draft.

- **addResourceRequirement(applicationDraft, resources)** - Trả draft mới từ full client-owned draft.

- **addWorkload(applicationDraft, workloads)** - Trả draft mới; backend không giữ draft.

- **defineConfigurationRequirement(applicationDraft, requirements)** - Khai báo requirement trên full draft.

- **defineDependency(applicationDraft, dependencies)** - Khai báo dependency trên full draft.

- **validateApplicationDefinition(applicationDraft)** - Kiểm tra full client-owned draft.

- **saveApplicationDefinition(applicationDraft)** - Validate và lưu full client-owned draft; referenced item bị loại sẽ được retire.

- **generateApplicationSpecification(applicationDefinition)** - Sinh/cập nhật specification từ definition đã lưu.

Ở mức Use Case Realization, không tách nhỏ hơn thành các operation như addEnvironmentVariable(), addSecret() hoặc validatePort() để tránh làm sequence diagram quá vụn.

## UC 02 Configure Application Environment

- **selectEnvironment(applicationId, environment)** - Tạo/tải client-owned configuration draft.

- **loadConfigurationRequirements(applicationId)** - Lấy Environment Variable/Secret definitions từ UC-01.

- **setDirectConfigurationValue(configurationDraft, item, valueOrOpaqueReference)** - Gán value hoặc opaque Secret reference trên full client-owned draft.

- **bindResourceOutput(configurationDraft, item, resource, output)** - Trả full draft mới.

- **bindWorkloadOutput(configurationDraft, variable, workload, output)** - Chỉ bind output catalog đánh dấu plan-time resolvable.

- **stageSecret(applicationId, environment, secretDefinitionId, secretValue, idempotencyKey)** - Lưu secret tạm có TTL/idempotency và chỉ trả opaque reference.

- **validateEnvironmentConfiguration(configurationDraft)** - Kiểm tra full client-owned draft và opaque references.

- **saveEnvironmentConfiguration(configurationDraft)** - Lưu full draft và promote/revoke staged secrets theo kết quả.

UC-02 chỉ lưu value hoặc reference, chưa resolve giá trị thật của Resource Output / Workload Output. Việc resolve thuộc UC-03 khi deployment thực sự diễn ra.

## UC 03 Deploy Application

- **createDeployment(applicationId, environment, target, images, context)** - Tạo deployment và plan review.

- **validateDeploymentInput(applicationDefinition, configuration, images, context)** - Kiểm tra input snapshot.

- **buildDeploymentGraph(applicationDefinition, configuration, images, context)** - Dựng dependency/resource graph.

- **resolveResourceDefinitions(graph, context)** - Chọn Resource Definition phù hợp.

- **planInfrastructureChanges(graph, resolutions, scopedInstances)** - Tạo typed create/update/reuse plan.

- **loadInfrastructureOverrides(plan)** - Tạo typed Override Definition list.

- **applyInfrastructureOverrides(plan, overrideValues)** - Validate và áp dụng selected values lên rebuilt plan.

- **confirmDeployment(deploymentId, overrides, idempotencyKey)** - Rebuild/check fingerprint, validate override, atomically enqueue durable job và trả accepted/tracking.

- **reconcileInfrastructure(finalPlan, deploymentId)** - Reconcile và persist scoped Resource Instance/Binding.

- **collectResourceOutputs(infrastructureReferences)** - Thu thập Resource Output sau khi instance ready.

- **resolvePlanTimeWorkloadOutputs(graph, context)** - Tính plan-time Workload Output trước configuration resolution.

- **resolveEnvironmentConfiguration(configuration, resourceOutputs, workloadOutputs)** - Resolve ba nguồn value.

- **generateResolvedApplicationSpecification(applicationDefinition, images, resolvedConfiguration)** - Tạo resolved specification.

- **generateKubernetesManifest(resolvedSpecification)** - Sinh base manifest bằng score-k8s.

- **adaptManifestForTarget(baseManifest, target)** - Áp dụng target-specific adaptation.

- **materializeEnvironmentConfiguration(manifest, resolvedEnvironmentVariables)** - Materialize non-secret configuration.

- **materializeSecretConfiguration(manifest, resolvedSecrets)** - Materialize Secret/reference không log plaintext.

- **publishDesiredDeploymentState(desiredState, deploymentId, idempotencyKey)** - Publish idempotently và nhận delivery reference/status.

- **saveDeploymentRecord(deploymentId, infrastructureReferences, deliveryReference, deliveryStatus, lifecycleStatus)** - Upsert record cuối.

Chuỗi chính: tạo deployment → dựng graph → resolve resource → lập typed plan/override → HTTP confirm rebuild/check fingerprint và atomically enqueue → worker đánh dấu `RUNNING`, rebuild/check fingerprint → reconcile infra → resolve output/config → sinh resolved spec → sinh base manifest → adapt theo target → materialize config/secret → publish sang CD → lưu deployment record.

## UC 04 View Deployment Result

- **listDeployments(applicationId)** - Lấy lịch sử deployment của application.

- **getDeploymentDetail(deploymentId)** - Lấy detail/prerequisite references.

- **getDeploymentProgress(deploymentId)** - Lấy ba internal persisted steps.

- **getInfrastructureStatus(infrastructureReferences)** - Chỉ gọi khi references không rỗng.

- **getCDStatus(deliveryReference)** - Chỉ gọi khi delivery reference tồn tại.

- **getWorkloadStatus(target, workloads)** - Chỉ gọi sau publish prerequisite.

- **getDeploymentEndpoints(target, workloads)** - Chỉ gọi sau publish prerequisite.

- **getDeploymentImages(deploymentId)** - Lấy image snapshot thực tế.

- **getDeploymentFailureDetail(deploymentId)** - Lấy failed internal step/error summary.

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

- **Environment Configuration Service** - Stateless; nhận/trả full client-owned draft, stage/promote/revoke opaque secret references.

### Domain components

- **Resource Output Catalog / Resource Definition Query** - Cung cấp danh sách output hợp lệ mà một resource có thể expose để Developer lựa chọn.

- **Workload Output Catalog** - Chỉ liệt kê output `PLAN_TIME` mà workload expose cho same-deployment binding.

- **Environment Configuration Validator** - Kiểm tra direct value, resource reference, workload reference và output được chọn có hợp lệ hay không.

### Integration abstractions

- **Secret Store / Secret Management Adapter** - Stage secret bằng idempotency key + TTL, trả opaque reference, rồi promote/revoke idempotently.

Secret backend implementation cụ thể chưa được chốt, vì vậy tài liệu chưa liệt kê thành phần thuộc nhóm Integration implementations cho UC-02.

### Persistence

- **Application Query / Application Repository** - Lấy danh sách workload, Environment Variable, Secret và dependency đã được khai báo từ UC-01.

- **Environment Configuration Repository** - Lưu configuration và các reference riêng theo từng environment.

Luồng trách nhiệm: Web UI → Configuration API → Environment Configuration Service → Application/Output Catalog → Validator → Secret Store + Environment Configuration Repository.

UC-02 chưa cần Configuration Resolver. Hệ thống chỉ lưu reference như DB_HOST → postgresql.host; giá trị thật chỉ được resolve trong UC-03 khi deploy.

## UC 03 Deploy Application

### Boundary/UI

- **Web UI** - Cho Developer chọn environment, deployment target, image version, xem infrastructure plan, nhập override và xác nhận deploy.

- **Deployment API / Controller** - Nhận request từ UI, validate ở mức request và chuyển sang Deployment Orchestrator.

### Application services

- **Deployment Orchestrator** - Dựng/rebuild plan và accept/enqueue; không chạy tác vụ dài trong HTTP confirm.

- **Deployment Worker** - Claim durable job và thực thi reconcile → resolve → manifest → publish → record với retry/idempotency.

### Domain components

- **Deployment Graph Builder** - Dựng dependency/resource graph từ Application Definition, Environment Configuration và deployment context.

- **Resource Definition Resolver** - Chọn Resource Definition phù hợp cho từng logical resource dựa trên type và deployment context.

- **Infrastructure Planner** - So sánh desired state với resource hiện tại để xác định cần create, update hay reuse.

- **Infrastructure Plan / Item / Override Definition** - Typed transient canonical input cho fingerprint; payload không persist.

- **Infrastructure Reconciler** - Điều phối việc reconcile infrastructure theo plan đã xác định.

- **Resource Output Resolver / Collector** - Thu thập output từ infrastructure đã provision, ví dụ host, port, username, password.

- **Workload Output Resolver** - Tạo Service DNS/endpoint plan-time từ graph/workload metadata/deployment context và từ chối circular/runtime-only input.

- **Environment Configuration Resolver** - Resolve direct value, Resource Output reference và Workload Output reference thành configuration thực tế cho deployment.

- **Resolved Specification Generator** - Tạo resolved application specification chứa image version, dependency và configuration đã resolve.

- **Manifest Generator / Score Renderer** - Gọi score-k8s để sinh base Kubernetes manifest từ resolved specification.

- **Target Manifest Adapter** - Áp dụng patch/adaptation riêng theo deployment target.

- **Environment Configuration Materializer** - Chuyển Environment Variable đã resolve thành Kubernetes configuration tương ứng, ví dụ ConfigMap.

- **Secret Materializer** - Chuyển Secret thành Kubernetes Secret hoặc secret reference phù hợp mà không làm lộ plaintext.

### Integration abstractions

- **Provisioner Adapter / Provisioner Interface** - Abstraction cho cơ chế provision infrastructure; implementation cụ thể có thể gọi Terraform/OpenTofu/module tương ứng.

- **CD Integration / CD Provider Interface** - Abstraction để publish desired deployment state mà không phụ thuộc trực tiếp vào Argo CD, Flux hay implementation cụ thể.

### Integration implementations

- **Concrete CD Provider** - Implementation cụ thể của CD abstraction, ví dụ Argo CD Adapter hoặc Flux Adapter.

### External systems

- **Terraform/OpenTofu Runner** - Thực thi Terraform/OpenTofu module để tạo hoặc cập nhật infrastructure theo yêu cầu từ Provisioner Adapter.

- **score-k8s** - Sinh base Kubernetes manifest từ resolved application specification.

- **CD System** - Hệ thống CD bên ngoài, ví dụ Argo CD hoặc Flux, nhận desired deployment state và đồng bộ xuống Kubernetes.

- **Kubernetes Cluster** - Deployment target cuối nơi workload thực sự chạy.

### Persistence

- **Application Repository** - Đọc Application Definition từ UC-01.

- **Environment Configuration Repository** - Đọc configuration/reference đã lưu từ UC-02.

- **Resource Instance Repository** - Lưu Resource Instance/Binding và query exact application + environment + logical requirement + target; shared reuse cần policy/key.

- **Deployment Repository** - Lưu Deployment Record, image version, target, infrastructure reference, step/status và lỗi nếu có.

- **Deployment Execution Job / Outbox** - Lưu durable accepted work, selected overrides, fingerprint, lease/attempt và idempotency key.

Luồng responsibility: Web UI → Deployment API → Deployment Orchestrator → atomic Job/Outbox enqueue → Deployment Worker → Graph/Planner → Infrastructure Reconciler → Resource + Workload Output Resolvers → Configuration Resolver → Spec/Manifest/Materializers → CD Integration → Deployment Record.

Deployment Orchestrator chỉ prepare/rebuild/accept; Deployment Worker điều phối tác vụ dài. Các domain/integration component vẫn sở hữu logic chuyên biệt.

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

Luồng responsibility: Web UI → Deployment Query API → Deployment Query Service → Deployment Repository + Resource Repository + CD Status Provider → Concrete CD Provider → CD System + Kubernetes Adapter → Kubernetes Cluster → Result Aggregator → Web UI.

UC-04 không dùng Deployment Orchestrator để thực hiện hành động. UC-04 đi theo query path riêng vì chỉ đọc và tổng hợp trạng thái, không reconcile infrastructure hoặc trigger deployment.
