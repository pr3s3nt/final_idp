# Phạm vi MVP

Ngày cập nhật: 12/09/2026. Phạm vi hiện tại theo yêu cầu người dùng: **UC-03 happy path trên AWS thật**, thay thế mục tiêu local-only trước đây. Thiết kế chi tiết ở [MVP_DEPLOYMENT_DESIGN.md](MVP_DEPLOYMENT_DESIGN.md), prompt triển khai ở [plab_mvp.md](plab_mvp.md). Tám findings đã xử lý ở mức thiết kế rộng, R3/R9 hoãn; không phải bằng chứng runtime hoặc cloud đã chạy.

## Mục tiêu nghiệm thu

IDP deploy một ứng dụng CRUD gồm frontend, backend và PostgreSQL **đều chạy trên AWS**. Người dùng mở frontend, tạo một bản ghi qua backend và đọc lại từ database. Deploy lần đầu và CRUD thành công là mục tiêu mốc này; redeploy giữ dữ liệu và recovery đầy đủ là mốc sau.

Chỉ làm UC-03 (deploy) cùng phần tối thiểu của UC-04 (trạng thái/lỗi). Application Definition và Environment Configuration được chuẩn bị bằng fixture có kiểm tra hợp lệ; UI chỉnh sửa UC-01/02 làm sau. API/worker/metadata IDP có thể chạy local, kind có thể dùng test hỗ trợ; **chạy trên kind không thay thế nghiệm thu AWS**.

## Các quyết định đã có

| Hạng mục | Phạm vi |
|---|---|
| Stack IDP | Go: HTTP API, orchestration, worker và các adapter |
| Ứng dụng mẫu | CRUD ghi chú; frontend và backend là hai workload, image riêng |
| Frontend | Go phục vụ HTML/CSS/JavaScript tối thiểu; reverse proxy `/api` tới backend |
| Backend | Go REST API kết nối PostgreSQL |
| Truy cập | `/` là frontend, `/api` là backend trên AWS; endpoint hoặc tunnel/port-forward tới target AWS, chưa cần mua domain/public load balancer |
| Workload Output | Backend cung cấp Service URL plan-time cho proxy của frontend; trình duyệt chỉ dùng `/api` cùng origin |
| Environment | Một environment `dev` |
| Target | Một cụm Kubernetes trên AWS do task tạo/quản lý; ghi account, region, cluster identity/context thực tế. EKS hay Kubernetes trên EC2 cần chốt trước provision |
| Resource | Một logical PostgreSQL resource cho ứng dụng, StatefulSet + PVC + Service nội bộ trên target AWS; cấu hình storage hoạt động thật. Chưa chuyển sang RDS |
| Infrastructure overrides | Parameter database cố định; overrides rỗng, chỉ CREATE/REUSE, chưa UPDATE/resize/replace |
| Provisioner | Terraform bootstrap hạ tầng target AWS bằng state riêng; trong UC3, Go adapter gọi Terraform Kubernetes provider để tạo database trên target AWS. State database bền vững và khóa theo resource scope, không dùng state tạm |
| CD | Argo CD; Go adapter publish desired manifests thành OCI artifact, tạo/cập nhật Argo CD Application với `targetRevision` pin theo digest và đồng bộ frontend/backend |
| Renderer | `score-k8s` cho base manifests, sau đó target adapter và materialization |
| Registry | Registry hỗ trợ images/OCI artifacts mà host, AWS nodes và Argo CD repo-server truy cập/xác thực được; không giả định registry localhost dùng được từ cloud |
| Secret | Reference tới Kubernetes Secret chuẩn bị sẵn trong namespace demo; DB và backend dùng cùng reference. Fixture, API response, manifest artifact và Terraform config chỉ mang tên/key reference, không mang credential |
| Persistence IDP | PostgreSQL metadata riêng, được bootstrap độc lập với database của ứng dụng; dữ liệu IDP và job không mất khi restart API/worker |
| Worker | Một worker dưới exclusive host lock; confirm enqueue bền vững, có idempotency; execution gián đoạn cần operator verification, không replay job cũ |
| Quyền truy cập MVP | Một developer, API chỉ bind loopback và dùng token local; chưa triển khai SSO/multi-tenant |

Database ứng dụng trên AWS do provisioner quản lý; frontend/backend trên AWS do CD quản lý. Bootstrap quản lý target AWS, metadata database, namespace và secret chuẩn bị sẵn, không tạo sẵn database ứng dụng để bỏ qua UC3. Một Kubernetes object chỉ có một bên sở hữu vòng đời. Argo CD có thể đặt local hoặc trên AWS nhưng destination phải trỏ đúng target AWS đã đăng ký, không ngầm dùng in-cluster kind.

