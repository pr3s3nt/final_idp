# Phạm vi MVP

Ngày chốt: 12/09/2026. Trạng thái: phạm vi đã chốt cho mốc đầu tiên. Thiết kế chi tiết ở [MVP_DEPLOYMENT_DESIGN.md](MVP_DEPLOYMENT_DESIGN.md); 8 findings đã xử lý ở mức thiết kế MVP, R3/R9 hoãn theo phạm vi. Runtime acceptance chờ implementation.

## Mục tiêu nghiệm thu

IDP deploy một ứng dụng CRUD gồm frontend, backend và PostgreSQL. Người dùng mở frontend, tạo một bản ghi qua backend và đọc lại từ database. Redeploy backend với image mới vẫn giữ dữ liệu database.

Ưu tiên làm UC-03 (deploy) cùng phần tối thiểu của UC-04 (trạng thái/lỗi). Application Definition và Environment Configuration được chuẩn bị bằng fixture có kiểm tra hợp lệ; UI chỉnh sửa UC-01/02 làm sau.

## Các quyết định đã có

| Hạng mục | Phạm vi |
|---|---|
| Stack IDP | Go: HTTP API, orchestration, worker và các adapter |
| Ứng dụng mẫu | CRUD ghi chú; frontend và backend là hai workload, image riêng |
| Frontend | Go phục vụ HTML/CSS/JavaScript tối thiểu; reverse proxy `/api` tới backend |
| Backend | Go REST API kết nối PostgreSQL |
| Truy cập | Một địa chỉ local: `/` là frontend, `/api` là backend; có thể dùng port-forward cho demo, chưa cần mua domain/TLS |
| Workload Output | Backend cung cấp Service URL plan-time cho proxy của frontend; trình duyệt chỉ dùng `/api` cùng origin |
| Environment | Một environment `dev` |
| Target | Một cụm kind riêng tên `idp-mvp`, context `kind-idp-mvp`, được tạo ở bước dựng môi trường |
| Resource | Một logical PostgreSQL resource cho ứng dụng, StatefulSet + persistent volume + Service nội bộ |
| Infrastructure overrides | Parameter database cố định; overrides rỗng, chỉ CREATE/REUSE, chưa UPDATE/resize/replace |
| Provisioner | Go adapter gọi Terraform với Kubernetes provider; state bền vững và khóa theo resource scope, không chạy apply trên state tạm bị mất khi worker dừng |
| CD | Argo CD; Go adapter publish desired manifests thành OCI artifact, tạo/cập nhật Argo CD Application với `targetRevision` pin theo digest và đồng bộ frontend/backend |
| Renderer | `score-k8s` cho base manifests, sau đó target adapter và materialization |
| Registry | Dùng registry local với repository prefix `idp-mvp/`; cấu hình đường truy cập từ host và kind trước khi chạy |
| Secret | Reference tới Kubernetes Secret chuẩn bị sẵn trong namespace demo; DB và backend dùng cùng reference. Fixture, API response, manifest artifact và Terraform config chỉ mang tên/key reference, không mang credential |
| Persistence IDP | PostgreSQL metadata riêng, được bootstrap độc lập với database của ứng dụng; dữ liệu IDP và job không mất khi restart API/worker |
| Worker | Một worker dưới exclusive host lock; confirm enqueue bền vững, có idempotency; execution gián đoạn cần operator verification, không replay job cũ |
| Quyền truy cập MVP | Một developer, API chỉ bind loopback và dùng token local; chưa triển khai SSO/multi-tenant |

Database ứng dụng do provisioner quản lý; frontend/backend do CD quản lý. Bootstrap quản lý metadata database, namespace và secret chuẩn bị sẵn. Một Kubernetes object chỉ có một bên sở hữu vòng đời.

## Phạm vi AWS và hạ tầng hiện có

Người dùng cho phép tạo hạ tầng AWS phục vụ công việc và xóa các tài nguyên đó sau khi hoàn tất. Mốc MVP local không bắt buộc dùng AWS; chưa tạo tài nguyên AWS trong bước chốt phạm vi.

Nếu bổ sung thử nghiệm AWS, phải ghi lại resource ID, region, ownership và Terraform state của tài nguyên vừa tạo để cleanup đúng tập đó. Không xóa các cluster/resource tồn tại trước task. AWS deployment là mốc mở rộng, không phải điều kiện đạt bản MVP local này.

Kiểm tra môi trường ngày 12/09/2026:

