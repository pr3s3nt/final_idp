# Thiết kế deployment MVP

Tài liệu này chốt các quyết định triển khai của [MVP_SCOPE.md](MVP_SCOPE.md). Domain, schema, contracts, sequence, VOPC và state machines ở các thư mục 01–06 hiện thực cùng các quy tắc dưới đây. UC-01/02 đầy đủ và shared-resource administration là thiết kế mở rộng, không phải API cần code trong mốc này.

Mốc triển khai hiện tại là **UC3 happy path trên AWS thật**, theo [prompt triển khai](plab_mvp.md). Các mục REUSE/redeploy, recovery tool và query đầy đủ bên dưới giữ làm thiết kế mốc sau, không buộc code hết trước demo lần đầu. Kind/local chỉ phục vụ phát triển hoặc chạy control plane IDP, không thay thế target cloud. Target đã chốt là EKS 1.35 với một `t3.small` On-Demand; Aurora Serverless v2 0.5–1 ACU, ECR và Argo CD local/kind.

## 1. Ranh giới và ownership

- IDP Go API/worker chạy trên một máy; một worker duy nhất, được supervisor quản lý cả process group chứa Terraform/renderer. Metadata PostgreSQL, Terraform state directory và worker lock nằm trên storage bền vững, tách database ứng dụng.
- Target được allowlist là một cluster Kubernetes trên AWS với account/region/cluster identity/context đã xác minh, namespace ứng dụng `idp-demo-dev`; mọi adapter truyền target rõ ràng, không dùng kubeconfig current-context. Không gắn target nghiệm thu vào kind hoặc chấp nhận fallback local.
- Terraform bootstrap sở hữu target Kubernetes, private network và prerequisite với state riêng. Resource Definition Resolver chọn `postgres-aurora` cho AWS; provisioner sở hữu Aurora cluster/instance và RDS-managed Secrets Manager credential bằng state resource riêng. Với kind/local, resolver chọn `postgres-kubernetes` và provisioner sở hữu StatefulSet/PVC/Service trong cluster. Argo CD sở hữu frontend/backend cùng ExternalSecret materialization trên AWS. Không có object do hai controller quản lý và không tạo sẵn database ứng dụng ngoài UC3.
- Bootstrap tạo EKS 1.35 với một `t3.small` On-Demand trong public subnet, bật private endpoint cho node và giới hạn public API theo IP runner; không tạo NAT Gateway/load balancer. Aurora Serverless v2 0.5–1 ACU dùng private subnet group hai AZ và security group chỉ nhận từ EKS. ECR phục vụ images/OCI; Argo CD local phải đăng ký AWS destination rõ ràng. Giá nền kiểm tra khoảng 0.2314 USD/giờ tại `ap-southeast-1`, chưa gồm storage/I/O/data transfer/tax.
- Một resource scope là `(application_id, environment, resource_requirement_id, deployment_target)`. Một deployment scope là `(application_id, environment, deployment_target)`.
- MVP chỉ CREATE/REUSE database với parameter cố định từ catalog; `allowed_overrides = {}`, confirm chỉ nhận `overrides = {}`. UPDATE/resize/replace, cross-application sharing và secret rotation nằm ngoài scope. Yêu cầu khác trả `UNSUPPORTED_MVP_OPERATION`, không ngầm thay database.

## 2. Input snapshot và tên target

`createDeployment` đọc Application Definition và Environment Configuration trong một DB transaction REPEATABLE READ, bao gồm tất cả child rows đang active. Fixture/configuration write phải atomic; snapshot không được ghép dữ liệu từ hai lần save. Cấu hình tham chiếu retired/missing target bị từ chối; app không có requirement có thể dùng empty configuration được fixture loader tạo, nhưng chưa thuộc demo nghiệm thu R9.

Tạo `deployment_input_snapshot` 1:1 với Deployment: `schema_version`, `application_definition` (topology, outputs, ports, requirement identities), `environment_configuration` (direct non-secret source và logical output/secret references), `render_context` (namespace, naming policy, renderer/adapter versions, target parameters), `input_fingerprint`. Image tag được resolve thành digest ở prepare và lưu cả tag lẫn digest trong Workload Deployment; Deployment Context cũng bất biến. Hash snapshot bao gồm các image digests và context từ hai bảng này.