## Phạm vi AWS và hạ tầng hiện có

Người dùng cho phép tạo hạ tầng AWS phục vụ công việc và xóa các tài nguyên đó sau khi hoàn tất. **AWS deployment là điều kiện bắt buộc**, không phải mốc mở rộng tùy chọn. Lần cập nhật tài liệu này không tạo tài nguyên AWS.

Trước provision, ghi lựa chọn EKS hoặc Kubernetes trên EC2, account/region, compute, network/access, storage, registry, vị trí Argo CD, state và ước tính chi phí theo thời gian chạy. Xác minh giá/quyền thực tế, hỏi người dùng nếu cần chốt dịch vụ/ngân sách; chưa có lựa chọn dịch vụ cloud cụ thể được duyệt trong tài liệu này. Database không được public ra internet. Không tự thêm RDS/production architecture.

Ghi inventory resource ID/ARN, region, ownership và Terraform state ngay khi tạo. **Ngay sau smoke test, lưu bằng chứng rồi chủ động xóa tài nguyên AWS do task tạo để tránh tiếp tục phát sinh phí; không giữ demo live chờ bàn giao và không cần xin lại quyền cleanup tập đã xác định.** Thứ tự: workload/CD trước, database khi target còn hoạt động, target/network/registry sau; kiểm tra tài nguyên còn sót bằng inventory và AWS API, gồm volume/snapshot/load balancer/NAT/public IP/storage nếu có tạo. Không xóa cluster/resource tồn tại trước task hoặc xóa state khi chưa xác minh cleanup. Nếu thất bại/phải dừng giữa chừng, lưu chẩn đoán và dừng/xác minh writer trước teardown an toàn của tập task-owned. Cleanup bị chặn phải báo resource ID/region và cách xử lý tiếp; không tuyên bố đã xóa hết/hết phí. Cloud bị chặn phải báo blocker, không fallback kind rồi tuyên bố hoàn thành.

Thông tin kiểm tra môi trường trước đây ngày 12/09/2026 (lịch sử, phải kiểm tra lại khi triển khai; không phải lựa chọn target hiện tại):

- Có ba cụm kind `prod`, `staging`, `v2`; đã kiểm tra `kind-v2` có node Ready và có Fleet/Traefik đang chạy. Không thay đổi các cluster này; không dùng chúng làm target nghiệm thu AWS.
- Context Kubernetes mặc định trỏ tới EKS; mọi thao tác MVP phải chỉ định context/cluster đích rõ ràng.
- AWS STS xác thực thành công; điều này chưa chứng minh quyền tạo mọi loại tài nguyên AWS. Region CLI mặc định là `us-east-1`.
- Có Docker, registry local ở host port `5001`, Terraform `1.9.8`, Flux CLI `2.8.8`, Helm và `score-k8s 0.15.0`.
- Chưa tìm thấy Go SDK và Argo CD CLI trong PATH. Khi dựng môi trường, cài SDK và Argo CD trên cluster MVP; pin phiên bản Go, provider, Argo CD có hỗ trợ OCI source và image. Flux CLI có sẵn chỉ là thông tin môi trường; CD được chọn cho MVP là Argo CD.

## Luồng thực hiện

1. Bootstrap target AWS theo cấu hình cloud đã chọn, metadata database và secret; nạp definition/configuration mẫu gồm hai workload và một resource với context AWS đã xác minh.
2. Chọn hai image version và target cố định, tạo infrastructure plan để review.
3. Confirm gửi fingerprint đã review và idempotency key; API atomically enqueue và trả tracking ID.
4. Worker dùng input revision đã accept, provision PostgreSQL trên target AWS, lấy resource outputs và tạo Workload Output plan-time. REUSE thuộc thiết kế mở rộng, chưa là tiêu chí mốc đầu.
5. Resolve configuration/reference, render manifests, publish OCI artifact và cập nhật Argo CD Application tới đúng digest qua Go CD adapter.
6. Argo CD đồng bộ frontend/backend lên AWS; API query trả lifecycle, các step, delivery revision và readiness tương ứng phiên bản mong đợi. Trạng thái Sync/Health của Argo CD được ánh xạ vào delivery/view status, tách biệt với lifecycle của IDP.
7. Demo CRUD trên AWS, lưu bằng chứng target/resource/revision và kết thúc thử nghiệm theo runbook cleanup. Không yêu cầu redeploy trong mốc happy-path.

## Những bảo đảm đã đặc tả, cần kiểm chứng khi code

