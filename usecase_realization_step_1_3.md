# Use case realization — MVP profile

Phạm vi hiện tại: [MVP_SCOPE.md](MVP_SCOPE.md). Quyết định triển khai: [MVP_DEPLOYMENT_DESIGN.md](MVP_DEPLOYMENT_DESIGN.md). Bản này dùng Go, Terraform, Argo CD, một environment dev và một target Kubernetes trên AWS. **UC3 happy path phải deploy workloads/database lên AWS thật**; kind chỉ hỗ trợ phát triển/test, không là target nghiệm thu. Cấu hình dịch vụ AWS cụ thể/chi phí cần chốt trước provision.

[Prompt triển khai](plab_mvp.md) giới hạn mốc đầu ở deploy lần đầu + CRUD + cleanup. REUSE/redeploy, recovery tool và UC-04 đầy đủ bên dưới là thiết kế mở rộng, không yêu cầu code hết trong mốc này.

## UC-01 — Create / Configure Application (editor deferred)

Mục tiêu sản phẩm: Developer khai báo Application Definition gồm workloads, logical resource requirements, configuration definitions và dependencies; save sinh Application Specification. Workload có repository/port, image version được chọn ở deploy. Definition chỉ mô tả resource logic, không yêu cầu Developer viết Terraform/Kubernetes.

MVP chuẩn bị fixture gồm frontend, backend và PostgreSQL requirement. Internal fixture loader validate và save aggregate/specification/configuration atomic theo C1. Full Web UI/client-owned draft và edit endpoints chưa triển khai. Sơ đồ UC-01 hiện giữ làm reference cho editor tương lai; không dùng để suy ra MVP worker behavior.

Referenced identity bị bỏ khỏi source được retire, không hard-delete phá history/configuration FK. Deployment đã chuẩn bị có source snapshot riêng; source edit không làm đổi deployment đó.

## UC-02 — Configure Application Environment (editor deferred)

Mục tiêu sản phẩm: bind variable vào direct non-secret value, Resource Output hoặc Workload Output. Full editor và secret staging flow trong diagram UC-02 là tương lai; R3 chưa được giải quyết cho tính năng đó.

MVP fixture chứa backend DB_HOST/DB_PORT từ PostgreSQL output, credentials là permanent Kubernetes Secret reference, frontend backend URL là PLAN_TIME Workload Output. Cấu hình chỉ lưu source/reference; worker resolve lúc deploy. Fixture có thể tạo empty configuration nếu không có requirement, nhưng app không configuration chưa thuộc demo nghiệm thu (R9 deferred).

Secret metadata: target/namespace/name/UID/key, immutable Secret do bootstrap tạo trước deploy; check tại prepare/confirm/precheck. Không stage/promote/revoke, không resolve plaintext vào DB/manifest/Terraform state.

## UC-03 — Deploy Application

### Actor, input và hậu điều kiện

Developer gọi API IDP đã xác thực (được phép chạy local); application/environment/target AWS được allowlist. Input gồm fixture identity, AWS account/region/cluster context và image version của frontend/backend. Registry mà cloud truy cập được resolve image thành digest trước snapshot. Vị trí API không quyết định nơi chạy ứng dụng: frontend/backend/PostgreSQL phải ở AWS.

Hậu điều kiện HTTP confirm là durable tracking ID + QUEUED job. Hậu điều kiện worker success là infrastructure ready, configuration resolved, manifest generated và Argo Application source được acknowledge; Deployment SUBMITTED không khẳng định workload Healthy.

### Luồng chính

Prerequisite: Terraform bootstrap hạ tầng target AWS bằng state riêng; namespace/Secret/metadata/Argo CD sẵn sàng, registry và storage truy cập được. Database ứng dụng chưa được bootstrap sẵn; UC3 phải provision thật.

1. createDeployment đọc source nhất quán, lưu immutable snapshot, image digests/context, typed transient plan và fingerprint; trả AWAITING_CONFIRMATION để review.
2. Planner tìm resource qua exact active binding ở mọi status. CREATE khi chưa có hoặc recovery-approved reserved identity; compatible READY -> REUSE. Unknown/failed/busy -> recovery required; update/shared/overrides bị từ chối trong MVP.
3. confirmDeployment nhận expectedPlanFingerprint, overrides={} và idempotencyKey. Tìm accepted request trước; cùng key/hash trả cùng tracking ID. So client token với stored và rebuilt; mismatch trả PLAN_CHANGED, kể cả khi DB đã refresh sau một response bị mất.
4. Transaction accept lấy deployment scope guard, đổi lifecycle, tạo record/ba steps/job. Một worker dưới exclusive host lock claim đúng một lần.
5. Worker đọc snapshot, kiểm lại current catalog/resource plan trước side effects, reserve deterministic resource/state/binding trước Terraform.
6. Terraform CREATE PostgreSQL trên target AWS (inspect REUSE thuộc mốc sau); checkpoint READY, chuyển từ infrastructure step sang configuration step atomically.
7. Trong CONFIGURATION_RESOLVED: đọc provider outputs từ durable state; tính Workload Output từ naming policy; resolve source references.
8. Trong MANIFEST_GENERATED: generate spec, score-k8s render, target adaptation và materialize config/secretKeyRef. Final names/digests/deployment-id annotation phải khớp snapshot.
9. Publish OCI artifact, lưu digest/expected Argo Application trước Argo API write, upsert owned Application với targetRevision=digest và destination cluster AWS đã xác minh.
10. Một transaction hoàn tất delivery acknowledgment, Deployment/Record SUBMITTED, job SUCCEEDED và release guard.