Snapshot là bản sao **source input**, không chứa resolved Resource Output, credential hoặc Infrastructure Plan. DB role không được UPDATE/DELETE snapshot và input của deployment đã tạo; migration phải enforce bằng permission/trigger, không chỉ quy ước ở Go. Các source table tiếp tục sửa/retire được mà không đổi snapshot. Confirm/worker/query đọc snapshot, không đọc lại current application/configuration. Muốn đổi source/image/target phải tạo Deployment mới.

Naming policy `mvp-v1` sinh Service/Deployment names từ application/workload identity, ổn định giữa các deployment. Pod template có annotation `idp.deployment-id` bằng deployment UUID. Backend Service URL được tính từ chính tên/namespace/port này; target adapter phải kiểm manifest cuối còn khớp naming policy. Frontend Go nhận URL đó, proxy `/api`; trình duyệt không dùng Service DNS.

## 3. Secret reference và materialization

Với AWS, Aurora bật RDS-managed master password. Provider output chỉ trả Secrets Manager ARN/reference, không trả secret value. Snapshot giữ logical `RESOURCE_SECRET` intent gồm resource requirement/output, remote property và Kubernetes destination name/key. Sau infrastructure READY, Secret Materializer kiểm tra ARN thuộc resource/target mong đợi và sinh ExternalSecret reference; backend vẫn dùng `secretKeyRef`. IDP, Terraform input/state và OCI artifact không chứa plaintext credential.

Với kind/local, bootstrap tạo Kubernetes Secret `immutable: true`; snapshot giữ `target + namespace + name + uid + key`, adapter xác minh tồn tại/UID/immutable/key trước confirm và worker. Không xóa/tạo lại hoặc rotate secret trong một execution. MVP không claim distributed atomicity giữa Kubernetes, AWS Secrets Manager và metadata DB.

Không có stage/promote/revoke API trong MVP. R3 được hoãn theo phạm vi, không được đánh dấu đã sửa giao thức staging. Fixture loader chỉ lưu references tới permanent Secret đã tồn tại, không công bố staged configuration.

## 4. Prepare, fingerprint và confirm

Infrastructure Planner đọc exact active binding **ở mọi resource status**. Không binding: CREATE; existing PLANNED có recovery_verified=true sau operator inspection: CREATE trên cùng reserved identity. READY với owner/target/definition fingerprint/parameter fingerprint tương thích: REUSE. PLANNED/PROVISIONING/FAILED: `RESOURCE_RECOVERY_REQUIRED`; READY không tương thích: `RESOURCE_CHANGE_UNSUPPORTED`. Không chuyển query rỗng do lọc READY thành CREATE.

Plan canonical version `sha256-mvp-v1` gồm input fingerprint; sorted requirement IDs; Resource Definition identity và các field type, provisioner/module version, contexts, defaults, overrides, outputs; action; instance ID/version; scope; parameter fingerprint. Object keys sort lexical, array có nghĩa tập hợp sort theo stable ID, integer dạng thập phân, string giữ nguyên, không whitespace/timestamps. MVP cấm float/NaN và normalize units thành integer từ input validator. Chỉ hash + algorithm persist; typed plan transient.

Prepare không provision/reserve instance, persist Deployment AWAITING_CONFIRMATION + snapshot + images/context atomically và trả plan/fingerprint. Resource/catalog có thể đổi giữa prepare và confirm nên cần rebuild. Catalog thay đổi trước execute cũng phải được phát hiện; current app/configuration không được đọc lại.

`confirmDeployment(deploymentId, expectedPlanFingerprint, overrides, idempotencyKey)`:

1. Authenticate/authorize. Tính `request_fingerprint` trên deploymentId + expected fingerprint + canonical overrides. Tìm job đã accept trước mọi guard trạng thái/rebuild: cùng key và request hash trả cùng trackingId dù đã FAILED/SUBMITTED; cùng key khác payload trả `IDEMPOTENCY_KEY_REUSED`; key khác khi deployment đã có job trả `ALREADY_ACCEPTED`.
2. Lock Deployment row, đọc lại accepted job và áp dụng lại kiểm tra key/hash ở bước 1 trước khi xét lifecycle (đóng race giữa lookup đầu và lock). Nếu chưa accept thì lock guard row theo deployment scope. Chỉ một EXECUTING/RECOVERY_REQUIRED owner mỗi scope. Bận trả `SCOPE_BUSY` hoặc `RECOVERY_REQUIRED`.
3. Rebuild từ snapshot, catalog và all-status bindings. Nếu expected fingerprint khác stored hoặc rebuilt fingerprint, refresh stored fingerprint khi còn AWAITING_CONFIRMATION và trả 409 PLAN_CHANGED + plan mới. Không accept chỉ vì rebuilt khớp DB. Retry request mang token cũ vẫn phải bị từ chối.
4. Revalidate secret refs và overrides rỗng. Cùng transaction: CAS status **và expected fingerprint**, set guard EXECUTING, insert job QUEUED, record QUEUED/NOT_PUBLISHED và ba step PENDING. Commit rồi trả 202 trackingId.

Guard row được insert-if-absent rồi SELECT FOR UPDATE; mọi path cần cả hai row đều lock Deployment trước guard. Hai deployment khác nhau trong cùng scope được serialize bởi guard. Không giữ DB transaction qua external provider calls; read-only preflight có timeout ngắn. Unique job/deployment bảo vệ idempotency.

## 5. Worker và resource identity

Worker giữ host-level exclusive lock suốt process lifetime. Supervisor phải dừng/đợi cả process group cũ trước khi worker mới được chạy; Terraform giữ backend state lock. Heartbeat chỉ để chẩn đoán, hết heartbeat không cho phép worker khác chạy lại job. Nếu chưa chứng minh process/provider cũ dừng, hệ thống giữ scope blocked.

Claim chỉ nhận QUEUED job, không nhận lại CLAIMED: một transaction chuyển job CLAIMED + worker_run_id, lifecycle/record RUNNING, phase INFRASTRUCTURE và step INFRASTRUCTURE_READY RUNNING. Các subsequent DB writes yêu cầu job CLAIMED và matching worker_run_id.

Worker rebuild/check accepted plan **một lần trước side effects**, đọc snapshot và current catalog/bindings. Mismatch thành PLAN_STALE_AFTER_ACCEPT, failed infrastructure step, job/deployment FAILED và release guard nếu chưa có side effect. Worker không replay job và không so CREATE fingerprint sau khi chính nó đã tạo instance.

Trước Terraform: transaction `reserveResource` tạo PLANNED instance + OWNER binding + record association, hoặc xác nhận existing READY binding. Lưu deterministic provider_state_reference và tên resource từ scope hash, không từ deployment ID; state path phải tồn tại trên durable storage. Sau đó ghi PROVISIONING trước apply. Kết quả READY/reference/version và step completion được checkpoint trước phase sau.

REUSE phải kiểm tra provider objects/identity và parameters thực tế; nếu drift/mất object, ghi failure và giữ RECOVERY_REQUIRED, không implicit recreate. Provider-state IO lỗi/không rõ apply outcome cũng giữ scope recovery. Resource đã tạo thành công giữ nguyên qua lỗi ở phase sau; không rollback database.

## 6. Phase, completion và failure

Chỉ ba step nội bộ: INFRASTRUCTURE_READY, CONFIGURATION_RESOLVED, MANIFEST_GENERATED. Job phase thêm PUBLISH/COMPLETE để audit, không tạo step thứ tư.

