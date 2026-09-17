---
id: LEGACY-DESIGN-DECISION-LOG
artifact: consolidated-decision-log
status: historical
current_index: ../../decisions/README.md
last_reviewed: 2026-09-17
---

> **Historical consolidated record.** Use the split [ADR index](../../decisions/README.md) to determine current decision status. The completed comparison is recorded in [`documentation-reconciliation.md`](../../implementation/documentation-reconciliation.md). This file is retained for chronology and provenance, and its summary rows may describe an earlier implementation state.

# Nhật ký quyết định thiết kế

Tài liệu này ghi lại quyết định cho từng vấn đề thiết kế khi rà soát lại tài liệu trên nhánh `refine_design` (bắt đầu từ `11ad582`). Danh sách 11 vấn đề lấy theo commit message của `88585cc`; các cách sửa trong commit đó chỉ dùng để tham khảo, chưa được xác nhận là đúng.

Cách làm: bàn và chốt lần lượt từng vấn đề, ghi quyết định vào đây; sau khi chốt hết mới sửa tài liệu một lượt theo thứ tự **use case realization → sequence diagram → VOPC → domain model → ERD → operation contracts → state machine → traceability**.

Từ vấn đề 12 trở đi, vấn đề phát sinh khi đối chiếu bản cài đặt UC-03 (`uc03/`, danh sách lệch thiết kế ở `implementation_plan.md` §10 và §12) với tài liệu thiết kế; mỗi vấn đề được bàn, chốt rồi sửa tài liệu ngay.

**Về file `usecase_realization_step_1_3.md`.** File được tổ chức theo thứ tự:

1. **Đặc tả use case:** viết lần lượt UC-01, UC-02, UC-03, UC-04 (mục tiêu, tiền/hậu điều kiện, luồng chính, luồng ngoại lệ, dữ liệu chính, quy tắc nghiệp vụ).
2. **Bước 1:** heading `# Bước 1 Chốt responsibility của từng Use Case`, chốt trách nhiệm cho từng use case.
3. **Bước 2:** heading `# Bước 2 Xác định các System Operation chính`, liệt kê system operation cho từng use case.
4. **Bước 3:** heading `# Bước 3 Xác định các thành phần tham gia`, liệt kê thành phần tham gia cho từng use case.

Trong mỗi Bước, từng use case là một mục con `## UC 01 …`, `## UC 02 …`, `## UC 03 …`, `## UC 04 …`. Vì vậy trong nhật ký này, **"Bước 2 UC-03"** nghĩa là mục `## UC 03 Deploy Application` nằm trong phần Bước 2; **"Đặc tả UC-03"** nghĩa là phần đặc tả của UC-03 ở đầu file.

## Tổng quan

| # | Vấn đề | Quyết định | Trạng thái áp dụng |
|---|---|---|---|
| 1 | Bản nháp UC-01/UC-02 được giữ ở đâu giữa các request | Hoãn sau MVP | Đã ghi vào `deferred_issues.md` (D1) |
| 2 | Giá trị Workload Output do ai tính | Triển khai theo thứ tự phụ thuộc (theo tầng) | Đã áp dụng vào toàn bộ tài liệu thiết kế; phần hoãn ghi vào D8 |
| 3 | Scope khi tìm Resource Instance để reuse | Mặc định riêng; dùng chung phải khai báo tường minh ở Resource Definition (theo mô hình Humanitec) | Đã áp dụng vào toàn bộ tài liệu thiết kế; phần hoãn ghi vào D9 |
| 4 | UC-04 gọi provider khi thiếu điều kiện | Hoãn, giải quyết sau | Đã ghi vào `deferred_issues.md` (D2) |
| 5 | Xóa definition làm hỏng FK/lịch sử | Định nghĩa app có phiên bản; deploy chọn phiên bản vào từng environment (staging, production); hạ tầng/pod bị gỡ khi environment deploy phiên bản không còn thành phần đó | Đã áp dụng vào toàn bộ tài liệu thiết kế |
| 6 | Infrastructure Plan chưa có typed model | Hoãn, xem xét sau | Đã ghi vào `deferred_issues.md` (D3) |
| 7 | Ai ghi progress marker | Hoãn, giải quyết sau | Đã ghi vào `deferred_issues.md` (D4) |
| 8 | Lẫn lifecycle status với CD delivery status | Hoãn, giải quyết sau | Đã ghi vào `deferred_issues.md` (D5) |
| 9 | Literal ENUM chưa chốt | Một bảng danh mục ENUM duy nhất trong `schema.md`; chốt giá trị cho các ENUM thuộc vấn đề đã chốt, ghi giá trị dự kiến cho các ENUM thuộc D3/D4/D5 | Đã áp dụng vào toàn bộ tài liệu thiết kế (danh mục ENUM trong `schema.md`); giá trị dự kiến đã ghi vào D3, D4, D5, D6 |
| 10 | Confirm chạy tác vụ dài trong HTTP request | `confirmDeployment` chỉ nhận việc (đổi status + tạo job trong DB cùng một transaction) và trả lời ngay; Deployment Worker chạy nền. Phục hồi khi worker chết: hoãn | Đã áp dụng vào toàn bộ tài liệu thiết kế; phần hoãn ghi vào `deferred_issues.md` (D6) |
| 11 | Secret bị orphan khi save lỗi | Hoãn, giải quyết sau | Đã ghi vào `deferred_issues.md` (D7) |
| 12 | Nơi triển khai và hạ tầng của nó; phiên bản catalog | Hai loại nơi triển khai: cloud (IDP dựng VPC và cụm) và cụm Kubernetes nội bộ (có sẵn); cụm/VPC nằm trong đồ thị deploy; dưới đổi thì trên làm lại; catalog có phiên bản, Developer chọn khi deploy | Đã áp dụng vào toàn bộ tài liệu thiết kế (code chưa sửa); phần hoãn ghi vào D10, D11, D12 |
| 13 | Desired state của mọi application nằm chung một nơi | Mỗi application có một Delivery Repository riêng; IDP tự tạo repo và sinh cặp khóa riêng cho app ở lần deploy đầu | Đã áp dụng vào toàn bộ tài liệu thiết kế và code |
| 14 | Hệ thống CD cụ thể là Argo CD | Đổi mặc định sang Fleet, giữ cả hai adapter và chọn bằng cấu hình | Đã áp dụng; kiến trúc không đổi vì CD vốn là abstraction |
| 15 | Không có thao tác gỡ application khỏi một environment, dù tài liệu đã nhắc tới nó | Tách thành use case riêng **UC-05 – Remove Application from Environment** | Đã áp dụng vào tài liệu thiết kế; code đã có sẵn từ trước |

## Vấn đề 1 — Bản nháp UC-01/UC-02

**Quyết định:** chưa giải quyết ở MVP.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D1.

## Vấn đề 2 — Workload Output / triển khai theo thứ tự phụ thuộc

**Vấn đề:** UC-02 cho phép gán `BACKEND_URL ← backend.endpoint`, nhưng UC-03 không có thành phần nào lấy được giá trị output của workload; `workload.exposed_outputs` chỉ là danh sách tên.

**Quyết định:**

