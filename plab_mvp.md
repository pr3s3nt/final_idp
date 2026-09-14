# Prompt triển khai UC-03 — MVP luồng chính trên AWS

**Bắt buộc: deploy frontend + backend lên target Kubernetes AWS và provision Aurora PostgreSQL trên AWS, kiểm thử CRUD, lưu bằng chứng rồi xóa ngay toàn bộ tài nguyên AWS do task tạo để tránh tiếp tục phát sinh phí. Không bàn giao một demo chỉ chạy trên local/kind; không giữ cloud chạy chờ tôi xem.**

Bạn là AI coding agent làm việc trực tiếp trong repository này. Hãy triển khai và chạy kiểm chứng **UC-03: Deploy Application**, chỉ ở mức MVP có luồng chính chạy end-to-end. Tôi cần code chạy thật, không chỉ kế hoạch, scaffold hoặc mock các tích hợp chính.

## 1. Mục tiêu và thứ tự ưu tiên

Một developer deploy một ứng dụng mẫu gồm **frontend Go + backend Go trên target Kubernetes AWS và Aurora PostgreSQL trên AWS thật**. Cả hai workload và database ứng dụng phải chạy trên hạ tầng AWS do task quản lý. Sau deployment, mở frontend, tạo một ghi chú qua backend và đọc lại dữ liệu đã lưu trong Aurora PostgreSQL.

**Deploy thành công trên kind/local không đạt nghiệm thu.** IDP API/worker, metadata database và công cụ phát triển được chạy local; kind chỉ hỗ trợ phát triển/test. Không thay target AWS bằng kind khi gặp lỗi cloud: báo blocker và phần chưa hoàn thành.

Đây là mốc **UC3 happy-path**, hẹp hơn toàn bộ MVP đã mô tả trong repository. Không cần hoàn thành mọi use case, bảng dữ liệu hay kịch bản lỗi trước khi bàn giao mốc này. Không tuyên bố đã hoàn thành toàn bộ MVP hoặc đã kiểm chứng mọi finding cũ.

Đọc `AGENTS.md` nếu có, sau đó đọc:

- `MVP_SCOPE.md`
- `MVP_DEPLOYMENT_DESIGN.md`
- `04_operation_contracts/operation_contracts.md`
- `03_database_erd/schema.md`
- `sequence_digrams/uc_03_deploy_application.puml`
- `06_traceability/mvp_design_review.md`

Giữ các quyết định kiến trúc liên quan đến luồng chính. Dùng phạm vi trong prompt này để chọn phần cần triển khai ngay; ghi rõ phần hoãn vào tài liệu implementation, không sửa tài liệu nền thành tuyên bố mọi thứ đã hoàn thành. Có thể chỉ tạo migrations cho các bảng thực sự cần, nhưng giữ đúng ý nghĩa dữ liệu và các ràng buộc áp dụng cho phần đã triển khai.

## 2. Stack và môi trường đã chốt

- IDP API/worker và app mẫu viết bằng Go. Frontend có thể là Go phục vụ HTML/JavaScript tối giản và proxy `/api` tới backend; không cần framework frontend.
- Một application, một environment `dev`, một target Kubernetes trên AWS, namespace ứng dụng `idp-demo-dev`. Lưu rõ AWS account, region, cluster identity và context, không dùng tên kind làm target nghiệm thu.
- Fixture chuẩn bị sẵn Application Definition, Environment Configuration và catalog cần thiết. Không làm UI/API editor UC-01/02.
- Terraform tạo target Kubernetes, private network và prerequisite ở bước bootstrap, với state riêng. Trong UC3, Resource Definition Resolver ánh xạ logical requirement `postgres` theo deployment context: target AWS chọn module Aurora PostgreSQL qua AWS provider; target kind/local chọn module PostgreSQL StatefulSet/PVC/Service qua Kubernetes provider. Không bootstrap sẵn database ứng dụng rồi bỏ qua provisioner. State của từng resource instance tách state bootstrap và được lưu bền vững.
- Render base manifests bằng `score-k8s`, resolve/materialize cấu hình cho hai workload.
- **Argo CD** triển khai frontend/backend từ manifest OCI artifact được pin digest. Không thay bằng Flux hoặc dùng `kubectl apply` workload để bỏ qua CD adapter.
- Dùng registry mà host, AWS worker nodes và Argo CD repo-server truy cập/xác thực được; không mặc định registry localhost:5001 dùng được từ AWS. Ghi lại lựa chọn registry cùng cấu hình pull image và pull OCI manifests.
- Metadata của IDP dùng PostgreSQL riêng, không dùng chung database ứng dụng do provisioner tạo.
- Bootstrap quản lý namespace, metadata database, Argo CD và prerequisite đồng bộ secret. UC3 Terraform module quản lý Aurora; RDS quản lý master credential trong AWS Secrets Manager; secret materializer chỉ truyền reference tới workload, không đọc/lưu plaintext trong IDP. Argo CD quản lý frontend/backend và object materialization thuộc release. Không để hai bên cùng sở hữu một object.
- Kiểm tra tool/version hiện có trước khi cài thêm; pin các phiên bản tương thích, đặc biệt khả năng OCI source của Argo CD. Tra tài liệu chính thức khi cần xác minh cú pháp/khả năng hỗ trợ.

