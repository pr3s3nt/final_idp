# Review findings — lịch sử và trạng thái MVP

Ngày kiểm tra: 12/09/2026.

Trạng thái sau thiết kế MVP: **8 RESOLVED_DESIGN_MVP, 2 DEFERRED_MVP (R3, R9)**. Xem [traceability](traceability_matrix.md) và [MVP design review](mvp_design_review.md). Đây là kết luận ở mức tài liệu; chưa có runtime implementation/tests. Các mục R1–R10 bên dưới giữ nguyên tình huống và link commit baseline để truy vết lịch sử, không mô tả HEAD hiện tại.

Phạm vi triển khai mới: **UC3 happy path trên AWS thật**, không local-only. Các trạng thái thiết kế trên không chứng minh AWS đã chạy, cũng không buộc implement toàn bộ recovery/redeploy ở mốc đầu. Xem [scope](../MVP_SCOPE.md) và [prompt](../plab_mvp.md); lựa chọn dịch vụ AWS/chi phí còn cần chốt trước provision.

## Kết luận của lần review baseline

Bản main trước merge còn các khoảng trống thiết kế mà commit draft nhắm tới. Bản draft đã sửa nhiều điểm đúng hướng, nhưng chưa đủ cơ sở để kết luận “đã sửa hết, sửa đúng hoàn toàn”. Sau merge, main tiếp nhận bản thiết kế này cùng các vấn đề còn mở bên dưới.

Review này ghi nhận 10 findings: 4 ưu tiên cao (P1), 6 ưu tiên vừa (P2). Đây là lỗi/mâu thuẫn hoặc khoảng trống trong đặc tả; repository không có mã triển khai để chứng minh hành vi runtime. Các kịch bản bên dưới là phép đối chiếu logic với chính contracts, sequence và schema.

## Phạm vi và bằng chứng

- main: `11ad58249f58e06108ab86da787223254be9b05f`.
- draft: `88585ccdd1a7896c615ad43fbb0d0f6b62d231ef`, đi trước main đúng một commit, thay đổi 21 file.
- Đã đối chiếu SHA local với remote bằng `git ls-remote`; chúng khớp tại thời điểm kiểm tra.
- Đối chiếu use case realization, VOPC, domain/persistence, schema/ERD, operation contracts, state machines, traceability và 4 sequence diagrams.
- Static checks xác nhận main có 43 operation, 42 class/interface, 19 table; draft có 45 operation, 48 class/interface, 21 table. Tên các operation có xuất hiện trong sequence tương ứng; danh sách table schema và ERD khớp.
- Có 13 PlantUML diagrams trên mỗi nhánh. Đã kiểm tra cấu trúc văn bản cơ bản; chưa render/kiểm tra cú pháp đầy đủ vì môi trường không có Java/PlantUML.
- `git diff --check main draft` không báo lỗi whitespace.
- Lần review ban đầu chỉ đọc, không sửa tài liệu hoặc merge. Trong bước tích hợp tiếp theo, báo cáo này được lưu vào Git và trạng thái acceptance được cập nhật; trạng thái hiện tại được cập nhật trong bảng theo dõi bên dưới.
- Commit draft nói đến “11 vấn đề trong plan.md”, nhưng `plan.md` không có trong cây file hoặc lịch sử reachable của hai nhánh. Vì vậy danh sách 11 vấn đề được đối chiếu theo commit message và bảng closure trong traceability, không phải bản yêu cầu gốc.

## Danh sách theo dõi hiện tại

| ID | Ưu tiên | Vấn đề | Trạng thái |
|---|---|---|---|
| R1 | P1 | Retry làm đổi fingerprint bởi kết quả lần chạy trước | RESOLVED_DESIGN_MVP |
| R2 | P1 | Confirm không gắn với plan người dùng đã xem | RESOLVED_DESIGN_MVP |
| R3 | P1 | Configuration được công bố trước khi secret được promote | DEFERRED_MVP |
| R4 | P1 | Worker đọc definition/configuration có thể thay đổi sau accept | RESOLVED_DESIGN_MVP |
| R5 | P2 | Readiness chưa đối chiếu đúng phiên bản workload | RESOLVED_DESIGN_MVP |
| R6 | P2 | Lookup READY không hỗ trợ recovery instance FAILED | RESOLVED_DESIGN_MVP |
| R7 | P2 | Hoàn tất job và lifecycle chưa có transaction chung rõ ràng | RESOLVED_DESIGN_MVP |
| R8 | P2 | Lỗi collect output chưa có active step xác định | RESOLVED_DESIGN_MVP |
| R9 | P2 | App không có configuration requirement chưa có đường deploy | DEFERRED_MVP |
| R10 | P2 | VOPC chưa đồng bộ với sequence | RESOLVED_DESIGN_MVP |