1. Đồ thị: node là Resource Requirement và Workload; cạnh chỉ lấy từ Dependency khai báo ở UC-01. Không khai báo thì hai thành phần độc lập. Đồ thị có vòng → A1.
2. UC-02 chỉ cho tham chiếu output (resource output, sensitive output, workload output) của thành phần mà workload đã khai báo depends on; ngược lại → A1.
3. Thực thi theo tầng. Mỗi tầng: resolve cấu hình → triển khai → chờ sẵn sàng → thu output.
4. Workload "sẵn sàng" nghĩa là pod healthy.
5. Cơ chế đọc workload output để ở mức trừu tượng (Workload Output Collector đọc từ workload đang chạy), chưa chi tiết.
6. Cho phép deploy một phần app. Workload phụ thuộc không nằm trong deployment thì lấy output từ bản đang chạy cùng environment + target; chưa từng chạy → A1.
7. Output của một thành phần đổi sau khi deploy lại → tự động đưa các thành phần phụ thuộc (bắc cầu) vào các tầng sau của cùng deployment, dùng image version đang chạy. Plan hiển thị trước danh sách thành phần có thể bị deploy lại. So sánh bằng dấu vân tay (hash) output, không lưu giá trị.
8. Deploy một phần chỉ reconcile resource mà các workload được chọn phụ thuộc trực tiếp; workload phụ thuộc không được chọn thì đọc output từ bản đang chạy, không đi sâu tiếp.
9. Thêm aggregate Workload Instance: mỗi workload một dòng theo environment + target, gồm image đang chạy, trạng thái, dấu vân tay output; đối xứng Resource Instance, có repository riêng (thành sáu repository).
10. `deployment.status` rút gọn thành `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED`; tiến trình chi tiết theo tầng/thành phần/bước nằm ở `deployment_step`. `SUCCEEDED` nghĩa là mọi workload trong phạm vi healthy.

**Bổ sung khi sửa use case (đã được duyệt):**

- Thêm thành phần **Deployment Wave Planner** chịu trách nhiệm chia tầng và lan truyền thay đổi output.
- UC-04 đọc **Workload Instance Repository** để cho biết deployment đang xem có còn là bản đang chạy hay không.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):**

- `usecase_realization_step_1_3.md`: đặc tả UC-01 đến UC-04 và Bước 1–3.
- Sequence diagram: `uc_02`, `uc_03` (luồng request và luồng Deployment Worker theo tầng), `uc_04`.
- VOPC: `docs/use-cases/UC-02/vopc.puml`, `docs/use-cases/UC-03/vopc.puml`, `docs/use-cases/UC-04/vopc.puml`, `docs/architecture/design-class-diagram.puml`, `docs/architecture/design-classes.md` (Deployment Wave Planner, Workload Output Collector, Workload Instance Repository).
- Domain model: Workload Instance, Workload Output, Deployment Graph có phạm vi/tầng, Workload Deployment có `inclusionReason`/`waveNumber`.
- ERD: `workload_instance`, `output_fingerprint`, `inclusion_reason`, `wave_number`.
- Operation contracts: C4, C6–C9; thêm C10 `collectWorkloadOutputs`, C11 `propagateOutputChanges`.
- State machines: Deployment (`DEPLOYING`/`SUCCEEDED`), thêm Workload Instance.
- Traceability matrix.
- `deferred_issues.md`: D8 — cơ chế cụ thể đọc workload output.

**Lưu ý cho các vấn đề sau:** quyết định 10 đụng tới vấn đề 7, 8, 9; việc chờ pod healthy qua nhiều tầng đụng tới vấn đề 10.

## Vấn đề 3 — Tìm Resource Instance để dùng lại theo tiêu chí nào

**Vấn đề:** UC-03 tìm Resource Instance có sẵn bằng `findResourceInstances(resolvedResources, target)`, tức chỉ theo loại Resource Definition và deployment target. Resource Instance trong domain model và ERD cũng chỉ lưu `resourceDefinitionId` và `deploymentTarget`, không ghi thuộc app, environment hay requirement nào. Hệ quả là IDP có thể dùng nhầm resource:

- Hai app khác nhau cùng cần PostgreSQL RDS trên target `prod` → app sau dùng nhầm database của app trước.
- Hai environment (`dev`, `staging`) cùng deploy lên một cluster → dùng chung database ngoài ý muốn.
- Một app có hai requirement cùng loại (`orders-db`, `users-db`) → không phân biệt được instance nào của requirement nào.

Vấn đề này cũng ảnh hưởng vấn đề 2: deploy một phần phải đọc output của đúng resource mà workload phụ thuộc, và dấu vân tay output được lưu trên Resource Instance.

**Quyết định:** mặc định mỗi resource là riêng; dùng chung phải khai báo tường minh — theo mô hình Humanitec.

1. **Resource Instance thuộc về đúng một chủ:** application + environment + resource requirement + deployment target. Tìm để dùng lại theo đủ bộ này; không khớp thì tạo mới. Resource Instance tương ứng với "active resource" của Humanitec.
2. **Dùng chung giữa các app hoặc environment được khai báo ở Resource Definition** (do platform quản lý): platform tạo definition trỏ tới resource có sẵn, kèm điều kiện áp dụng (ví dụ mọi app ở environment `dev`). App khớp điều kiện thì Resource Instance của nó trỏ tới resource chung đó. Không dùng bảng binding cấp quyền giữa các app (hướng của `88585cc` không được chọn).
3. **Resource Instance trỏ tới resource có sẵn không được tạo, sửa hay xóa hạ tầng thật**, chỉ đọc output. Nhờ vậy deploy của app này không thể làm thay đổi resource mà app khác đang dùng.

Dùng chung trong cùng app + environment đã có sẵn trong thiết kế: nhiều workload cùng depends on một Resource Requirement thì dùng chung Resource Instance của requirement đó. Mức "resource riêng của từng workload" chưa cần trong giai đoạn này.

**Hệ quả cần xử lý khi sửa tài liệu:**

- `resource_instance` thêm liên kết tới application, environment, resource requirement; khóa tìm kiếm là (application, environment, resource requirement, deployment target).
- Bỏ ràng buộc `UNIQUE` trên `resource_instance.infrastructure_reference`, vì nhiều Resource Instance có thể trỏ cùng một resource chung.
- Resource Definition cần phân biệt loại **quản lý hạ tầng** (tạo/sửa/xóa) với loại **trỏ tới resource có sẵn** (chỉ đọc output).
- `findResourceInstances(...)` đổi tiêu chí tìm theo đủ bộ chủ sở hữu.

**Hoãn (đã ghi vào `deferred_issues.md`, mục D9):**

- Khi output của resource dùng chung thay đổi, chỉ app nào deploy lần sau mới phát hiện qua dấu vân tay output; IDP không tự deploy lại các app khác đang dùng chung.
- Resource riêng của từng workload (mức private theo workload của Humanitec).

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (UC-03, Bước 2, Bước 3), sequence UC-03, VOPC (`findResourceInstances` theo khóa chủ sở hữu), domain model (Resource Instance có chủ sở hữu; Resource Definition có `managementMode`), ERD (`resource_instance` có cột chủ sở hữu, partial UNIQUE, bỏ UNIQUE `infrastructure_reference`; `resource_definition.management_mode`), contracts 4–6, state machine Resource Instance (`EXISTING` → `READY`, `UNLINKED`), traceability; phần hoãn ghi vào D9.

**Sẽ ảnh hưởng (danh sách ban đầu):**