### Chốt cấu hình cloud trước khi provision

AWS là bắt buộc và cấu hình nghiệm thu đã chốt: EKS 1.35 với một managed node `t3.small` On-Demand, Aurora PostgreSQL Serverless v2 0.5–1 ACU, ECR, Argo CD local/kind; node dùng public subnet nhưng EKS bật private endpoint, API public giới hạn IP runner, Aurora ở hai DB subnet private. Không tạo NAT Gateway hoặc load balancer. Giá nền đã kiểm tra tại `ap-southeast-1` khoảng 0.2314 USD/giờ ở 0.5 ACU, chưa gồm storage/I/O/data transfer/tax. State bootstrap và state Aurora tách riêng; vẫn phải xác minh identity/quyền trước apply và cleanup ngay sau test. Không mở rộng Aurora thành kiến trúc production ngoài nhu cầu demo.

Argo CD phải được cấu hình destination tới đúng cluster AWS; nếu chạy Argo CD local/kind thì phải đăng ký target AWS, không dùng in-cluster destination trỏ nhầm về kind. Aurora không public ra internet; security group chỉ cho phép kết nối từ workload target qua private network. Đường truy cập frontend demo có thể là tunnel/port-forward tới cluster AWS, không bắt buộc domain/public load balancer.

## 3. Luồng bắt buộc phải chạy thật