- Có ba cụm kind `prod`, `staging`, `v2`; đã kiểm tra `kind-v2` có node Ready và có Fleet/Traefik đang chạy. MVP dùng cluster riêng để giữ ownership rõ ràng.
- Context Kubernetes mặc định trỏ tới EKS; mọi thao tác MVP phải chỉ định context/cluster đích rõ ràng.
- AWS STS xác thực thành công; điều này chưa chứng minh quyền tạo mọi loại tài nguyên AWS. Region CLI mặc định là `us-east-1`.
- Có Docker, registry local ở host port `5001`, Terraform `1.9.8`, Flux CLI `2.8.8`, Helm và `score-k8s 0.15.0`.
- Chưa tìm thấy Go SDK và Argo CD CLI trong PATH. Khi dựng môi trường, cài SDK và Argo CD trên cluster MVP; pin phiên bản Go, provider, Argo CD có hỗ trợ OCI source và image. Flux CLI có sẵn chỉ là thông tin môi trường; CD được chọn cho MVP là Argo CD.

## Luồng thực hiện

1. Bootstrap môi trường MVP, metadata database và secret; nạp definition/configuration mẫu gồm hai workload và một resource.
2. Chọn hai image version và target cố định, tạo infrastructure plan để review.
3. Confirm gửi fingerprint đã review và idempotency key; API atomically enqueue và trả tracking ID.
4. Worker dùng input revision đã accept, provision/reuse PostgreSQL, lấy resource outputs và tạo Workload Output plan-time.
5. Resolve configuration/reference, render manifests, publish OCI artifact và cập nhật Argo CD Application tới đúng digest qua Go CD adapter.
6. Argo CD đồng bộ frontend/backend; API query trả lifecycle, các step, delivery revision và readiness tương ứng phiên bản mong đợi. Trạng thái Sync/Health của Argo CD được ánh xạ vào delivery/view status, tách biệt với lifecycle của IDP.
7. Demo CRUD, redeploy backend, kiểm tra dữ liệu còn nguyên và kết thúc thử nghiệm theo runbook cleanup.

## Những bảo đảm đã đặc tả, cần kiểm chứng khi code

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
| Deploy lần đầu | Plan CREATE database; worker provision thành công, Argo CD đồng bộ đúng artifact digest và hai image, CRUD hoạt động |
| Redeploy backend | Database được reuse, không mất dữ liệu; readiness khớp phiên bản backend mới |
| Confirm lặp | Cùng request đã accept trả cùng tracking ID, có một job và không provision trùng |
| Plan đã thay đổi | Confirm với fingerprint cũ bị từ chối và trả plan mới để review |
| Input sửa sau enqueue | Deployment tiếp tục dùng đúng input đã accept |
| Provider hoặc renderer lỗi | API hiển thị failed phase/error đã loại secret, không báo SUBMITTED/Ready giả |
| Worker dừng sau provision | Giữ resource identity/state, nhận biết execution gián đoạn, thực hiện được runbook recovery mà không tạo database trùng |
| Backend mới chưa Ready | Health của backend cũ không làm deployment mới được báo Ready |
| Kết thúc thử nghiệm | Có lệnh/runbook cleanup đúng tài nguyên MVP; giữ database suốt phép thử redeploy, chỉ xóa dữ liệu demo ở teardown cuối |

## Chưa làm trong mốc này

- UI đầy đủ cho UC-01/02, catalog editor và quản trị platform.
- Nhiều environment/target, multi-tenant, SSO, shared resource giữa application.
- Automatic retry/resume toàn bộ pipeline, rollback tự động, HA, autoscaling.
- Nhập secret trực tiếp, staging/promote, rotation secret và credential động từ provider.
- App không có configuration requirement chưa thuộc demo chính; R9 vẫn mở để giải quyết khi mở rộng phạm vi.
- Database HA/backup phục vụ production, cloud networking và AWS deployment.
- Logs/metrics/traces dashboard nâng cao; vẫn cần log vận hành tối thiểu đã loại credential.

## Tài liệu liên quan

- [Các vấn đề còn mở và điều kiện đóng](06_traceability/review_open_issues.md).
- [Traceability và trạng thái review](06_traceability/traceability_matrix.md).
- [Cài đặt Argo CD](https://argo-cd.readthedocs.io/en/stable/getting_started/).
- [Argo CD OCI sources](https://argo-cd.readthedocs.io/en/stable/user-guide/oci/).
- [Terraform Kubernetes StatefulSet](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/stateful_set_v1).
- [Go HTTP reverse proxy](https://go.dev/pkg/net/http/httputil/).