- `usecase_realization_step_1_3.md`:
  - Đặc tả UC-03: quy tắc nghiệp vụ về tìm resource để dùng lại theo chủ sở hữu và dùng chung khai báo ở Resource Definition.
  - Bước 2 UC-03: mô tả `resolveResourceDefinitions()`, `planInfrastructureChanges()`, `reconcileInfrastructure()` (tìm theo đủ bộ chủ sở hữu; resource loại `EXISTING` chỉ đọc output).
  - Bước 3 UC-03: Resource Definition Resolver, Infrastructure Planner, Infrastructure Reconciler, Resource Instance Repository.
- Các artifact khác: sequence UC-03, VOPC (`Resource Instance Repository`), domain model (Resource Instance, Resource Definition), ERD (`resource_instance`, `resource_definition`), operation contracts 4–6, state machine Resource Instance (instance trỏ resource có sẵn không đi qua provisioning), traceability.

## Vấn đề 4 — UC-04 gọi hệ thống bên ngoài mà không kiểm tra điều kiện

**Vấn đề:** khi Developer mở bất kỳ deployment nào, sequence UC-04 (`docs/use-cases/UC-04/sequence.puml:38-53`) luôn gọi đủ `getInfrastructureStatus`, `getCDStatus`, `getWorkloadStatus`, `getDeploymentEndpoints` mà không kiểm tra deployment đã tới bước tương ứng chưa. Hệ quả:

- Deployment thất bại trước khi gửi sang CD vẫn bị gọi `getCDStatus(NULL)` vì `delivery_reference` rỗng.
- **Hiển thị "Healthy" giả:** Kubernetes trả lời về workload đang chạy trên cluster, không phải của deployment đang xem. Deployment #42 thất bại trước khi deploy vẫn hiện "backend Healthy" — thực ra là bản của #41 đang chạy. Mở lại một deployment cũ thì bị gán health của bản mới hơn.
- Deployment ở `AWAITING_CONFIRMATION` chưa có record, hạ tầng hay delivery nhưng vẫn bị hỏi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D2 — gồm các kịch bản lỗi, hướng giải quyết đã đề xuất (kết quả lấy từ DB; trạng thái trực tiếp chỉ hỏi khi đủ điều kiện) và các câu hỏi cần chốt.

**Lưu ý khi giải quyết sau:** tận dụng hai quyết định của vấn đề 2 — `deployment_step` theo tầng/thành phần (kết quả của deployment đã có trong DB) và Workload Instance (biết deployment đang xem còn là bản đang chạy hay không). Khi sửa sequence UC-04 trong lượt sửa chung cho vấn đề 2, không được làm lỗi này nặng thêm.

**Đã áp dụng:** `deferred_issues.md` (D2). Chưa sửa file thiết kế nào.

## Vấn đề 5 — Xóa thành phần khỏi application làm hỏng lịch sử

**Vấn đề:** Contract 1 (`saveApplicationDefinition`) quy định thành phần bị loại khỏi definition thì **bị xóa hẳn** row, nhưng cũng trong contract đó cam kết không tạo, xóa hay sửa `Deployment`, `Workload Deployment`, `Environment Configuration`, `Resource Instance`. Hai điều này mâu thuẫn, vì nhiều bảng đang trỏ tới workload/definition qua khóa ngoại:

| Bảng | Trỏ tới |
|---|---|
| `workload_deployment.workload_id` | workload đã deploy (lịch sử) |
| `environment_variable.variable_definition_id`, `secret.secret_definition_id` | cấu hình theo environment |
| `configuration_value.workload_id`, `configuration_value.resource_requirement_id` | tham chiếu output |
| `dependency.target_workload_id`, `dependency.target_resource_requirement_id` | quan hệ phụ thuộc |

Vấn đề 2 và 3 còn thêm Workload Instance → workload và Resource Instance → resource requirement.

Ví dụ `worker` đã deploy 20 lần; bảng lịch sử chỉ lưu mã workload (W7) rồi tra bảng `workload` để hiển thị tên. Xóa hẳn dòng W7 thì chỉ có ba khả năng, đều sai: DB từ chối xóa (không bao giờ xóa được workload đã deploy); DB xóa dây chuyền (mất lịch sử); hoặc lịch sử trỏ vào chỗ trống (UC-04 hiển thị `???`). Thiết kế cũng không nói hạ tầng thật (database trên AWS, pod trên cluster) xử lý thế nào khi thành phần bị xóa.

**Quá trình bàn:**

1. Người dùng đưa quy tắc: muốn xóa một thành phần thì phải sửa trước những gì đang dùng nó (dependency, biến môi trường tham chiếu output của nó).
2. Quy tắc đó xử lý được phần "đang dùng" nhưng không xử lý được lịch sử; đã cân nhắc "ngừng dùng" (`retired_at`) và "lịch sử lưu bản chụp" (snapshot).
3. Gỡ hạ tầng ngay khi Save ở UC-01 bị loại: UC-01 không biết environment nào, app đang chạy vẫn còn dùng, hủy database là không lấy lại được, resource dùng chung không được hủy.
4. Phát hiện gốc vấn đề: cả hai environment dùng chung **một** bản định nghĩa app bị ghi đè khi Save, nên developer không thể thử thay đổi ở staging mà giữ nguyên production.

**Quyết định (cách B — định nghĩa app có phiên bản):**

1. **Environment cố định:** mọi application có đúng hai environment `staging` và `production`; không khai báo environment ở UC-01.
2. **Application Definition có phiên bản bất biến:** mỗi lần Save ở UC-01 tạo một phiên bản mới; phiên bản cũ không bao giờ bị sửa hay xóa.
3. **Deploy chọn phiên bản:** UC-03 deploy một phiên bản cụ thể vào một environment. Thử ở staging xong thì deploy cùng phiên bản đó lên production (promote).
4. **Xóa thành phần** nghĩa là phiên bản mới không còn thành phần đó. Lịch sử deploy trỏ tới phiên bản đã dùng nên vẫn hiển thị đúng; không cần snapshot riêng.
5. **Kiểm tra khi Save ở UC-01** chỉ trong nội bộ phiên bản: dependency trỏ tới thành phần có thật, không tạo vòng.
6. **Kiểm tra khi deploy phiên bản X vào environment Y (UC-03):** cấu hình của Y phải khớp với X; biến còn tham chiếu output của thành phần không có trong X → A1.
7. **Gỡ pod và hạ tầng thật** khi một environment deploy phiên bản không còn thành phần đó: plan hiển thị `REMOVE` (workload) hoặc `DESTROY` (resource, kèm cảnh báo mất dữ liệu) và developer xác nhận. Resource dùng chung (vấn đề 3) chỉ gỡ liên kết, không hủy hạ tầng. Giữa lúc Save và lần deploy đó, mọi thứ đang chạy giữ nguyên.
8. **Đổi sang phiên bản mới phải deploy toàn bộ app;** deploy một phần (vấn đề 2) chỉ dùng phiên bản đang chạy ở environment đó, ví dụ đổi image của một workload. *(Đề xuất đi kèm, ghi nhận cùng cách B.)*
9. **Cấu hình ở UC-02 chưa có phiên bản:** vẫn quản lý theo environment như hiện tại và được kiểm tra khớp với phiên bản khi deploy. *(Đề xuất đi kèm, ghi nhận cùng cách B.)*