- Finish infrastructure và start configuration trong cùng transaction. Collect Resource Output từ provider bằng durable state/reference và resolve Workload Output nằm **bên trong** CONFIGURATION_RESOLVED.
- Finish configuration/start manifest cùng transaction. Generate resolved spec, score render, target adapt, config/secret reference materialization thuộc MANIFEST_GENERATED. Manifest phải giữ image digest, names và deployment-id annotation mong đợi.
- Finish manifest chuyển phase PUBLISH. Publish artifact và Argo CD thuộc phase này, không có internal step mới.
- Success: một DB transaction lưu delivery acknowledgment + artifact URI/digest, lifecycle/record SUBMITTED, job SUCCEEDED/COMPLETE và guard IDLE. Không gọi save record và complete job thành hai commit độc lập.
- Terminal failure: một transaction set job/deployment/record FAILED, failure_code/error đã redact, active internal step FAILED, các step chưa chạy SKIPPED. Publish failure chỉ dùng record delivery/error; ba step đã thành công không bị đổi FAILED.
- Lỗi external chắc chắn chưa tạo side effect (validation/render): release guard khi resource đã checkpoint READY. Terraform/CD timeout hoặc mất acknowledgment: delivery UNKNOWN nếu chưa biết; guard RECOVERY_REQUIRED. Không gán CD FAILED cho tình huống chỉ mất response.

## 7. Argo CD publish và query

Go CD Adapter đóng gói deterministic OCI artifact từ manifest đã hoàn chỉnh, push registry, nhận digest. **Trước** create/update Argo Application, persist publication intent trên record: artifact URI/digest + expected Application name; job phase PUBLISH. Nếu worker chết trong artifact upload, có thể có artifact orphan nhưng chưa gọi Argo, không có workload side effect.

Application name ổn định theo deployment scope. Adapter dùng Kubernetes API upsert Application trong namespace argocd với resourceVersion/CAS, source OCI repoURL/path, targetRevision **digest**, destination đúng namespace/cluster và ownership labels. Nếu Application hiện có không thuộc IDP scope, fail `DELIVERY_OWNERSHIP_CONFLICT`. Automated sync được bật cho workload objects do CD sở hữu; không bật tự động retry operation trong cấu hình MVP.

Acknowledgment chỉ nghĩa API đã lưu Application source mong đợi, không nghĩa Synced/Healthy. Delivery reference là Application identity (namespace/name/UID); record còn có desired artifact digest để phân biệt các lần deploy chung một Application.

UC-04 luôn read-only, chỉ đọc snapshot/images/record/steps và provider khi prerequisite tồn tại:

| Quan sát | View result |
|---|---|
| Chưa có delivery reference | PENDING nếu còn chạy; NOT_AVAILABLE nếu terminal |
| Application UID/source đã đổi sang deployment khác | NOT_AVAILABLE, reason SUPERSEDED/REPLACED; không gán health hiện tại cho deployment cũ |
| Source đúng digest nhưng Argo chưa sync revision đó | CD_SYNCED PENDING; APPLICATION_READY PENDING |
| Sync error được quan sát cho đúng revision | CD_SYNCED FAILED; giữ platform lifecycle SUBMITTED nếu publish đã được accept |
| Sync Synced, observed revision đúng digest | CD_SYNCED SUCCEEDED; tiếp tục xác minh workloads |
| Provider unavailable/unknown | PENDING với reason PROVIDER_UNAVAILABLE; không tự ghi failure vào DB |

APPLICATION_READY chỉ SUCCEEDED khi đúng Application UID/source/revision; Argo health Healthy; tất cả expected Kubernetes Deployments thuộc đúng namespace/name, template annotation deployment-id và image digests; observedGeneration >= metadata.generation; updated/ready/available replicas đạt desired và không còn replicas từ revision cũ. Missing/rolling workload là PENDING; terminal unhealthy của đúng revision là FAILED. HTTP endpoint lấy từ expected Service, không từ workload hiện tại của deployment khác. Không so digest đa kiến trúc với Pod imageID tùy tiện: expected Deployment container image phải được pin digest, kết hợp rollout/generation/template identity.

## 8. Recovery thủ công cho execution bị gián đoạn