Chỉ đóng finding khi đáp ứng điều kiện đóng trong mục tương ứng và đồng bộ các artifact bị ảnh hưởng. Sau khi chốt phạm vi MVP, có thể quyết định hoãn một tính năng kèm lý do và ranh giới cụ thể; việc đếm coverage không tự đóng finding.

## Đối chiếu 11 vấn đề mà draft tuyên bố đã sửa

“Đã sửa” trong bảng chỉ nói về phạm vi hẹp của issue đó, không đồng nghĩa toàn bộ use case đã đúng.

| Issue cũ | main | Đánh giá draft |
|---|---|---|
| #1 Ownership của draft UC-01/02 qua nhiều request | Sequence chưa mô tả đầy đủ cách truyền/khôi phục draft giữa các request | Đã làm rõ Web UI sở hữu DTO, mỗi edit truyền full draft, backend stateless |
| #2 Nguồn tạo Workload Output | UC-03 sử dụng workloadOutputs nhưng chỉ ghi chú lấy từ graph | Đã có resolver, dữ liệu output và quy tắc PLAN_TIME/circular |
| #3 Scope khi tìm Resource Instance | Lookup theo resolved resources + target, schema thiếu consumer binding | Đã thêm exact active binding và sharing policy; recovery cho instance chưa READY còn thiếu (R6) |
| #4 UC-04 gọi provider khi thiếu prerequisite | Gọi infrastructure/CD/Kubernetes không có guard | Đã thêm guard và PENDING/NOT_AVAILABLE; vẫn thiếu đối chiếu phiên bản workload (R5) |
| #5 Xóa definition làm hỏng FK/history | Contract 1 yêu cầu xóa item bị bỏ, đồng thời không thay đổi history/configuration | Đã dùng retired_at + RESTRICT để bảo vệ identity/FK; chính sách thêm lại tên và xử lý binding cũ cần làm rõ |
| #6 Infrastructure Plan chưa có typed model | Có fingerprint nhưng chưa có Plan/Item/Override Definition trong domain | Đã thêm typed transient model; giao thức confirm vẫn có lỗi R2, không bảo vệ toàn bộ deployment input (R4) |
| #7 Writer của progress marker | Chưa có writer rõ cho từng marker hiển thị | Đã có 3 step nội bộ và 2 marker live; còn khoảng trống ở bước collect output (R8) |
| #8 Lẫn lifecycle với CD delivery status | Contract 9 cho record.status phản ánh delivery status rồi đồng bộ deployment.status | Đã tách delivery_status khỏi lifecycle trong schema/domain/contracts |
| #9 Enum chưa chốt | Chỉ mô tả logical state và kiểu ENUM | Đã chốt các literal và đồng bộ enum registry/schema/ERD/domain |
| #10 Confirm chạy tác vụ dài trong HTTP | Reconcile tới publish diễn ra trong request confirm | Đã chuyển sang atomic enqueue + worker; retry và hoàn tất job chưa an toàn theo đặc tả (R1, R7) |
| #11 Secret bị orphan khi save lỗi | Store secret trước save, không đặc tả compensation | Đã thêm staging/promote/revoke/TTL; chưa đóng khoảng trống commit–promotion (R3) |

Hai finding bổ sung trong commit: transition stale-plan đã đổi đúng thành RUNNING → FAILED; canonical lookup đã được thống nhất là binding, không fallback owner columns. Tuy nhiên đổi transition không khắc phục việc retry tự làm fingerprint đổi (R1).