**Hệ quả cần xử lý khi sửa tài liệu:**

- Workload và Resource Requirement cần **identity logic ổn định qua các phiên bản**, để Resource Instance và Workload Instance (vấn đề 2, 3) không bị tạo mới mỗi khi có phiên bản mới.
- Deployment ghi phiên bản Application Definition đã dùng.
- Contract 1 bỏ quy định xóa thành phần bị loại, thay bằng tạo phiên bản mới.
- Plan của UC-03 thêm hành động `REMOVE`/`DESTROY` (liên quan vấn đề 6).
- UC-01 A1 chỉ còn lỗi nội bộ phiên bản; UC-03 A1 thêm lỗi cấu hình environment không khớp phiên bản.
- Ví dụ environment `dev staging production` trong UC-02 đổi thành `staging`, `production`.
- Liên quan vấn đề 9: environment trở thành tập giá trị cố định.

**Sẽ ảnh hưởng:**

- `usecase_realization_step_1_3.md`:
  - Đặc tả UC-01: Save tạo phiên bản mới; A1 chỉ còn lỗi nội bộ phiên bản; quy tắc nghiệp vụ về phiên bản.
  - Đặc tả UC-02: environment cố định `staging`, `production`.
  - Đặc tả UC-03: chọn phiên bản để deploy, promote từ staging lên production; plan có hành động gỡ/hủy; A1 thêm lỗi cấu hình environment không khớp phiên bản; đổi phiên bản phải deploy toàn bộ.
  - Đặc tả UC-04: hiển thị phiên bản định nghĩa mà deployment đã dùng.
  - Bước 1: trách nhiệm của UC-01 (lưu phiên bản) và UC-03 (deploy một phiên bản vào một environment, gỡ thành phần không còn trong phiên bản).
  - Bước 2: `saveApplicationDefinition()` (tạo phiên bản), `createDeployment()` (chọn phiên bản), `validateDeploymentInput()` (cấu hình khớp phiên bản), `planInfrastructureChanges()`/`reconcileInfrastructure()` (gỡ/hủy).
  - Bước 3: Application Repository (lưu và đọc theo phiên bản), Infrastructure Reconciler (gỡ/hủy).
- Các artifact khác: sequence UC-01, UC-03; VOPC; domain model (Application Definition có phiên bản, Deployment trỏ phiên bản); ERD (bảng phiên bản, identity logic); operation contracts 1, 3, 4, 5; state machine Deployment (nếu cần); traceability.

**Quyết định bổ sung (chốt khi lập plan cho lượt sửa chung):**

10. **ID cố định qua phiên bản:** Workload, Resource Requirement, Environment Variable Definition và Secret Definition có ID logic không đổi qua các phiên bản; đổi tên vẫn giữ ID. Cấu hình UC-02, Resource Instance, Workload Instance và Workload Deployment tham chiếu ID logic này.
11. **Lưu phiên bản bằng bảng phiên bản + dòng con:** bảng `application_definition_version` cùng các dòng `workload`, `resource_requirement`, `environment_variable_definition`, `secret_definition`, `dependency` theo từng phiên bản; ID logic nằm ở identity table `application_component`; giữ khóa ngoại đầy đủ (không lưu phiên bản dạng khối JSON).

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (đặc tả UC-01 đến UC-04, Bước 1–3), sequence UC-01 và UC-03, VOPC (Application Repository theo phiên bản), domain model (Application Definition Version, Deployment trỏ phiên bản), ERD (`application_definition_version`, `application_component`, bảng theo phiên bản, `environment` ENUM, `removed_components`), contracts 1–5 và 8–9 (phiên bản, gỡ workload), state machines (`DESTROYED`, `UNLINKED`, `REMOVED`), traceability.

## Vấn đề 6 — Plan của UC-03 chưa có cấu trúc rõ ràng

**Vấn đề:** plan của UC-03 (dùng để hiển thị cho Developer, cho override tham số, và tính fingerprint phát hiện thay đổi giữa lúc lập plan và lúc xác nhận) được nhắc ở sequence, VOPC, contract 4–5 và schema, nhưng không có class trong domain model: VOPC ghi `-plan: Object`, override là `Map` không kiểu, contract 4 chỉ liệt kê trường đưa vào fingerprint bằng một đoạn văn. Hệ quả là fingerprint có thể báo `PLAN_CHANGED` giả hoặc bỏ sót thay đổi thật, và không rõ override gắn vào resource nào. Sau vấn đề 2, 3, 5, plan còn phải chứa phiên bản định nghĩa, các tầng, workload, hành động `REMOVE`/`DESTROY` và resource dùng chung.

**Quyết định:** xem xét sau. Vấn đề chỉ gây hại khi dữ liệu đầu vào của plan bị thay đổi giữa lúc lập plan và lúc xác nhận (ví dụ người khác sửa cấu hình hoặc định nghĩa trong lúc đó); chưa cần lo ở giai đoạn hiện tại.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D3 — gồm bảng các nơi nhắc tới plan, các tình huống gây hại, hướng đã đề xuất (Deployment Plan có cấu trúc theo tầng) và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** use case và sequence UC-03 vẫn mô tả nội dung plan ở mức khái niệm (tầng, action create/update/reuse/remove/destroy, override được phép) để thể hiện quyết định của vấn đề 2, 3, 5; chỉ phần cấu trúc chi tiết và cách tính fingerprint là để sau.

**Đã áp dụng:** `deferred_issues.md` (D3). Chưa sửa file thiết kế nào.

## Vấn đề 7 — Ai ghi các bước tiến trình của deployment

**Vấn đề:** UC-04 hiển thị tiến trình bằng các dấu kiểm Infrastructure Ready, Configuration Resolved, Manifest Generated, CD Synced, Application Ready — mỗi dấu kiểm là một dòng `deployment_step`. Nhưng trong sequence UC-03, `deployment_step` chỉ được ghi một lần trong `saveDeploymentRecord` ở bước cuối (sau khi gửi sang CD), nên:

- Khi deployment đang chạy, UC-04 không thấy bước nào.
- `CD Synced` và `Application Ready` xảy ra sau `saveDeploymentRecord`, mà `docs/architecture/state-machines/README.md` ghi rõ không có operation nào sau đó, còn UC-04 chỉ đọc — nên hai dấu kiểm này không bao giờ được ghi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D4 — gồm hướng đã đề xuất (Deployment Orchestrator ghi từng bước ngay khi bắt đầu/kết thúc, tập bước theo loại thành phần, tạo sẵn `PENDING`/`SKIPPED`), ví dụ và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** quyết định 10 của vấn đề 2 vẫn được thể hiện ở mức khái niệm (tiến trình theo tầng/thành phần được ghi lại, UC-04 đọc được); chi tiết ai ghi và ghi lúc nào để lại cho D4. Vấn đề này liên quan D2 (UC-04 không nên tự suy tiến trình từ Kubernetes).

**Đã áp dụng:** `deferred_issues.md` (D4). Chưa sửa file thiết kế nào.

## Vấn đề 8 — Trạng thái deployment bị trộn với trạng thái của hệ thống CD

**Vấn đề:** Contract 9 (`saveDeploymentRecord`) quy định `deployment_record.status` phản ánh delivery status mà CD báo về, rồi `deployment.status` được cập nhật đồng nhất với record. Tức là CD báo gì (ví dụ `Synced`, `Progressing`, `OutOfSync` của Argo CD) thì trạng thái vòng đời của deployment mang theo giá trị đó. Hệ quả:

- `deployment.status` có thể nhận giá trị không nằm trong state machine, làm các guard dựa trên status mất ý nghĩa.
- Phá quy tắc không phụ thuộc trực tiếp vào Argo CD/Flux.
- `deployment.status` và `deployment_record.status` lưu cùng một thứ ở hai nơi, dễ lệch nhau.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D5 — gồm hướng đã đề xuất (status chỉ là lifecycle của IDP; bỏ `deployment_record.status`; trạng thái CD lưu riêng với tập giá trị trung lập) và câu hỏi cần chốt.

**Lưu ý khi sửa tài liệu:** vấn đề 2 đã chốt `deployment.status` là `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED` do IDP tự quyết; khi sửa tài liệu không được mô tả status nhận giá trị từ CD. Liên quan D4 (bước `CD_SYNCED`) và D2 (trạng thái CD ở UC-04).

**Đã áp dụng:** `deferred_issues.md` (D5). Chưa sửa file thiết kế nào.

## Vấn đề 9 — Các giá trị ENUM chưa được chốt

**Vấn đề:** `docs/architecture/database/schema.md` có ba cột ENUM đã ghi giá trị (`dependency.target_type`, `configuration_value.value_source`, `secret.value_source`), nhưng bốn cột trạng thái chỉ ghi `ENUM` mà không có giá trị: `resource_instance.status`, `deployment.status`, `deployment_record.status`, `deployment_step.status`. Chính `operation_contracts.md` (dòng 3) và `docs/architecture/state-machines/README.md` (dòng 5) ghi nhận "chưa chốt tập literal vật lý". Hệ quả:

- Contract và state machine dùng tên "logical state", còn DB không nói giá trị thật; khi code mỗi người tự đặt tên (`READY`, `Ready`, `AVAILABLE`…).
- Guard và truy vấn dựa trên giá trị cụ thể (ví dụ `status = 'AWAITING_CONFIRMATION'`, chỉ tìm Resource Instance `READY`) sẽ không khớp khi giá trị không thống nhất.

**Quyết định:**

1. **Một bảng danh mục ENUM duy nhất trong `schema.md`** liệt kê mọi cột ENUM và tập giá trị; contract, state machine, domain model dùng đúng các giá trị trong bảng này.
2. **Quy ước tên:** `UPPER_SNAKE_CASE`, như các ENUM đang có.
3. **`environment` là ENUM** với hai giá trị cố định (theo vấn đề 5).
4. **Gỡ xong thì giữ dòng Instance với trạng thái kết thúc** (`DESTROYED`, `UNLINKED`, `REMOVED`) thay vì xóa dòng, vì deployment record cũ vẫn trỏ tới Instance (tránh lỗi lịch sử trỏ vào chỗ trống của vấn đề 5).
5. **Danh mục chia hai phần:** "Đã chốt" và "Dự kiến, chưa chốt" (ghi rõ chốt khi giải quyết D3/D4/D5), để không nhầm giá trị dự kiến là đã chốt.

**ENUM đã chốt:**

| ENUM | Giá trị | Nguồn |
|---|---|---|
| `dependency.target_type` | `WORKLOAD`, `RESOURCE` | Có sẵn |
| `configuration_value.value_source` | `DIRECT`, `RESOURCE_OUTPUT`, `WORKLOAD_OUTPUT` | Có sẵn |
| `secret.value_source` | `SECRET_REF`, `RESOURCE_OUTPUT` | Có sẵn |
| `deployment.status` | `AWAITING_CONFIRMATION`, `CONFIRMED`, `DEPLOYING`, `SUCCEEDED`, `FAILED` | Vấn đề 2 |
| `resource_instance.status` | `PLANNED`, `PROVISIONING`, `READY`, `FAILED`, `DESTROYED`, `UNLINKED` | State machine + vấn đề 3, 5 |
| `workload_instance.status` | `DEPLOYING`, `HEALTHY`, `FAILED`, `REMOVED` | Vấn đề 2, 5 |
| `environment` | `STAGING`, `PRODUCTION` | Vấn đề 5 |
| `workload_deployment.inclusion_reason` | `SELECTED`, `CASCADED` | Vấn đề 2 |
| Loại quản lý của `resource_definition` | `MANAGED` (IDP tạo/sửa/hủy), `EXISTING` (trỏ tới resource có sẵn, chỉ đọc) | Vấn đề 3 |

`UNLINKED` dành cho Resource Instance trỏ tới resource dùng chung: app thôi dùng thì chỉ gỡ liên kết, hạ tầng thật vẫn còn.

**ENUM dự kiến, chưa chốt:**