1. Supervisor/operator chứng minh worker/process group cũ và Terraform operation đã dừng; không force-unlock state khi chưa chắc.
2. Worker mới giữ host lock, tìm job CLAIMED cũ và atomically đánh dấu FAILED/EXECUTION_INTERRUPTED, ghi step/error phù hợp, guard RECOVERY_REQUIRED. Không chạy provider lại.
3. Operator gọi internal recovery tool dưới cùng exclusive lock: đọc snapshot, record associations, deterministic state path, Terraform state và Kubernetes objects. Nếu apply tạo object nhưng state chưa ghi, import đúng identity bằng runbook; không xóa state rồi apply mới.
4. Resource tồn tại hợp lệ: reconcile metadata sang READY, increment instance version, giữ binding. Nếu chứng minh chắc chưa từng có object: giữ reserved identity/binding, set PLANNED; lần deployment mới được dùng action CREATE trên **cùng** identity sau recovery approval. Để phân biệt với uninspected PLANNED, lưu `recovery_verified` trên instance; reset false trước provider call.
5. Với phase PUBLISH, inspect expected Argo Application và artifact digest đã persist. Ghi observed reference/status vào failed record nếu xác nhận acknowledgment bị mất; không tự đổi failed deployment thành success. Nếu có trạng thái không xác định, giữ guard RECOVERY_REQUIRED.
6. Chỉ khi state/provider/CD đã xác định và không còn writer cũ: transaction audit `deployment_recovery` (operator, evidence refs, timestamp), cập nhật resource metadata và release guard IDLE. Old job/deployment giữ FAILED; người dùng tạo deployment mới với snapshot/fingerprint mới.
7. Không có cleanup database tự động trong recovery. Teardown cuối demo dùng exact owned IDs/state và chỉ xóa sau phép thử redeploy giữ dữ liệu.

## 9. API surface và lỗi tối thiểu

| Operation | HTTP / caller | Kết quả |
|---|---|---|
| createDeployment | POST /deployments | 201 deploymentId, inputFingerprint, plan, planFingerprint, algorithm |
| confirmDeployment | POST /deployments/{id}/confirm | Body expectedPlanFingerprint, overrides {}; Idempotency-Key header; 202 trackingId |
| getDeploymentDetail | GET /deployments/{id} | Lifecycle, internal steps, expected/observed delivery, images, readiness và reason |
| listDeployments | GET /applications/{id}/deployments | History từ DB, không gọi provider cho mỗi row |
| getDeploymentFailureDetail | GET /deployments/{id}/failure | Sanitized failed phase/error và recovery requirement |
| executeDeploymentJob | Internal worker | Claim một lần và durable completion |
| recoverDeployment | Internal operator tool | Audit/release hoặc vẫn blocked; không public Developer API |

400 malformed request; 401/403 authentication/scope; 404 unknown identity; 409 PLAN_CHANGED/ALREADY_ACCEPTED/IDEMPOTENCY_KEY_REUSED/SCOPE_BUSY/RECOVERY_REQUIRED; 422 invalid binding/secret/unsupported parameter; 503 provider preflight unavailable. Error response không chứa plaintext/provider environment dump. Lỗi sau 202 được đọc qua query, không đổi response confirm cũ.

## 10. Giới hạn đã chọn và tài liệu nguồn

Nghiệm thu mốc đầu: prepare/confirm/worker thật → resolver chọn `postgres-aurora` → Terraform provision Aurora private → materialize credential reference → render/publish → Argo CD sync frontend/backend tới AWS → CRUD thành công. Lưu account/region/cluster/Aurora ARN/revision và kết quả smoke test. **Xóa ngay tài nguyên AWS của task sau kiểm thử, không giữ demo chạy chờ bàn giao.** Cleanup theo dependency order: workload/CD, Aurora, target/network/registry; giữ state cho tới khi đối chiếu inventory và AWS API xác minh cleanup. Nếu thất bại/phải dừng, lưu chẩn đoán và xác minh writer đã dừng trước teardown an toàn. Không đụng tài nguyên tồn tại trước task; báo rõ tài nguyên còn sót nếu cleanup bị chặn. Cloud bị chặn là chưa hoàn thành, không thay bằng kết quả kind.

R3 staging và R9 demo không configuration được DEFERRED_MVP; không claim đã giải quyết use case tổng quát. R1/R2/R4–R8/R10 chỉ đóng ở mức thiết kế khi artifacts đồng bộ và các kịch bản ở review acceptance đi qua được; runtime acceptance cần code/test sau merge.

- [Argo CD OCI source và digest](https://argo-cd.readthedocs.io/en/stable/user-guide/oci/).
- [Kubernetes immutable Secret](https://kubernetes.io/docs/concepts/configuration/secret/#secret-immutable).
- [Kubernetes Deployment rollout](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/).