Nghiệm thu tiếp tục chờ đúng revision Ready trên AWS, chạy CRUD và lưu bằng chứng cloud; sau đó cleanup đúng tài nguyên task theo scope. Chỉ SUBMITTED hoặc CRUD trên kind không đủ.

### Ngoại lệ và recovery

- Invalid input/secret/ref/unsupported override: reject trước execution, không có provider side effect.
- PLAN_CHANGED: giữ awaiting, trả plan/token mới để review. Same accepted key/different payload: IDEMPOTENCY_KEY_REUSED.
- PLAN_STALE_AFTER_ACCEPT: fresh worker preflight fail, không apply.
- Provider/render/config error: ghi đúng phase, giữ resource đã có; pending steps -> SKIPPED. CD timeout chưa rõ outcome -> UNKNOWN/RECOVERY_REQUIRED.
- Worker chết: supervisor xác minh process group cũ dừng; mark interrupted job FAILED, giữ guard; operator inspect state/Argo và audit recovery trước khi cho deployment mới.
- Không tự replay pipeline, không tự xóa/recreate database, không dùng CREATE fingerprint cũ để đánh giá trạng thái sau apply.

### System và internal operations

Public: createDeployment; confirmDeployment. Internal: fixtureImport, buildInputSnapshot, buildDeploymentGraph, resolveResourceDefinitions, findScopedInstances, planInfrastructureChanges, claimQueuedJob, reconcileInfrastructure, collectResourceOutputs, resolvePlanTimeWorkloadOutputs, resolveEnvironmentConfiguration, generateResolvedApplicationSpecification, generateKubernetesManifest, adaptAndMaterialize, publishDesiredDeploymentState, completeExecution, failExecution, markInterruptedJobs, recoverDeployment.

Contracts C1–C12 là normative; không đếm internal operations thành public system operations. Provider workers không đọc lại mutable Application/Configuration Repository sau prepare.

## UC-04 — View Deployment Result

Public queries: listDeployments, getDeploymentDetail, getDeploymentFailureDetail. Query Service read-only; history từ metadata DB, live provider được gọi khi prerequisite/reference tồn tại.

- Hiển thị lifecycle, ba internal steps, image digest snapshot, resource state, expected/observed Argo revision, endpoint và sanitized error/recovery requirement.
- Application UID/source khác expected -> NOT_AVAILABLE với REPLACED/SUPERSEDED; observed revision chưa đúng hoặc chưa Synced -> PENDING.
- Chỉ query workload readiness sau khi revision correlated. APPLICATION_READY yêu cầu template deployment-id, image digests, observedGeneration và rollout counts của đúng version; health cũ không đủ.
- CD_SYNCED/APPLICATION_READY là marker view, không có step row hoặc write vào lifecycle.
- Provider unavailable -> PENDING với reason, không sửa DB thành FAILED. SUBMITTED chỉ là platform delivery acceptance.
- Infrastructure identities chưa có -> PENDING/NOT_AVAILABLE; không gọi provider với NULL.
- Logs/metrics/traces dashboard nâng cao ngoài scope.

## Layer ownership

| Layer | Participating components |
|---|---|
| Boundary | API client/Web UI; Deployment API; Query API; operator recovery tool |
| Application services | Orchestrator prepare/confirm; Worker execute; Query Service; Recovery Service |
| Domain | Snapshot/Graph Builders, Definition Resolver, Planner, Reconciler, Output/Configuration Resolvers, Spec/Manifest Generators, Materializers, Result Aggregator |
| Integration | Secret Reference Adapter; Go Provisioner Adapter; Go Argo CD Adapter; Workload Status Provider |
| External | Terraform + durable state, score-k8s, OCI registry, Argo CD, Kubernetes |
| Persistence | Source repositories for fixture/prepare; Deployment Repository for snapshot/job/record/guard/recovery; Resource Repository for identity/binding/state |

Definition/configuration/schema/domain source types retained for future editor are not permission to enable staged secrets, shared consumers, in-place database update or automatic replay in MVP.