| ENUM | Giá trị dự kiến | Chốt khi |
|---|---|---|
| Loại thành phần trong plan | `RESOURCE`, `WORKLOAD` | D3 |
| Action cho resource | `CREATE`, `UPDATE`, `REUSE`, `DESTROY`, `UNLINK` | D3 |
| Action cho workload | `DEPLOY`, `REMOVE` | D3 |
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` | D4 |
| `deployment_step.step_name` (đổi từ VARCHAR sang ENUM) | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, `CD_SYNCED`, `APPLICATION_READY`, `REMOVED`, `DESTROYED`, `UNLINKED` | D4 |
| `deployment_record.status` | Bỏ cột, dùng `deployment.status` | D5 |
| `delivery_status` (trường riêng, nếu lưu) | `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED` | D5 |

**Sẽ ảnh hưởng:** `schema.md` (bảng danh mục ENUM, các cột status), `erd.puml`, domain model, operation contracts (dòng 3 và các status được nhắc), `docs/architecture/state-machines/README.md` (dòng 5) cùng các state machine, traceability.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** mục **Danh mục ENUM** trong `docs/architecture/database/schema.md` (đã chốt / dự kiến, thêm `application_component.component_type` và `deployment_execution_job.status` dự kiến theo D6), các cột status trong `schema.md`/`erd.puml`, domain model, đoạn mở đầu operation contracts, `docs/architecture/state-machines/README.md` cùng ba state machine; giá trị dự kiến đã ghi vào `deferred_issues.md` các mục D3, D4, D5, D6.

## Vấn đề 10 — Toàn bộ việc triển khai chạy trong request `confirmDeployment`

**Vấn đề:** trong `docs/use-cases/UC-03/sequence.puml`, sau khi nhận `confirmDeployment`, toàn bộ chuỗi `reconcileInfrastructure` → `collectResourceOutputs` → `resolveEnvironmentConfiguration` → sinh/adapt/materialize manifest → `publishDesiredDeploymentState` → `saveDeploymentRecord` chạy trong cùng request; trình duyệt chờ tới khi tất cả xong mới nhận "Deployment created". Hệ quả:

- **Request quá lâu bị cắt:** tạo database có thể mất 10–20 phút, trong khi trình duyệt/load balancer/API gateway thường cắt kết nối sau khoảng 30–60 giây; UI báo lỗi dù việc có thể vẫn chạy hoặc đã dừng giữa chừng.
- **Server khởi động lại giữa chừng thì không ai làm tiếp:** việc chạy trong bộ nhớ của request, deployment kẹt ở `CONFIRMED` hoặc nửa chừng, không ai tiếp tục hay đánh dấu thất bại.
- **Không xem được tiến trình** vì kết quả chỉ trả về khi xong (liên quan D4).
- **Vấn đề 2 làm việc này nặng hơn:** worker phải chờ pod healthy qua nhiều tầng và có thể tự deploy lại thành phần phụ thuộc; một lần deploy có thể kéo dài hàng chục phút.

**Quyết định:**

1. **`confirmDeployment` chỉ nhận việc và trả lời ngay:** kiểm tra như hiện tại (status `AWAITING_CONFIRMATION`, override hợp lệ); rồi trong **cùng một transaction DB** đổi `deployment.status` thành `CONFIRMED` **và** tạo một job "thực thi deployment". Không bao giờ có deployment `CONFIRMED` mà không có job, hay job mà deployment chưa được xác nhận. UI nhận "Đã nhận, đang triển khai" và chuyển sang theo dõi ở UC-04.
2. **Deployment Worker chạy nền:** một tiến trình riêng lấy job và chạy toàn bộ chuỗi theo tầng của vấn đề 2; bắt đầu thì đổi status sang `DEPLOYING`, xong thì `SUCCEEDED` hoặc `FAILED`.
3. **Job lưu trong bảng DB**, không dùng message queue riêng: đơn giản, job còn nguyên khi server restart, và tránh trường hợp DB đã lưu mà queue chưa nhận (hoặc ngược lại).
4. **Phục hồi khi worker chết giữa chừng:** hoãn — xem `deferred_issues.md`, mục D6.

**Hệ quả cần xử lý khi sửa tài liệu:**

- Thêm bảng job (ví dụ `deployment_execution_job`) gắn với deployment.
- Giá trị override mà Developer chọn khi xác nhận phải được **lưu cùng job**, vì worker chạy sau, không còn giữ request xác nhận.
- Sequence UC-03 tách thành hai phần: request xác nhận (dừng ở tạo job, trả lời UI) và luồng của Deployment Worker.
- Thêm thành phần **Deployment Worker** (UC-03 Bước 3, VOPC); Deployment Orchestrator ở phía request chỉ còn tạo và xác nhận deployment.
- Contract 5 (`confirmDeployment`) đổi hậu điều kiện: không còn gọi `reconcileInfrastructure` trực tiếp, mà tạo job cùng transaction với việc đổi status.
- State machine Deployment: `CONFIRMED` nghĩa là đã xác nhận và job đã được tạo; worker chuyển sang `DEPLOYING`.
- Trạng thái của job cần có giá trị ENUM; giá trị dự kiến ghi ở D6 vì phụ thuộc cách phục hồi.

**Sẽ ảnh hưởng:**

- `usecase_realization_step_1_3.md`:
  - Đặc tả UC-03: sau khi Developer chọn Deploy, IDP xác nhận và trả lời ngay; việc triển khai chạy nền và được theo dõi ở UC-04.
  - Bước 1 UC-03: trách nhiệm tách thành nhận việc (xác nhận, tạo job) và làm việc (Deployment Worker thực thi).
  - Bước 2 UC-03: `confirmDeployment()` chỉ đổi status và tạo job; các operation thực thi do Deployment Worker chạy.
  - Bước 3 UC-03: thêm Deployment Worker; Deployment Orchestrator chỉ còn tạo và xác nhận deployment; Deployment Repository lưu job.
- Các artifact khác: sequence UC-03, VOPC UC-03 và design class diagram, domain model và persistence classification (job), ERD (bảng job), operation contracts 5–9, state machine Deployment, traceability.

**Đã áp dụng (lượt sửa chung, nhánh `refine_design`):** use case realization (đặc tả UC-03, Bước 1–3 tách nhận yêu cầu và thực thi nền), sequence UC-03 (request trả lời ngay; luồng Deployment Worker), VOPC (Deployment Worker; `confirmDeploymentAndCreateJob`, `claimNextExecutionJob`), domain model và persistence classification (Deployment Execution Job), ERD (`deployment_execution_job`), contracts 5–9, state machine Deployment (`CONFIRMED` có job → `DEPLOYING`), traceability; phần hoãn ghi vào `deferred_issues.md` (D6).

## Vấn đề 11 — Secret bị bỏ rơi trong Secret Store

**Vấn đề:** trong sequence UC-02, secret được ghi vào Secret Store ngay lúc Developer nhập (`storeSecret`), còn secret reference chỉ được lưu vào DB khi bấm Save. Mọi trường hợp không đi tới được bước lưu DB đều để lại secret không ai trỏ tới: Developer đóng tab không Save, validate thất bại rồi bỏ đi, lưu DB lỗi, nhập lại secret nhiều lần, hoặc đổi secret ở lần cấu hình sau (bản cũ vẫn còn). Hậu quả là secret thật tồn tại mà không ai quản lý hay thu hồi (rủi ro bảo mật), rác tích tụ theo thời gian, và contract 3 không nói gì về việc dọn dẹp.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

**Chi tiết:** xem `06_traceability/deferred_issues.md`, mục D7 — gồm bảng các tình huống sinh secret orphan, ba hướng có thể cân nhắc (ghi lúc Save kèm xóa bù; lưu tạm có hạn dùng; dọn rác định kỳ) và câu hỏi cần chốt.

**Lưu ý:** vấn đề này gắn với D1 (bản nháp — secret nằm ở đâu trước khi Save); nên cân nhắc giải quyết cùng lúc. Khi sửa tài liệu cho các vấn đề đã chốt, giữ nguyên luồng lưu secret hiện tại của UC-02.

**Đã áp dụng:** `deferred_issues.md` (D7). Chưa sửa file thiết kế nào.

## Vấn đề 12 — Nơi triển khai và hạ tầng của nó; phiên bản catalog

**Vấn đề:** ở UC-03, Developer chọn deployment target và IDP "reconcile infrastructure". Nhưng thiết kế chưa nói có những loại nơi triển khai nào, cụm Kubernetes lấy từ đâu, và VPC mà cụm cần được dựng ở bước nào.

Ví dụ: deploy `shop-app` (frontend, backend, PostgreSQL) lên AWS. IDP phải dựng VPC, rồi EKS và Aurora trên VPC, rồi mới chạy backend và frontend trên EKS. Khi làm theo hướng này, thiết kế còn thiếu 4 chỗ:

1. Đồ thị deploy chỉ có thứ Developer khai báo (frontend, backend, postgresql). Không có chỗ cho cụm, VPC, và quan hệ "Aurora cần VPC".
2. Phạm vi deploy chỉ gồm workload được chọn và resource chúng dùng trực tiếp. Không workload nào dùng trực tiếp VPC, nên VPC không bao giờ được tạo.
3. Khi output thay đổi, IDP chỉ deploy lại workload. VPC đổi subnet thì EKS và Aurora không được cập nhật theo. Cụm bị dựng lại thì app không được đưa lên cụm mới.
4. Platform sửa catalog (vd đổi cấu hình VPC) thì lần deploy sau của mọi app đều nhận thay đổi đó, kể cả khi Developer chỉ deploy lại frontend.

**Quyết định:**

1. **Có hai loại nơi triển khai**; Developer chọn một khi deploy.
   - **Cloud** (vd AWS, region ap-southeast-1): IDP dựng VPC và cụm (vd EKS) riêng cho mỗi app + environment, và xóa chúng khi gỡ app.
   - **Cụm Kubernetes nội bộ:** cụm đã có sẵn. IDP chỉ kết nối vào để deploy, không tạo, không xóa cụm. Platform có thể khai báo nhiều cụm nội bộ (vd một cho staging, một cho production); catalog quyết định app và environment nào dùng cụm nào.
2. **Cụm và VPC nằm trong đồ thị deploy** như resource khác; Developer không khai báo chúng. IDP tự thêm:
   - App nào cũng cần một cụm (`k8s-cluster`).
   - Công thức trong catalog ghi thêm nó cần gì (`requires`), vd EKS và Aurora cần `network`. Công thức cần gì là do platform team quyết định; IDP làm theo.
   - Trên cloud, công thức `k8s-cluster` là `MANAGED`: IDP tạo, sửa, xóa.
   - Trên cụm nội bộ, công thức `k8s-cluster` là `EXISTING`: trỏ tới cụm có sẵn, IDP chỉ liên kết.
3. **Database và các resource khác:** IDP vẫn tạo cho mỗi app + environment, kể cả trên cụm nội bộ (vd Postgres chạy trong cụm). Muốn dùng database có sẵn thì platform khai báo công thức `EXISTING`.
4. **Phạm vi deploy** = workload được chọn + resource chúng dùng trực tiếp + cụm/VPC mà các resource đó cần. Thứ gì không đổi thì dùng lại, không chạy lại Terraform.
5. **Dưới đổi thì trên làm lại:** output của một thành phần đổi thì mọi resource và workload dựa trên nó được làm lại trong cùng lần deploy, kể cả khi nằm ngoài phạm vi. Plan hiện trước những thứ có thể bị làm lại.
6. **Catalog có phiên bản.** Sửa catalog là tạo phiên bản mới; phiên bản cũ giữ nguyên. Developer chọn phiên bản catalog khi deploy, nên app đang chạy không bị ảnh hưởng cho tới khi Developer chọn phiên bản mới.
   - Deploy một phần phải dùng phiên bản catalog đang chạy. Muốn đổi phiên bản thì phải deploy toàn bộ app.
   - Được chọn phiên bản cũ hơn; plan vẫn kiểm tra như bình thường.
   - Form chọn sẵn phiên bản đang chạy và báo nếu có bản mới hơn. Lần deploy đầu thì chọn sẵn bản mới nhất.
   - Promote staging → production dùng cả phiên bản app lẫn phiên bản catalog của staging.

**Hoãn** (ghi vào `deferred_issues.md`):

- **D10:** platform khóa hoặc ngừng hỗ trợ phiên bản catalog cũ.
- **D11:** phiên bản catalog mới đổi hẳn công thức của một resource đang chạy (vd Postgres trong cụm → database dùng chung): báo lỗi hay tự dựng lại.
- **D12:** UC-02 lấy danh sách output từ phiên bản catalog nào, khi cấu hình không gắn với phiên bản catalog.

**Sẽ sửa:** use case realization (đặc tả UC-03, UC-04, Bước 1–3), sequence UC-03 và UC-04, VOPC, domain model, ERD, operation contracts 4–6 và 11, state machine Resource Instance, traceability. Contract 3 (UC-02) chưa sửa vì phụ thuộc D12.

**Đã áp dụng (nhánh `uc03-impl`, 15/09/2026):** use case realization (đặc tả UC-03, UC-04; Bước 1–3 UC-03), sequence UC-03 và UC-04, VOPC (`vopc_uc03`, design class diagram, README), domain model, domain objects, persistence classification, ERD (`catalog_version`, `resource_definition`, `application_component`, `resource_instance`, `deployment`), contracts 4, 5, 6, 11, state machine Resource Instance và ghi chú Deployment, traceability; phần hoãn ghi vào D10, D11, D12. Code trong `uc03/` chưa sửa theo quyết định này.

## Vấn đề 13 — Mỗi application một nơi chứa desired state

**Vấn đề:** thiết kế chỉ nói IDP "publish desired deployment state tới CD abstraction", không nói desired state nằm ở đâu. Bản cài đặt đặt mọi application vào **một** repo Git dùng chung, phân tách bằng thư mục `<nơi triển khai>/<app>-<environment>/workloads/…`, và mọi cụm dùng chung một cặp khóa. Hệ quả:

- Khóa ghi của IDP và khóa đọc nạp vào Argo CD của một cụm mở được desired state của **mọi** application, kể cả app không deploy lên cụm đó.
- Lịch sử commit của các application trộn lẫn; không phân quyền hay audit theo application được.

**Quyết định:**

1. **Mỗi application có một Delivery Repository riêng** — nơi chứa desired state của application đó. Trong repo vẫn chia theo nơi triển khai và environment như cũ (`<target>/<app>-<env>/workloads/…`), vì một application dùng chung một repo cho mọi environment.
2. **IDP tự tạo repo ở lần deploy đầu tiên của application.** Platform không phải làm thủ công cho từng app. Tên repo suy ra từ mẫu platform cấu hình, ví dụ `<tổ chức>/idp-<app>-gitops`. Repo đã tồn tại đúng tên thì dùng lại, không báo lỗi.
3. **IDP sinh một cặp khóa riêng cho từng application** khi tạo repo: khóa ghi để IDP đẩy manifest, khóa đọc nạp vào Argo CD của cụm dưới dạng thông tin truy cập của riêng repo đó. Nhờ vậy khóa của application này không mở được repo của application khác. Khóa lưu trong Secret Store; database, log và Deployment Record chỉ thấy reference.
4. **Thông tin gọi hệ thống lưu trữ Git** (token của tài khoản máy, quyền tạo repo và gắn deploy key) nằm trong Secret Store, chỉ Deployment Worker đọc.
5. **Việc đăng ký application → repo được lưu bền vững** (URL, nhánh). IDP ghi khi tự tạo; platform cũng đăng ký tay được một repo có sẵn.
6. **Gỡ application khỏi một environment/nơi triển khai** chỉ xóa phần desired state của environment đó trong repo. Repo và khóa giữ lại để còn lịch sử.
7. **Tạo repo, sinh khóa hoặc đăng ký thất bại** → deployment FAILED ở bước giao hàng (A2), không publish sang repo nào khác.

**Hệ quả cần xử lý khi sửa tài liệu:**

- Thêm domain object **Delivery Repository** (thuộc Application) và bảng tương ứng; thêm thành phần lưu trữ để đọc/ghi đăng ký.
- Thêm abstraction cho hệ thống lưu trữ Git (tạo repo, gắn khóa) và implementation cụ thể; UC-03 vẫn không phụ thuộc vào một sản phẩm cụ thể.
- Contract 8 mô tả việc bảo đảm repo của application tồn tại trước khi publish; A2 thêm trường hợp tạo repo thất bại.
- Đặc tả UC-03 thêm quy tắc: desired state của mỗi application nằm ở repo riêng, application này không ghi được vào repo của application khác.

**Sẽ ảnh hưởng:** `usecase_realization_step_1_3.md` (đặc tả UC-03, Bước 1–3), sequence UC-03, VOPC UC-03 + design class diagram + README, domain model + domain objects + persistence classification, ERD, operation contract 8, traceability.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** toàn bộ danh sách trên. Code trong `uc03/` chưa sửa theo quyết định này.

## Vấn đề 14 — Đổi hệ thống CD sang Fleet

**Bối cảnh:** Bản cài đặt đang dùng Argo CD. Người dùng muốn chuyển sang Fleet (Rancher Fleet).

**Điều đáng chú ý nhất: kiến trúc không phải sửa.** Từ đầu, UC-03 chỉ nói chuyện với **CD Integration / CD Provider Interface**; tên sản phẩm chỉ xuất hiện trong tài liệu dưới dạng ví dụ, và business rule đã ghi rõ "Argo CD chỉ nằm trong adapter". Vì vậy sequence UC-03, VOPC, design class diagram, domain model, ERD, operation contract và traceability **không đổi một dòng nào** khi thay CD system. Đây là lần đầu lớp trừu tượng đó được kiểm chứng bằng một sản phẩm thứ hai chạy thật, chứ không chỉ là tuyên bố trên giấy.

**Quyết định:**

1. **Giữ cả hai adapter**, chọn bằng cấu hình (`IDP_CD_PROVIDER`), mặc định Fleet. Lý do: chứng minh được abstraction thay thế được, và hai application có thể chạy hai CD system khác nhau trên cùng một cụm.
2. **Delivery reference không đổi**: vẫn là commit SHA, vì Fleet cũng báo commit đang triển khai (`GitRepo.status.commit`). Không có thay đổi schema nào.
3. **Trạng thái CD vẫn được quy về tập trung lập** của IDP (`SYNCED`, `SYNCING`, `OUT_OF_SYNC`, `HEALTHY`, `PROGRESSING`, `DEGRADED`, `MISSING`); adapter Fleet chuyển đổi từ `status.summary` và condition `Ready`, không để giá trị riêng của Fleet lọt ra ngoài adapter.
4. **Phần Git là chung, phần cụm là riêng.** Việc bảo đảm delivery repository, đẩy desired state và lấy commit SHA giống hệt nhau ở cả hai CD system, nên được tách thành phần dùng chung; mỗi adapter chỉ khác ở đối tượng tạo trong cụm và cách đọc trạng thái.
5. **Phạm vi đợt này là target `kind-local`.** Module `eks-cluster` của target `aws` vẫn cài Argo CD; chưa chạy Fleet trên AWS.

**Khác biệt cụ thể giữa hai CD system** (để người đọc tài liệu không phải tra lại):

| Việc | Argo CD | Fleet |
|---|---|---|
| Đối tượng khai báo | `Application` trong `argocd` | `GitRepo` (`fleet.cattle.io/v1alpha1`) trong `fleet-local` |
| Khóa đọc repo | Secret nhãn `argocd.argoproj.io/secret-type: repository` | Secret kiểu `kubernetes.io/ssh-auth`, cùng namespace với `GitRepo`, trỏ bằng `clientSecretName` |
| Đã sync tới commit nào | `status.sync.revision` | `status.commit` |
| Sức khỏe | `status.health.status` | condition `Ready` + `status.summary` |
| Xóa resource khi file biến mất | `syncPolicy.automated.prune` | `keepResources` (mặc định tắt nên có xóa) |

**Sẽ ảnh hưởng:** chỉ các dòng ví dụ trong `usecase_realization_step_1_3.md`, `docs/architecture/design-classes.md`, `06_traceability/deferred_issues.md`; phần còn lại là code và tài liệu vận hành.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** tài liệu và code đều xong, kiểm chứng thật trên cụm nội bộ (`uc03/docs/VERIFICATION.md` §6). Hai application chạy hai CD system khác nhau trên cùng một cụm.

**Một hệ quả phát hiện khi chạy thật:** manifest do IDP sinh ra từng đánh dấu bằng nhãn chuẩn `app.kubernetes.io/managed-by: idp`. Fleet triển khai bundle qua Helm, mà Helm luôn đặt nhãn đó thành `Helm`, nên object trong cụm lệch Git vĩnh viễn và Fleet không bao giờ báo xong. Bài học chung, không riêng Fleet: **IDP chỉ được đánh dấu bằng nhãn thuộc không gian tên của mình** (`idp.dev/managed-by`), vì nhãn `app.kubernetes.io/managed-by` thuộc về công cụ trực tiếp áp manifest xuống cụm, và công cụ đó thay đổi theo CD system.

## Vấn đề 15 — Gỡ application khỏi một environment trở thành UC-05

**Vấn đề:** Thiết kế mô tả việc gỡ bỏ **chỉ như hệ quả của việc deploy một phiên bản mới**: phiên bản mới bỏ workload nào thì workload đó bị gỡ, resource không còn được dùng thì bị hủy hoặc gỡ liên kết. Không có chỗ nào cho tình huống Developer muốn gỡ **toàn bộ** application khỏi một environment mà không deploy gì cả.

Tệ hơn, sau vấn đề 13 tài liệu có hai câu nói về "gỡ application khỏi một environment" (trong đặc tả UC-03 và trong mô tả Delivery Repository) — tức là **tài liệu tham chiếu tới một thao tác mà chính nó không định nghĩa ở đâu**. Người đọc đi tìm sẽ không thấy luồng, không thấy contract, không thấy trạng thái.

Trong khi đó code đã chạy thao tác này nhiều tháng: `POST /api/teardowns`, nút trên UI, `deployment.kind = TEARDOWN`.

**Quyết định:**

1. **Tách thành use case riêng, UC-05**, không nhét vào UC-03. Lý do: UC-03 là "triển khai một phiên bản" — nó nhận phiên bản, phiên bản catalog và image. UC-05 không nhận thứ nào trong ba thứ đó; nó lấy lại đúng những gì đang chạy. Hai use case có mục tiêu, tiền điều kiện và hậu điều kiện khác hẳn nhau.
2. **Phạm vi là một environment + một nơi triển khai.** Xóa hẳn application khỏi IDP (Application Definition, các phiên bản, Environment Configuration, Delivery Repository) **không** thuộc UC-05 và hiện chưa có trong hệ thống.
3. **Dùng lại nguyên vẹn phần sau của UC-03**: cùng `confirmDeployment()`, cùng job, cùng Deployment Worker, cùng Deployment Record, cùng state machine. Chỉ thêm `createTeardown()` và một operation thực thi cho các tầng gỡ.
4. **Phân biệt trong dữ liệu bằng `deployment.kind`** (`DEPLOY`, `TEARDOWN`). Deployment loại gỡ bỏ không có `workload_deployment` nào.
5. **Thứ tự là ngược lại**: workload trước, rồi tới resource mà chúng phụ thuộc. Workload chỉ được đánh dấu đã gỡ sau khi xác minh nó biến mất khỏi cụm; resource chỉ bị hủy sau đó. Resource dùng chung, có sẵn chỉ bị gỡ liên kết, không bao giờ bị hủy.
6. **Không thêm class nào** vào VOPC: UC-05 dùng lại đúng các thành phần của UC-03.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** đặc tả UC-05 trong `usecase_realization_step_1_3.md` (kèm Bước 1, 2, 3), `docs/use-cases/UC-05/sequence.puml`, `docs/use-cases/UC-05/vopc.puml`, cột `deployment.kind` trong ERD, operation contract 12 `createTeardown()`, nhánh mới trong state machine deployment, traceability. Code trong `uc03/` đã có sẵn thao tác này nên lần này thiết kế đuổi theo code, ngược với các vấn đề trước.