Đây là thiết kế rộng, không phải yêu cầu triển khai toàn bộ recovery/redeploy trước khi xong happy path. Mốc đầu vẫn giữ snapshot, confirm idempotency, durable state/identity, lỗi đúng phase và không tự replay khi outcome không rõ; phần tool recovery đầy đủ và fault-injection matrix để sau.

- Confirm phải kiểm tra fingerprint do client xác nhận; request lặp không tạo job hoặc provider side effect trùng (R2).
- Definition/configuration dùng cho deployment được pin revision/snapshot của source input, không bị thay đổi bởi lần chỉnh sửa sau accept (R4).
- Không automatic retry toàn bộ pipeline. Khi worker dừng giữa chừng, restart phải nhận biết execution bị gián đoạn, giữ provider references/state và có đường kiểm tra/recovery thủ công rõ ràng; không tự tạo resource thay thế chỉ vì trạng thái chưa READY (R1, R6).
- Final lifecycle, delivery metadata và trạng thái job phải được ghi nhất quán; lỗi thuộc phase nào phải hiện ở phase đó, kể cả collect output (R7, R8).
- Chỉ báo application Ready khi observed workload tương ứng image/revision mong đợi; không lấy health của bản cũ làm kết quả bản mới (R5).
- Secret reference phải tồn tại trước enqueue; contract C1–C4 dùng permanent immutable reference, không có API nhập/stage/promote trong MVP. R3 được hoãn theo phạm vi, không coi giao thức staging đã được sửa.
- Schema, contract, VOPC và sequence đã đồng bộ theo thiết kế deployment; trạng thái từng finding và điều kiện kiểm chứng ở traceability (R10).

## Tiêu chí nghiệm thu

| Kịch bản | Kết quả cần đạt |
|---|---|
| Deploy lần đầu trên AWS | Terraform bootstrap hạ tầng AWS; plan CREATE database; worker provision database trên AWS, Argo CD đồng bộ đúng artifact digest và hai image lên AWS, CRUD hoạt động |
| Bằng chứng cloud | Ghi account/region/cluster identity, resource IDs, destination và revision; cả workloads lẫn database ở AWS. Local/kind test không đủ |
| Confirm lặp | Cùng request đã accept trả cùng tracking ID, có một job và không provision trùng |
| Plan đã thay đổi | Confirm với fingerprint cũ bị từ chối và trả plan mới để review |
| Input sửa sau enqueue | Deployment tiếp tục dùng đúng input đã accept |
| Provider hoặc renderer lỗi | API hiển thị failed phase/error đã loại secret, không báo SUBMITTED/Ready giả |
| Execution gián đoạn | Giữ resource identity/state, báo cần kiểm tra thủ công và chặn replay mù quáng; tool recovery đầy đủ chưa bắt buộc |
| Readiness | SUBMITTED không đồng nghĩa Ready; kiểm đúng target/revision/image và rollout mong đợi |
| Kết thúc thử nghiệm | Lưu bằng chứng CRUD và chạy cleanup đúng tài nguyên AWS do task tạo, kiểm tra tài nguyên còn sót; báo rõ nếu cleanup bị chặn |

## Chưa làm trong mốc này

- UI đầy đủ cho UC-01/02, catalog editor và quản trị platform.
- Nhiều environment/target, multi-tenant, SSO, shared resource giữa application.
- Automatic retry/resume toàn bộ pipeline, rollback tự động, HA, autoscaling.
- Redeploy/REUSE như tiêu chí nghiệm thu, tool recovery đầy đủ và fault-injection matrix. Các thiết kế đó vẫn giữ làm đầu vào mốc sau.
- Nhập secret trực tiếp, staging/promote, rotation secret và credential động từ provider.
- App không có configuration requirement chưa thuộc demo chính; R9 vẫn mở để giải quyết khi mở rộng phạm vi.
- Database HA/backup và cloud production hardening. Network/access/storage tối thiểu để deploy AWS chạy thật vẫn trong scope.
- Logs/metrics/traces dashboard nâng cao; vẫn cần log vận hành tối thiểu đã loại credential.

## Tài liệu liên quan

- [Các vấn đề còn mở và điều kiện đóng](06_traceability/review_open_issues.md).
- [Traceability và trạng thái review](06_traceability/traceability_matrix.md).
- [Cài đặt Argo CD](https://argo-cd.readthedocs.io/en/stable/getting_started/).
- [Argo CD OCI sources](https://argo-cd.readthedocs.io/en/stable/user-guide/oci/).
- [Terraform Kubernetes StatefulSet](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/stateful_set_v1).
- [Go HTTP reverse proxy](https://go.dev/pkg/net/http/httputil/).
