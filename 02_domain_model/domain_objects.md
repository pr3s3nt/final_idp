# Step 2: Domain Objects

Tài liệu này mô tả domain model được rút ra từ các bảng **Dữ liệu chính**, business rules của UC-01 đến UC-04 và các main-flow sequence diagram. Model chỉ chứa business entity, value object, generated artifact và execution-time data holder; không chứa Boundary, Controller, Service, Validator, Resolver, Generator, Adapter hay Repository.

Các aggregate root chính là **Application Definition**, **Environment Configuration**, **Deployment** và **Resource Instance**. `Resource Definition` là dữ liệu catalog độc lập do platform quản lý; `Application Specification` là artifact được sinh từ `Application Definition`.

| Domain Object | Thuộc tính chính | Mô tả | Aggregate nó thuộc về |
|---|---|---|---|
| Application Definition | `applicationId`, `name`, `description`, `createdAt`, `updatedAt` | Mô tả cấu trúc logic của application; là nguồn cho workload, logical resource, dependency và configuration requirement. | **Application Definition** (aggregate root) |
| Workload | `workloadId`, `name`, `type`, `imageRepository`, `port`, `exposedOutputs` | Một workload thuộc application. Chỉ giữ image repository; image tag/version cụ thể thuộc deployment. `exposedOutputs` là các output logic có thể được tham chiếu, ví dụ `endpoint`. | Application Definition |
| Resource Requirement | `resourceRequirementId`, `name`, `resourceType` | Nhu cầu resource ở mức logic, ví dụ PostgreSQL hoặc Redis; không chứa cách provision. | Application Definition |
| Environment Variable Definition | `variableDefinitionId`, `name`, `required` | Khai báo một Environment Variable mà workload cần, chưa có giá trị theo environment. | Application Definition (qua Workload) |
| Secret Definition | `secretDefinitionId`, `name`, `required` | Khai báo một Secret mà workload cần và phân biệt nó với Environment Variable thông thường. | Application Definition (qua Workload) |
| Dependency | `dependencyId`, `sourceWorkloadId`, `targetType`, `targetId` | Quan hệ `depends on` từ một workload tới đúng một workload hoặc Resource Requirement khác. | Application Definition |
| Application Specification | `specificationId`, `applicationId`, `format`, `content`, `version`, `updatedAt` | Generated artifact, ví dụ `score.yaml`, được sinh/cập nhật từ Application Definition. | Application Definition (generated artifact) |
| Environment Configuration | `environmentConfigurationId`, `applicationId`, `environment`, `createdAt`, `updatedAt` | Cấu hình của một application cho đúng một environment; thay đổi object này không tự động thay đổi deployment đang chạy. | **Environment Configuration** (aggregate root) |
| Environment Variable | `environmentVariableId`, `workloadId`, `variableName` | Binding theo environment cho một Environment Variable Definition của workload. Giá trị nằm trong một `Configuration Value`. | Environment Configuration |
| Secret | `secretId`, `workloadId`, `secretName`, `secretReference` | Binding theo environment cho một Secret Definition. Khi Developer nhập secret trực tiếp, object chỉ giữ reference do Secret Store trả về; không giữ plaintext. Với sensitive resource output, source được biểu diễn bằng `Resource Output Reference`. | Environment Configuration |
| Configuration Value | `valueSourceType` | Abstract value object biểu diễn nguồn của configuration: direct value, Resource Output Reference hoặc Workload Output Reference. | Environment Configuration |
| Direct Configuration Value | `value` | Giá trị trực tiếp của Environment Variable thông thường. Không được dùng để persist plaintext Secret. | Environment Configuration |
| Resource Output Reference | `resourceRequirementId`, `outputName` | Reference bền vững tới output logic của một Resource Requirement, ví dụ `postgresql.host`; giá trị thực chỉ được resolve khi deploy. | Environment Configuration |
| Workload Output Reference | `workloadId`, `outputName` | Reference bền vững tới output logic mà workload expose, ví dụ `backend.endpoint`; giá trị được resolve trong deployment. | Environment Configuration |
| Resource Definition | `resourceDefinitionId`, `name`, `resourceType`, `provisionerReference`, `supportedContexts`, `exposedOutputs`, `sensitiveOutputs` | Định nghĩa do platform cung cấp để map logical Resource Requirement sang provisioner phù hợp theo deployment context, đồng thời công bố normal/sensitive outputs hợp lệ. | Platform Resource Definition catalog (độc lập) |
| Deployment | `deploymentId`, `applicationId`, `environment`, `deploymentTarget`, `planFingerprint`, `planFingerprintAlgo`, `status`, `createdAt`, `updatedAt` | Một lần triển khai application cụ thể. Đây là lifecycle aggregate giữ context, actual workload images và record theo dõi. Infrastructure Plan vẫn là `TRANSIENT`; chỉ fingerprint và phiên bản thuật toán của nó được persist để kiểm tra plan rebuild khi confirm. | **Deployment** (aggregate root) |
| Workload Deployment | `workloadDeploymentId`, `workloadId`, `imageRepository`, `imageVersion` | Snapshot image thực tế của một workload trong deployment. `imageVersion` được Developer chọn hoặc CI cung cấp tại thời điểm deploy. | Deployment |
| Deployment Context | `cloudProvider`, `region`, `targetSpecificInput` | Context dùng để resolve Resource Definition và thực hiện target-specific adaptation cho deployment target đã chọn. | Deployment |
| Deployment Graph | `deploymentId`, `workloadIds`, `resourceRequirementIds`, `dependencyIds`, `configurationReferenceIds` | Dependency/resource graph được dựng từ Application Definition, Environment Configuration, workload images và Deployment Context cho một execution. | Deployment (execution-scoped) |
| Resource Resolution | `resourceRequirementId`, `resourceDefinitionId`, `resolutionStatus`, `reason` | Kết quả chọn Resource Definition phù hợp cho một logical Resource Requirement; có thể dẫn tới create, update hoặc reuse Resource Instance. | Deployment (execution-scoped) |
| Resource Instance | `resourceInstanceId`, `resourceDefinitionId`, `deploymentTarget`, `infrastructureReference`, `providerStateReference`, `status`, `createdAt`, `updatedAt` | Đại diện durable cho infrastructure đã provision/reconcile để hệ thống có thể theo dõi và reuse. | **Resource Instance** (aggregate root) |
| Resource Output | `resourceInstanceId`, `outputName`, `resolvedValue`, `sensitive` | Runtime output được collector nạp từ Resource Instance sau khi resource sẵn sàng và dùng để resolve configuration trong một deployment. | Resource Instance (runtime view, execution-scoped) |
| Resolved Configuration | `deploymentId`, `resolvedEnvironmentVariables`, `resolvedSecrets`, `resolvedDependencies` | Snapshot in-memory sau khi direct values và output references đã được resolve cho deployment hiện tại. | Deployment (execution-scoped) |
| Resolved Specification | `deploymentId`, `format`, `workloadImages`, `resolvedDependencies`, `generatedAt` | Resolved application specification chứa image version, configuration và dependency đã resolve; là input cho `score-k8s`. | Deployment (execution-scoped) |
| Deployment Record | `deploymentRecordId`, `environment`, `deploymentTarget`, `infrastructureReferences`, `deliveryReference`, `status`, `errorSummary`, `createdAt`, `updatedAt` | Durable record phục vụ deployment history/result, bao gồm target, actual images qua Workload Deployment, infrastructure references, delivery status và lỗi tổng quát. | Deployment |
| Deployment Step | `deploymentStepId`, `sequenceNumber`, `stepName`, `status`, `relatedComponentReference`, `errorSummary`, `startedAt`, `completedAt` | Trạng thái/progress của từng bước; giữ failed step và workload/resource liên quan khi deployment thất bại. | Deployment (qua Deployment Record) |

## Invariants chính

- Một `Application Definition` có `1..*` Workload, `0..*` Resource Requirement và `0..*` Dependency.
- `Environment Variable Definition` và `Secret Definition` luôn thuộc đúng một Workload.
- Mỗi Dependency có đúng một source Workload và đúng một target: Workload hoặc Resource Requirement.
- Một `Environment Configuration` thuộc đúng một Application Definition và một environment; mỗi configured Environment Variable có đúng một Configuration Value.
- Một configured Secret có đúng một source: `secretReference` hoặc sensitive `Resource Output Reference`. Plaintext Secret không thuộc persistent domain model.
- Một `Deployment` có đúng một Deployment Context và `1..*` Workload Deployment; mỗi Workload Deployment ghi image version thực tế.
- Resource Output chỉ được dùng sau khi Resource Instance tương ứng đã resolve/reconcile và ở trạng thái sẵn sàng.