Bằng chứng baseline nổi bật: [main Contract 1](https://github.com/pr3s3nt/final_idp/blob/11ad58249f58e06108ab86da787223254be9b05f/04_operation_contracts/operation_contracts.md#L23), [main confirm](https://github.com/pr3s3nt/final_idp/blob/11ad58249f58e06108ab86da787223254be9b05f/04_operation_contracts/operation_contracts.md#L96), [main record status](https://github.com/pr3s3nt/final_idp/blob/11ad58249f58e06108ab86da787223254be9b05f/04_operation_contracts/operation_contracts.md#L195), [main UC-04](https://github.com/pr3s3nt/final_idp/blob/11ad58249f58e06108ab86da787223254be9b05f/sequence_digrams/uc_04_view_deployment_result.puml#L37).

## R1 — P1: Retry tự coi kết quả lần chạy trước là stale plan

**Phạm vi:** phát sinh khi draft bổ sung worker/retry.

**Bằng chứng:** [Contract 4: fingerprint có action và instance identity](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L32); [Contract 6: mỗi attempt rebuild và mismatch là terminal](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L45); [UC-03 persist instance rồi tiếp tục execution](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_03_deploy_application.puml#L127).

**Kịch bản:** plan được accept với resource CREATE, chưa có instance. Attempt 1 tạo resource, lưu READY instance/binding, rồi worker chết hoặc một phase sau gặp lỗi retryable. Attempt 2 đọc current instances và dựng plan thành REUSE/UPDATE với instance ID đã tồn tại. Action và instance ID thuộc fingerprint nên fingerprint khác accepted value. Contract 6 yêu cầu fail PLAN_STALE_AFTER_ACCEPT trước khi có thể tiếp tục phase còn lại.

Provider idempotency không giải quyết được vì job đã bị chặn tại fingerprint check. Đặc tả cũng có thể ghi step INFRASTRUCTURE_READY thành FAILED dù hạ tầng của attempt trước đã thành công.

**Hướng sửa:** phân biệt input/intent đã accept với execution progress/current provider state; lưu checkpoint/resource identity đủ để resume từng item/phase. Không bắt retry so sánh fingerprint của “CREATE trước khi chạy” với plan suy ra từ “resource đã tạo xong”.

**Điều kiện đóng:** mô tả và kiểm tra được kịch bản tạo hạ tầng xong → worker chết → retry → resolve/publish thành công, không tạo resource trùng, không fail stale giả.

## R2 — P1: Confirm không gắn với plan mà người dùng thực sự đã xem

**Phạm vi:** tồn tại ở main và chưa được sửa trong draft.

**Bằng chứng:** [signature và xử lý PLAN_CHANGED, Contract 5](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L35).

Request chỉ truyền deploymentId, overrides và idempotencyKey. Server so rebuilt fingerprint với fingerprint đang lưu trong DB; khi khác thì cập nhật fingerprint DB.

**Kịch bản:** UI xem plan A. Catalog/state đổi thành B. Confirm đầu tiên cập nhật DB sang B và trả PLAN_CHANGED, nhưng phản hồi bị mất. Client retry cùng request trong khi vẫn hiển thị A. Server dựng B, thấy B khớp DB và enqueue. Như vậy B được accept dù người dùng chưa review B. Hai tab cùng mở deployment cũng gây tình huống tương tự.

**Hướng sửa:** confirm phải mang expected/reviewed plan fingerprint hoặc review revision từ UI; so sánh cả token của client, current plan và giá trị CAS. Mất response PLAN_CHANGED phải tiếp tục trả PLAN_CHANGED/conflict cho request mang token cũ. Lookup idempotency result của một confirm đã accept cần được đặc tả trước guard trạng thái/rebuild.

**Điều kiện đóng:** lost-response và two-tab scenario không thể accept plan mà request chưa xác nhận.

## R3 — P1: Configuration được công bố trước khi secret staging được promote

**Phạm vi:** issue #11 được sửa một phần trong draft.

**Bằng chứng:** [UC-02 commit DB trước promote](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_02_configure_application_environment.puml#L90); [Contract 3](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L19); [schema Environment Configuration](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/03_database_erd/schema.md#L96).

**Kịch bản:** transaction DB commit configuration có staged reference; tiến trình chết trước promote hoặc Secret Store tạm lỗi. Configuration mới đã đọc được từ DB, trong khi reference chưa permanent và còn TTL. UC-03 không có gate “promotion hoàn tất”. Nếu reference hết hạn trước recovery, DB có thể trỏ đến secret không còn tồn tại.

Tài liệu có nhắc retry/orphan reconciliation, nhưng không chỉ rõ durable pending work, trạng thái publication, actor recovery, hoặc quy tắc ngăn TTL xóa reference đã commit. “Không trả HTTP success trước promote” không bảo vệ các reader khác sau DB commit.

**Hướng sửa:** đặc tả durable promotion intent và publication/readiness gate cho configuration; recovery phải biết reference nào đã commit để promote/pin thay vì cleanup. Nêu rõ hành vi khi promote thất bại, TTL tới hạn và caller retry.

**Điều kiện đóng:** fault ở mọi ranh giới stage/commit/promote vẫn giữ cấu hình cũ usable hoặc cấu hình mới pending; không cho deploy dùng reference chưa sẵn sàng/đã hết hạn.

## R4 — P1: Worker dùng definition/configuration hiện tại thay vì input đã accept

**Phạm vi:** main đã dùng mutable references; async worker ở draft mở rộng khoảng thời gian chịu ảnh hưởng.

**Bằng chứng:** [worker đọc Current definition/configuration](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_03_deploy_application.puml#L111); [fingerprint chỉ mô tả infrastructure plan](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L32); [deployment FK tới configuration](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/03_database_erd/schema.md#L223).

Workload Deployment snapshot image và Deployment Context đã persist. Application Definition và Environment Configuration vẫn là dữ liệu có thể sửa tại chỗ; chưa có revision/snapshot bất biến của toàn bộ input.

**Kịch bản:** confirm/enqueue deployment D với một direct environment value; trước khi worker chạy, UC-02 đổi giá trị đó. Worker đọc giá trị mới. Thay đổi direct value không nhất thiết ảnh hưởng resource plan; đặc biệt app không có resource có items rỗng, nên fingerprint có thể giữ nguyên. D deploy configuration chưa nằm trong lần accept. Tương tự với metadata workload không ảnh hưởng infrastructure.

**Hướng sửa:** pin revision bất biến của definition/configuration và secret reference version, hoặc chốt rõ “deploy latest” và bổ sung kiểm tra/review lại toàn bộ input tương ứng. Không cần persist plaintext hay transient resolved values để giữ input identity ổn định.

**Điều kiện đóng:** sửa UC-01/02 sau enqueue không âm thầm thay đổi nội dung deployment đã accept.

## R5 — P2: APPLICATION_READY có thể dựa trên workload của lần deploy khác

**Phạm vi:** khoảng trống ở main, còn trong draft dù guard UC-04 đã được bổ sung.

**Bằng chứng:** [query Kubernetes theo target/workloads sau publish](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_04_view_deployment_result.puml#L49); [Contract 12: mọi workload Healthy là SUCCEEDED](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L86).

**Kịch bản:** v1 đang Healthy; v2 vừa được CD accept nhưng chưa rollout. deliveryReference tồn tại, query trả workload v1 Healthy. Quy tắc hiện tại chưa yêu cầu observed revision/generation/image tương ứng v2, nên có thể suy ra APPLICATION_READY cho v2 quá sớm. Khi mở deployment cũ cũng có thể ghép image snapshot cũ với health live của bản mới.

**Hướng sửa:** quy định correlation giữa deployment/delivery revision và observed workload generation/image/identity; chỉ báo ready khi bản mong đợi đã thực sự hội tụ. Với deployment cũ đã bị thay thế, UI phải thể hiện rõ trạng thái live không phải bằng chứng readiness lịch sử.

**Điều kiện đóng:** v1 Healthy trong lúc v2 chưa chạy không làm v2 SUCCEEDED; xem history không gán health sai revision.

## R6 — P2: Lookup chỉ READY không hiện thực được retry Resource Instance FAILED

**Phạm vi:** khoảng trống phát sinh trong quy tắc lookup của draft.

**Bằng chứng:** [Contract 4 lookup READY](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L31); [schema lọc READY và active binding unique](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/03_database_erd/schema.md#L194); [state machine FAILED → PLANNED](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/05_state_machines/resource_instance_state.puml#L18).

**Kịch bản:** update một instance đã có OWNER binding nhưng provider thất bại, instance thành FAILED. Lần deploy/retry kế tiếp chỉ nhận READY instances nên không lấy được identity cần retry. Nếu planner coi kết quả rỗng là cần CREATE, OWNER binding mới lại xung đột với unique active tuple của binding cũ. Nếu không CREATE thì chưa có đường execution nào thực hiện FAILED → PLANNED như state machine.

**Hướng sửa:** vẫn bắt đầu từ exact active binding, nhưng phân biệt lookup instance hiện hữu ở mọi trạng thái với kiểm tra có đủ điều kiện REUSE hay không. FAILED phải có explicit recovery policy; PROVISIONING phải có busy/resume policy; replacement phải retire binding đúng transaction.

**Điều kiện đóng:** instance FAILED được retry/recover đúng identity, không mất dấu provider state và không va unique binding.

## R7 — P2: Lưu SUBMITTED và hoàn tất job chưa có cùng ranh giới transaction

**Phạm vi:** draft.

**Bằng chứng:** [sequence gọi saveDeploymentRecord rồi completeJob riêng](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_03_deploy_application.puml#L178); [Contract 6 chỉ cho execute khi deployment QUEUED/RUNNING](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L45).

**Kịch bản:** worker lưu lifecycle/record SUBMITTED rồi chết trước completeJob. Job còn CLAIMED và sẽ hết lease, nhưng deployment đã terminal SUBMITTED, không thỏa precondition retry. Đặc tả chưa cho biết job được complete hồi phục thế nào, và sequence không đánh dấu hai write này atomic.

**Hướng sửa:** transaction chung cho job SUCCEEDED + deployment/record SUBMITTED + delivery metadata, hoặc đặc tả recovery riêng nhận biết publish đã hoàn tất và chỉ hoàn tất bookkeeping. Các cập nhật cần gắn với lease/claim hợp lệ.

**Điều kiện đóng:** crash giữa các write không để job CLAIMED/QUEUED kẹt cạnh deployment SUBMITTED.

## R8 — P2: Collect output thất bại khi chưa có step nào RUNNING

**Phạm vi:** writer của draft được bổ sung nhưng chưa phủ hết execution.

**Bằng chứng:** [UC-03 đóng step infrastructure, collect outputs, rồi mới mở step configuration](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_03_deploy_application.puml#L139); [Contract 8 yêu cầu failure ghi active step](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/04_operation_contracts/operation_contracts.md#L59).

**Kịch bản:** INFRASTRUCTURE_READY đã SUCCEEDED; collectResourceOutputs hoặc resolvePlanTimeWorkloadOutputs gặp lỗi. CONFIGURATION_RESOLVED chỉ được chuyển RUNNING sau cả hai thao tác. “Fail active internal step” lúc này không xác định được step nào.

**Hướng sửa:** bắt đầu CONFIGURATION_RESOLVED trước việc thu thập/resolve output, hoặc quy định rõ mapping error của các thao tác này vào phase nào. Những step còn lại sau terminal failure cũng cần quy tắc PENDING/SKIPPED nhất quán.

**Điều kiện đóng:** mọi điểm lỗi trong chuỗi worker được ánh xạ đến step/lifecycle/error record xác định.

## R9 — P2: App không có variable/secret chưa có đường deploy nhất quán

**Phạm vi:** main và draft.

**Bằng chứng:** [UC-02 yêu cầu app có variable hoặc secret cần cấu hình](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/usecase_realization_step_1_3.md#L239); [deployment bắt buộc environment_configuration_id](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/03_database_erd/schema.md#L223).

UC-01/domain cho phép workload không có configuration requirement, ví dụ static service chỉ cần image và port. Nhưng UC-02 không áp dụng theo precondition đang viết; UC-03 lại cần Environment Configuration tồn tại. Chưa có operation nào được đặc tả tự tạo empty configuration cho trường hợp này.

**Hướng sửa:** cho phép UC-02 tạo cấu hình rỗng, hoặc UC-03 tạo/chọn một empty configuration hợp lệ theo transaction và invariants rõ ràng; đồng bộ high-level use case và contracts.

**Điều kiện đóng:** app chỉ có image/port vẫn đi được từ tạo application đến enqueue/deploy.

## R10 — P2: VOPC chưa đồng bộ với sequence dù traceability đánh PASS

**Phạm vi:** draft.

**Bằng chứng:** [UC-02 VOPC ConfigRepo chỉ có save](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/01_vopc_design_class_diagram/vopc_uc02.puml#L73), trong khi [UC-02 sequence gọi findByApplicationAndEnvironment](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_02_configure_application_environment.puml#L28).

[Consolidated VOPC phần DeploymentWorker](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/01_vopc_design_class_diagram/design_class_diagram.puml#L347) và VOPC UC-03 chưa có dependency từ Worker tới Application Repository, Environment Configuration Repository, Deployment Graph Builder, Resource Definition Resolver và Resource Instance Repository. [Sequence worker](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/sequence_digrams/uc_03_deploy_application.puml#L111) gọi trực tiếp cả năm component đó.

Đếm class/operation khớp không chứng minh đúng caller–callee, signature hoặc failure path. Vì vậy [Overall acceptance PASS](https://github.com/pr3s3nt/final_idp/blob/88585ccdd1a7896c615ad43fbb0d0f6b62d231ef/06_traceability/traceability_matrix.md#L111) đang mạnh hơn bằng chứng.

**Hướng sửa:** bổ sung operation/dependency tương ứng và kiểm tra traceability theo message receiver, signature, persistence writer và alternate/recovery path; tách coverage PASS khỏi behavioral acceptance.

**Điều kiện đóng:** mỗi direct sequence call có collaborator/operation tương ứng hoặc quy ước giản lược được ghi rõ; acceptance không che các finding chưa đóng.

## Những lựa chọn thiết kế còn cần chốt

Các mục sau chưa được tính vào 10 findings vì có thể là policy có chủ đích, nhưng tài liệu chưa nêu rõ:

- Sau retire, tên workload/requirement/variable/secret vẫn chịu UNIQUE không có điều kiện retired_at. Cần nói rõ thêm lại cùng tên là restore identity cũ, bị cấm, hay tạo identity mới bằng partial unique. Không nên tự sửa sang partial unique nếu chưa chốt semantics history.
- Retire một workload/definition bảo toàn FK, nhưng cấu hình cũ còn tham chiếu retired target: cần xác định lọc bỏ, yêu cầu người dùng sửa hay migration; tránh chỉ ghi “active definitions hợp lệ” mà không có đường xử lý cấu hình đang tồn tại.
- Application Definition được save rồi specification được generate/save trong operation kế tiếp. Nếu generation lỗi, definition mới có thể tồn tại cạnh artifact cũ; cần chốt transaction/recovery/version gating cho hậu điều kiện Save Application.
- Job lease chưa mô tả rõ heartbeat/fencing. Khi worker cũ hết lease vẫn chạy, idempotency provider không tự bảo vệ các write lifecycle/progress của worker cũ.
- Override có thể đổi resolved parameters sau khi action CREATE/UPDATE/REUSE đã được tính. Cần chốt action có được tính lại khi override biến một REUSE thành UPDATE không.
- Workload Output Resolver tính DNS trước target adaptation; cần quy ước tên Service/namespace/port chung với renderer/adapter để output trỏ đúng manifest cuối.
- UC-03 chưa mô tả rõ đường đọc Resource Output sau restart từ provider state và đường biến opaque secret reference thành credential/Kubernetes reference. Vì outputs được chủ đích transient, repository không thể chỉ dựa vào memory của lần reconcile trước.
- Không xác nhận được cú pháp render của inline class body tại vopc_uc03.puml:116; cần render cả 13 diagram bằng phiên bản PlantUML được dự án chọn trước khi acceptance.

## Thứ tự xử lý đề xuất tại thời điểm review baseline

1. Chốt phạm vi MVP và chính sách retry/secret; sau đó sửa confirm token (R2), pin deployment input (R4), thiết kế recovery và atomic completion theo phạm vi đó (R1, R6, R7).
2. Đóng secret commit/promotion protocol (R3).
3. Sửa readiness correlation và phase error ownership (R5, R8).
4. Bổ sung zero-configuration path (R9), đồng bộ VOPC/traceability (R10), chốt các policy còn mở.
5. Review lại bằng các kịch bản nêu trong từng finding rồi mới đổi Overall acceptance thành PASS.

Các ranh giới platform administration của Resource Definition catalog và SHARED_CONSUMER authorization đã được nêu rõ; review này không coi việc chúng nằm ngoài bốn Developer UC là lỗi.