1. Bootstrap control plane IDP, target AWS, private connectivity và secret-sync prerequisite theo cấu hình đã chọn; nạp fixture AWS có cả definition Aurora và kind/local, build rồi push hai image mẫu tới registry target đọc được; pin image digest cho deployment.
2. `POST /deployments`: đọc fixture/source input, lưu snapshot đầu vào, tạo deployment và trả infrastructure plan CREATE PostgreSQL cùng `planFingerprint`. Bước này chưa provision.
3. `POST /deployments/{id}/confirm`: nhận `expectedPlanFingerprint`, `overrides: {}` và `Idempotency-Key`; kiểm tra plan, lưu job bền vững cùng trạng thái accept trong một transaction, trả tracking ID. Confirm lặp cùng request không tạo thêm job.
4. Một worker lấy job, dùng snapshot đã accept, lưu resource identity/state reference trước khi gọi Terraform; resolver phải chọn `postgres-aurora` cho AWS rồi provision Aurora thật. Với test kind/local, cùng logical graph phải chọn `postgres-kubernetes` và tạo StatefulSet/PVC/Service trong kind.
5. Lấy Aurora endpoint/port/database/username cùng credential ARN/reference, tính backend Service URL, resolve configuration và materialize secret reference, rồi render manifests cho frontend/backend. Frontend proxy `/api` tới backend; trình duyệt không gọi Kubernetes Service DNS trực tiếp. Plaintext database credential không đi qua snapshot, API response, log hoặc OCI artifact.
6. Go CD adapter push OCI manifest artifact, lưu publication intent/digest rồi tạo hoặc cập nhật Argo CD Application trỏ tới đúng artifact digest và target.
7. Argo CD sync hai workload. Có `GET /deployments/{id}` tối thiểu để xem lifecycle, ba step `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, delivery và lỗi ngắn gọn.
8. Chờ đúng artifact/image revision được sync và workload Ready trên AWS; expose frontend qua endpoint hoặc tunnel/port-forward tới target AWS. Chạy smoke test tạo và đọc ghi chú qua frontend → backend → PostgreSQL; ghi bằng chứng account/region/cluster và workload/database thực sự ở AWS. Địa chỉ trình duyệt localhost qua tunnel không có nghĩa app chạy local.

Phần query ở bước 7–8 chỉ là hỗ trợ nghiệm thu UC3, **không phải triển khai đầy đủ UC-04**. Có thể dùng CLI/curl để gọi IDP, không cần giao diện portal.

## 4. Giới hạn và bảo vệ tối thiểu

- API IDP chỉ bind loopback, dùng token local. Một worker, một execution tại một thời điểm; không tự replay job đang chạy khi restart.
- Chỉ nhận fixture/target/parameters thuộc scope. Chưa hỗ trợ UPDATE, resize, replace hoặc shared database; yêu cầu ngoài scope phải bị từ chối rõ ràng.
- Aurora bật RDS-managed master password trong AWS Secrets Manager. Chỉ truyền ARN/reference và destination name/key qua orchestration/materialization; không ghi credential vào Git, metadata, Terraform input/state, API response, artifact hoặc logs. Với kind/local, dùng Kubernetes Secret reference đã được kiểm tra target/namespace/UID/key.
- Mọi thao tác Kubernetes/Terraform phải chỉ rõ target AWS đã allowlist, account và region. Kiểm tra identity/ownership trước khi tạo hoặc sửa; tên trùng không chứng minh tài nguyên thuộc task.
- Không dùng kubeconfig current-context ngầm định, không thay đổi cluster/resource có sẵn ngoài phạm vi. Người dùng cho phép tạo hạ tầng AWS phục vụ demo rồi xóa sau khi xong; ghi inventory resource ID/ARN, ownership và state ngay từ đầu. Không in credential hoặc mở rộng IAM ngoài nhu cầu demo.
- Provision/render/publish thất bại phải dừng, ghi phase/error đã loại secret; không báo thành công giả. Argo Application đã được ghi nhận không đồng nghĩa ứng dụng đã Ready.
- Nếu execution bị gián đoạn hoặc provider outcome không rõ, đánh dấu cần kiểm tra thủ công và chặn chạy lại mù quáng. Không cần xây recovery tool đầy đủ ở mốc này; không xóa state hoặc database để che lỗi.

## 5. Chưa làm ở mốc này

- UC-01/02 editor, portal UI, history/dashboard UC-04 đầy đủ.
- Redeploy/REUSE như một tiêu chí nghiệm thu, rollback, automatic retry/resume, recovery workflow đầy đủ và fault-injection matrix.
- Multi-tenant, SSO, HA, nhiều môi trường/target, cloud production hardening. AWS deployment happy path vẫn là yêu cầu bắt buộc.
- Secret rotation workflow, Aurora backup/failover/replica production hardening và observability nâng cao.
- Toàn bộ 24 bảng và mọi abstraction trong sơ đồ nếu luồng chính chưa cần đến.

Đặc biệt: không mở rộng sang các mục này chỉ để làm kiến trúc hoàn chỉnh. Nếu cần một phần nhỏ làm dependency cho happy path thì triển khai đúng phần nhỏ đó và giải thích ngắn gọn.

## 6. Cách thực hiện

1. Kiểm tra Git/worktree và môi trường. Bảo toàn thay đổi của người dùng. Tạo nhánh `feat/uc3-mvp` từ `main` nếu đang ở trạng thái phù hợp; nếu nhánh đã có, kiểm tra và tiếp tục, không reset/ghi đè. Không tự commit, merge hoặc push.
2. Nêu kế hoạch ngắn rồi bắt đầu code ngay. Tự chọn chi tiết implementation nhỏ trong phạm vi; chỉ hỏi khi thiếu quyền, credential hoặc quyết định làm thay đổi đáng kể phạm vi.
3. Làm theo từng lát cắt chạy được: app mẫu → bootstrap AWS/private connectivity → prepare/confirm/job → resolver chọn đúng definition → Terraform provision Aurora → secret materialization/configuration/render → Argo CD deploy lên AWS → smoke test. Dùng kind để kiểm chứng nhánh `postgres-kubernetes` trước khi chạy cloud, nhưng không dừng ở đó. Script bootstrap hỗ trợ demo nhưng không được thay thế luồng orchestration Go cần nghiệm thu.
4. Tạo lệnh/scripts rõ ràng cho bootstrap, build/push, migrate/seed, chạy IDP, deploy demo, smoke test và cleanup; ghi prerequisite và thứ tự chạy trong README implementation.
5. Chạy `go test ./...` cho từng Go module đã tạo, kiểm tra build và Terraform validate, sau đó chạy integration/smoke test trên AWS thật. Test tối thiểu phần fingerprint, confirm idempotency và lỗi không báo success giả. Không dùng mock/kind làm bằng chứng end-to-end cloud.
6. Lưu bằng chứng smoke test và hướng dẫn tái tạo trước teardown. **Ngay sau khi kiểm thử xong, chủ động xóa tài nguyên AWS do task tạo, không chờ bàn giao/xem live hoặc xin xác nhận lại cho tập tài nguyên đã được phép xóa.** Dữ liệu database demo cũng được xóa ở teardown này. Gỡ workload qua Argo CD, destroy database khi target còn truy cập được, rồi mới hạ target/network/registry thuộc task; kiểm tra tài nguyên còn sót theo inventory và AWS API, không chỉ dựa vào Terraform exit code. Bao gồm volume, snapshot, load balancer, NAT gateway, public IP, registry/artifact hoặc log storage nếu task có tạo. Không xóa tài nguyên có sẵn hoặc không chứng minh được ownership; không xóa state trước khi xác minh cleanup. Nếu demo thất bại hoặc phải dừng sau khi đã tạo tài nguyên, lưu chẩn đoán, xác minh không còn worker/provider đang ghi rồi thực hiện teardown an toàn cho tập task-owned đã xác định; cleanup không biến test thất bại thành pass. Nếu cleanup bị chặn, báo ngay resource ID/region, lỗi và lệnh cần chạy tiếp; không tuyên bố đã hết phí hay đã xóa hết khi chưa xác minh.
7. Nếu bị chặn, báo chính xác bước, bằng chứng lỗi và thông tin cần bổ sung. Phân biệt rõ đã code, đã test và chưa test; không coi scaffold là hoàn thành.

## 7. Điều kiện hoàn thành và bàn giao

Chỉ kết luận **UC3 happy-path MVP hoàn thành** khi có bằng chứng:

- Gọi prepare và confirm qua API Go tạo được deployment/job thật.
- Terraform tạo hạ tầng AWS; worker dùng definition đã resolve để gọi Terraform tạo Aurora PostgreSQL private, resolve endpoint/credential reference và render manifests thật.
- Go adapter publish artifact và Argo CD sync đúng revision tới AWS; cả frontend/backend Ready trên AWS.
- Smoke test đi qua frontend, tạo và đọc lại ghi chú từ Aurora PostgreSQL thành công; có bằng chứng workload target không phải kind/local và database resource là Aurora.
- Có hướng dẫn để tôi tái tạo demo AWS, xem trạng thái/logs và cleanup; có inventory và kết quả kiểm tra cleanup thực tế. Nếu cleanup bị chặn phải báo tài nguyên còn lại, không tuyên bố đã xóa hết.

Khi bàn giao, trả lời ngắn gọn: đã làm gì, kết quả test thực tế, bằng chứng demo AWS, lệnh tái tạo/cách mở frontend sau khi tái tạo, tài nguyên đã xóa hoặc còn sót và những phần chủ động hoãn. Nêu rõ demo đã teardown, không đưa endpoint đã bị xóa như thể còn hoạt động. Không tiếp tục mở rộng sau khi đạt mốc này.
