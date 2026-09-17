---
id: LEGACY-DEFERRED-ISSUE-LOG
artifact: consolidated-deferred-issue-log
status: historical
current_index: ../docs/backlog/README.md
last_reviewed: 2026-09-17
---

> **Historical consolidated record.** Use the split [backlog index](../docs/backlog/README.md) for current issue navigation and status. This file is retained for provenance.

# Các vấn đề thiết kế hoãn sau MVP

Tài liệu này ghi các vấn đề thiết kế đã được nhận diện nhưng **chưa cần giải quyết ở MVP**. Mỗi mục giữ đủ bối cảnh để sau này quay lại xử lý mà không phải dò lại từ đầu.

Nguồn đối chiếu: commit `88585ccdd1a7896c615ad43fbb0d0f6b62d231ef` (nhánh `draft`, "Vá 11 vấn đề thiết kế + 2 finding review") — commit này đề xuất cách sửa cho 11 vấn đề, nhưng các đề xuất đó chưa được xác nhận là đúng; chỉ dùng để tham khảo. Nhánh `refine_design` bắt đầu từ `11ad58249f58e06108ab86da787223254be9b05f`, tức là **trước** commit đó. Các vấn đề đã chốt (2, 3, 5, 9, 10) đã được áp dụng vào tài liệu thiết kế trên nhánh này; các mục dưới đây là phần còn hoãn (xem `design_decisions.md`).

## Danh sách theo dõi

| ID | Vấn đề | Trạng thái | Ngày ghi nhận |
|---|---|---|---|
| D1 | Bản nháp UC-01/UC-02 được giữ ở đâu giữa các request | DEFERRED_MVP | 14/09/2026 |
| D2 | UC-04 gọi hệ thống bên ngoài mà không kiểm tra điều kiện | DEFERRED | 14/09/2026 |
| D3 | Deployment/Infrastructure Plan chưa có cấu trúc rõ ràng; fingerprint không đáng tin khi dữ liệu thay đổi giữa lúc lập plan và lúc xác nhận | DEFERRED | 14/09/2026 |
| D4 | Chưa rõ ai ghi các bước tiến trình (`deployment_step`) của deployment và ghi vào lúc nào | DEFERRED | 14/09/2026 |
| D5 | Trạng thái vòng đời của deployment bị trộn với trạng thái giao hàng của hệ thống CD | DEFERRED | 14/09/2026 |
| D6 | Phục hồi khi Deployment Worker chết giữa chừng (làm tiếp hay làm lại, chống chạy trùng, không tạo trùng hạ tầng) | DEFERRED | 14/09/2026 |
| D7 | Secret bị bỏ rơi (orphan) trong Secret Store khi cấu hình không được lưu hoặc secret bị thay | DEFERRED | 14/09/2026 |
| D8 | Cơ chế cụ thể để đọc Workload Output từ workload đã healthy hoặc đang chạy | DEFERRED | 14/09/2026 |
| D9 | Output của resource dùng chung thay đổi không lan sang application khác; chưa có resource riêng theo từng workload | DEFERRED | 14/09/2026 |
| D10 | Platform khóa hoặc ngừng hỗ trợ phiên bản catalog cũ | DEFERRED | 15/09/2026 |
| D11 | Phiên bản catalog mới đổi hẳn công thức của một resource đang chạy | DEFERRED | 15/09/2026 |
| D12 | UC-02 lấy danh sách output từ phiên bản catalog nào | DEFERRED | 15/09/2026 |
| D13 | Teardown đánh dấu workload đã gỡ dù chưa xác minh được cụm | DEFERRED | 16/09/2026 |
| D14 | Teardown không dọn CD object, desired state và namespace khi plan chỉ còn resource | DEFERRED | 16/09/2026 |

## D1 — Bản nháp UC-01/UC-02 được giữ ở đâu giữa các request

**Tương ứng:** issue #1 trong commit `88585cc`.

**Quyết định:** chưa giải quyết ở MVP, để lại xử lý sau.

### Vấn đề

Trong UC-01 và UC-02, Developer chỉnh sửa qua nhiều bước (thêm resource, workload, configuration requirement, dependency, gán value/output…) rồi mới Save. Thiết kế hiện tại đặt bản nháp làm thuộc tính của application service ở backend, và các operation chỉnh sửa chỉ nhận `applicationId`:

- `01_vopc_design_class_diagram/vopc_uc01.puml:29` — `Application Service` có `-applicationDraft: Object`.
- `01_vopc_design_class_diagram/vopc_uc02.puml:30` — `Environment Configuration Service` có `-configurationDraft: Object`.
- `sequence_digrams/uc_01_create_configure_application.puml:21,28,48` — Service trả về "draft" nhưng không nói draft được lưu và khôi phục thế nào giữa các request.
- `06_traceability/traceability_matrix.md` (các dòng UC-01/UC-02) — ghi "draft; write deferred" nhưng không có bảng hay nơi lưu draft.

Hệ quả nếu giữ nguyên:

- Không rõ draft nằm trong bộ nhớ hay DB; restart server có thể mất draft, mà schema không có bảng draft.
- Chạy nhiều instance backend thì các request chỉnh sửa và Save có thể rơi vào instance khác nhau.
- Một thuộc tính draft trên service không phân biệt được draft của từng người dùng/phiên.
- Draft bị bỏ dở không có cơ chế dọn.

### Một hướng sửa đã được đề xuất (`88585cc`, chưa xác nhận là đúng)

- Web UI sở hữu `ApplicationDefinitionDraft` / `ConfigurationDefinitionDraft`; service stateless giữa các request.
- Mỗi operation chỉnh sửa nhận **toàn bộ** draft và trả về draft mới, ví dụ `addWorkload(applicationDraft, workloads)`; UI thay draft cục bộ.
- `saveApplicationDefinition(applicationDraft)` / `saveEnvironmentConfiguration(configurationDraft)` validate rồi mới ghi DB.
- Persistence classification xếp draft là `CLIENT-OWNED DTO`, không có backend draft store.
- UC-02: secret nhập trực tiếp được gửi plaintext đúng một lần qua `stageSecret`, UI chỉ giữ opaque reference (liên quan issue #11).

Xem: `git show 88585cc -- 01_vopc_design_class_diagram/vopc_uc01.puml 01_vopc_design_class_diagram/vopc_uc02.puml sequence_digrams/uc_01_create_configure_application.puml`.

### Câu hỏi cần chốt khi giải quyết

1. **Có cần gọi backend cho mỗi thao tác chỉnh sửa không?** Nếu UI đã giữ draft, các thao tác thêm/sửa có thể làm hoàn toàn phía client và chỉ gọi backend khi Save (hoặc khi cần validate/tra catalog). Chỉ giữ round-trip nếu backend thực sự xử lý logic trung gian.
2. **Mất draft khi đóng tab/refresh:** chấp nhận và ghi rõ trong use case, hay lưu tạm phía client (ví dụ localStorage)?
3. **Hai người cùng sửa một application/configuration:** cần đặc tả optimistic locking (draft mang version, Save kiểm tra version) và luồng xử lý khi người Save sau bị conflict.

### Điều kiện đóng

Sequence, VOPC, domain/persistence classification, operation contracts và traceability thống nhất một nơi sở hữu draft; ba câu hỏi trên có quyết định rõ ràng.

## D2 — UC-04 gọi hệ thống bên ngoài mà không kiểm tra điều kiện

**Tương ứng:** issue #4 trong commit `88585cc`.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

UC-04 hiển thị kết quả một deployment bằng hai nguồn: dữ liệu đã lưu trong DB (Deployment Record, step, image) và dữ liệu hỏi trực tiếp hệ thống bên ngoài (hạ tầng, CD, Kubernetes). Trong `sequence_digrams/uc_04_view_deployment_result.puml:38-53`, khi mở **bất kỳ** deployment nào, IDP luôn gọi đủ bốn lời gọi, không kiểm tra deployment đó đã tới bước tương ứng hay chưa:

- `getInfrastructureStatus(infrastructureReferences)`
- `getCDStatus(deliveryReference)`
- `getWorkloadStatus(target, workloads)`
- `getDeploymentEndpoints(target, workloads)`

Hệ quả:

1. **Hỏi thứ không tồn tại.** Deployment thất bại ở bước tạo hạ tầng thì chưa từng được gửi sang CD; `deployment_record.delivery_reference` bằng `NULL` (`03_database_erd/schema.md` cho phép). IDP vẫn gọi `getCDStatus(NULL)`, dẫn tới lỗi hoặc kết quả vô nghĩa.
2. **Hiển thị "Healthy" giả — lỗi nguy hiểm nhất.** `getWorkloadStatus(target, workloads)` hỏi Kubernetes về workload **đang chạy** trên cluster, không phải workload **của deployment đang xem**. Ví dụ:

    ```text
    Deployment #41: backend v1.4.2 → thành công, đang chạy
    Deployment #42: backend v1.4.3 → thất bại ở bước tạo hạ tầng, chưa từng deploy
    ```

    Mở #42, màn hình hiện "backend Healthy" — thực ra là v1.4.2 của #41. Ngược lại, mở lại #41 sau khi một deployment mới hơn đã thành công, màn hình gán health của bản mới cho #41.
3. **Deployment chưa xác nhận.** Deployment ở `AWAITING_CONFIRMATION` chưa có Deployment Record, chưa có hạ tầng hay delivery; hỏi hạ tầng, CD, Kubernetes đều vô nghĩa.

Điều này mâu thuẫn với quy tắc của UC-04 "phải hiển thị image/version thực tế đã sử dụng trong deployment": trạng thái hiển thị có thể thuộc về một deployment khác.

### Liên quan tới vấn đề 2 (đã chốt)

Hai quyết định của vấn đề 2 tạo sẵn nền để giải quyết:

- Worker chờ pod healthy và lưu kết quả vào `deployment_step` theo tầng/thành phần (`CD_SYNCED`, `APPLICATION_READY`) → kết quả "deployment này có chạy được không" đã nằm trong DB.
- **Workload Instance** cho biết mỗi workload đang chạy bản của deployment nào → biết được deployment đang xem có còn là bản đang chạy hay đã bị thay thế.

### Hướng giải quyết đã đề xuất (chưa chốt)

**A. Kết quả của deployment luôn lấy từ DB**, không hỏi bên ngoài: status, tiến trình theo tầng/thành phần, image, lỗi. Đây là lịch sử, không bị lẫn với deployment khác.

**B. Trạng thái trực tiếp chỉ hỏi bên ngoài khi đủ điều kiện:**

| Lời gọi | Chỉ gọi khi | Nếu không đủ điều kiện, hiển thị |
|---|---|---|
| Trạng thái hạ tầng | Deployment có Resource Instance liên quan | "Chưa có hạ tầng" |
| Trạng thái CD | Có `delivery_reference` | "Chưa gửi sang CD" |
| Health và endpoint của workload | Workload Instance của workload đó đang trỏ tới chính deployment này | "Chưa triển khai tới workload này" hoặc "Đã được thay thế bởi deployment #N" |

UC-04 vẫn chỉ đọc.

### Câu hỏi cần chốt khi giải quyết

1. Có đồng ý tách **kết quả lấy từ DB** với **trạng thái trực tiếp chỉ hỏi khi đủ điều kiện** như trên không?
2. Với deployment **đã bị thay thế**: chỉ hiển thị "đã được thay thế bởi #N" (không hỏi Kubernetes), hay vẫn hiển thị trạng thái trực tiếp của bản đang chạy kèm nhãn "đây là bản của #N"?
3. Khi hệ thống bên ngoài không trả lời được (CD hoặc cluster tạm lỗi), màn hình hiển thị thế nào mà không che mất phần kết quả lấy từ DB?

### Điều kiện đóng

Sequence UC-04, VOPC UC-04, operation list Bước 2 của UC-04 và traceability thể hiện rõ điều kiện trước mỗi lời gọi ra ngoài; không kịch bản nào (thất bại trước CD, chưa xác nhận, đã bị thay thế) hiển thị trạng thái của một deployment khác.

## D3 — Plan của UC-03 chưa có cấu trúc rõ ràng

**Tương ứng:** issue #6 trong commit `88585cc` ("Infrastructure Plan chưa có typed model").

**Quyết định:** xem xét sau. Vấn đề chỉ gây hại khi dữ liệu đầu vào của plan bị thay đổi trong khoảng thời gian giữa lúc lập plan (`createDeployment`) và lúc Developer xác nhận (`confirmDeployment`), ví dụ có người khác sửa cấu hình hoặc định nghĩa trong lúc đó. Trường hợp này chưa cần lo ở giai đoạn hiện tại.

### Vấn đề

Trước khi đụng vào hạ tầng thật, UC-03 lập một plan để: hiển thị cho Developer những gì sắp làm; cho Developer override một số tham số được phép; và tính dấu vân tay (fingerprint). Lúc tạo deployment, IDP lưu fingerprint của plan; lúc Developer xác nhận, IDP dựng lại plan, tính lại fingerprint, nếu khác thì báo `PLAN_CHANGED`.

Plan được nhắc ở nhiều nơi nhưng không nơi nào định nghĩa nó gồm những gì:

| Nơi | Plan được mô tả thế nào |
|---|---|
| `sequence_digrams/uc_03_deploy_application.puml` | Chỉ là dòng chữ `Infrastructure plan (create/update/reuse)` |
| `01_vopc_design_class_diagram/vopc_uc03.puml`, `design_class_diagram.puml` | `Infrastructure Planner` có `-plan: Object`, `-allowedOverrides: Map` |
| `04_operation_contracts/operation_contracts.md` (Contract 4, 5) | Một đoạn văn liệt kê "những thứ đưa vào fingerprint" |
| `03_database_erd/schema.md` | Chỉ có cột `plan_fingerprint`, `plan_fingerprint_algo` |
| `02_domain_model/domain_model.puml` | Không có class; chỉ có ghi chú "Infrastructure Plan remains TRANSIENT" |
| `02_domain_model/persistence_classification.md` | Tự nhận bao phủ toàn bộ domain object nhưng không có dòng Infrastructure Plan |

Hệ quả:

1. **Fingerprint không đáng tin.** Fingerprint chỉ đúng khi lúc tạo và lúc xác nhận tính trên cùng một tập trường, cùng thứ tự. Không có cấu trúc thì có thể báo `PLAN_CHANGED` giả dù không có gì đổi, hoặc bỏ sót thay đổi thật.
2. **Không rõ UI hiển thị gì và override gắn vào đâu:** một override như "tăng dung lượng lên 100GB" thuộc resource nào trong plan, giới hạn cho phép lấy ở đâu.

### Khi nào vấn đề gây hại

Dữ liệu đầu vào của plan thay đổi giữa lúc lập plan và lúc xác nhận, ví dụ:

- Người khác lưu một phiên bản định nghĩa app hoặc sửa cấu hình environment.
- Platform sửa Resource Definition (tham số mặc định, override được phép).
- Một deployment khác làm thay đổi Resource Instance hoặc Workload Instance đang dùng.

### Plan đã lớn hơn sau các quyết định trước

Sau vấn đề 2, 3 và 5, plan không còn chỉ là "hạ tầng" mà cần chứa:

- phiên bản Application Definition được deploy (vấn đề 5);
- các tầng triển khai và workload nào ở tầng nào (vấn đề 2);
- danh sách thành phần có thể bị deploy lại khi output thay đổi (vấn đề 2);
- hành động gỡ/hủy `REMOVE`, `DESTROY` (vấn đề 5);
- resource riêng hay dùng chung (vấn đề 3).

### Hướng giải quyết đã đề xuất (chưa chốt)

Định nghĩa plan thành object có cấu trúc (vẫn TRANSIENT, chỉ lưu fingerprint), đổi tên thành **Deployment Plan** vì chứa cả workload:

```text
Deployment Plan
├─ applicationDefinitionVersion
├─ environment, deploymentTarget
├─ waves: danh sách Plan Wave
│    └─ Plan Wave
│         ├─ waveNumber
│         └─ items: danh sách Plan Item
│              └─ Plan Item
│                   ├─ component: RESOURCE hoặc WORKLOAD (identity logic)
│                   ├─ action:
│                   │    resource: CREATE | UPDATE | REUSE | DESTROY | UNLINK
│                   │    workload: DEPLOY | REMOVE
│                   ├─ (resource) resourceDefinition, resourceInstance, shared
│                   ├─ (resource) resolvedParameters
│                   ├─ (resource) overrideOptions: parameterKey, giá trị mặc định, giới hạn (enum / min-max)
│                   └─ (workload) imageVersion
└─ potentialRedeployComponents
```

Fingerprint là hash của toàn bộ plan theo thứ tự cố định, không bao gồm giá trị override Developer chọn (giá trị này chỉ gửi lúc xác nhận và được kiểm tra theo `overrideOptions`).

### Câu hỏi cần chốt khi giải quyết

1. Gộp thành một Deployment Plan chứa cả resource lẫn workload chia theo tầng, hay giữ Infrastructure Plan chỉ cho resource?
2. Fingerprint tính trên toàn bộ plan trừ giá trị override đã chọn?
3. Resource dùng chung: dùng `REUSE` kèm cờ `shared` và `UNLINK` khi gỡ (không hủy hạ tầng), hay action riêng?

### Lưu ý khi sửa tài liệu cho các vấn đề đã chốt

Dù hoãn phần cấu trúc chi tiết và fingerprint, use case và sequence UC-03 vẫn phải mô tả nội dung plan ở mức khái niệm (tầng, action create/update/reuse/remove/destroy, override được phép) để thể hiện đúng quyết định của vấn đề 2, 3 và 5.

### Giá trị ENUM dự kiến (từ vấn đề 9, chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| Loại thành phần trong plan | `RESOURCE`, `WORKLOAD` |
| Action cho resource | `CREATE`, `UPDATE`, `REUSE`, `DESTROY`, `UNLINK` |
| Action cho workload | `DEPLOY`, `REMOVE` |

Action dùng động từ chỉ việc sẽ làm (`DESTROY`), khác với status chỉ kết quả đã xảy ra (`DESTROYED`).

### Điều kiện đóng

Domain model có class cho plan và các thành phần của nó; VOPC không còn `Object`/`Map` không kiểu cho plan và override; contract 4–5 định nghĩa fingerprint theo đúng các trường của class; persistence classification và traceability có dòng tương ứng.

## D4 — Chưa rõ ai ghi các bước tiến trình của deployment

**Tương ứng:** issue #7 trong commit `88585cc` ("Writer của progress marker").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

UC-04 hiển thị tiến trình một deployment bằng các dấu kiểm **Infrastructure Ready**, **Configuration Resolved**, **Manifest Generated**, **CD Synced**, **Application Ready**. Theo `05_state_machines/README.md`, mỗi dấu kiểm là một dòng riêng trong bảng `deployment_step`, và UC-04 đọc bảng này qua `getDeploymentProgress()`. Nhưng thiết kế không nói rõ thành phần nào ghi các dòng đó và ghi vào lúc nào:

1. **Chỉ ghi một lần ở cuối.** Trong `sequence_digrams/uc_03_deploy_application.puml`, `deployment_step` chỉ được ghi trong `saveDeploymentRecord(...)` ở bước cuối cùng, sau khi đã gửi desired state sang CD (Contract 9). Trong lúc chạy, IDP chỉ đổi `deployment.status`.
2. **Hai dấu kiểm cuối không ai ghi.** `CD Synced` và `Application Ready` chỉ xảy ra sau khi gửi sang CD, tức sau cả `saveDeploymentRecord`. `05_state_machines/README.md` ghi rõ "Không có operation contract sau `saveDeploymentRecord`", còn UC-04 chỉ đọc, không được ghi.

Hệ quả:

- **Không xem được tiến trình khi deployment đang chạy:** ví dụ đang tạo database mất 10 phút, mở UC-04 không thấy bước nào vì các dòng chỉ được ghi ở cuối.
- **Hai dấu kiểm cuối không bao giờ được tích.** Nếu UC-04 tự suy ra từ Kubernetes thì lại gặp lỗi hiển thị trạng thái của deployment khác (xem D2).

### Liên quan tới vấn đề 2 (đã chốt)

- Worker chờ pod healthy ở mỗi tầng, nên chính worker biết lúc nào CD sync xong và app ready — có thể là nơi ghi hai dấu kiểm cuối.
- Tiến trình được ghi theo tầng, thành phần và bước trong `deployment_step`.

### Hướng giải quyết đã đề xuất (chưa chốt)

1. **Người ghi:** Deployment Orchestrator (qua Deployment Repository), ghi ngay khi mỗi bước bắt đầu (`RUNNING`) và kết thúc (`SUCCEEDED`/`FAILED`); `saveDeploymentRecord` ở cuối chỉ ghi kết quả tổng, không ghi step.
2. **Các bước theo loại thành phần:**

    | Thành phần | Các bước |
    |---|---|
    | Resource | `INFRASTRUCTURE_READY` |
    | Workload | `CONFIGURATION_RESOLVED` → `MANIFEST_GENERATED` → `CD_SYNCED` → `APPLICATION_READY` |
    | Workload bị gỡ (vấn đề 5) | `REMOVED` |
    | Resource bị hủy hoặc gỡ liên kết (vấn đề 5, 3) | `DESTROYED` / `UNLINKED` |

3. **Tạo sẵn các bước `PENDING`** cho mọi thành phần trong plan ngay khi Developer xác nhận; thất bại giữa chừng thì các bước còn lại chuyển `SKIPPED`; thành phần được thêm do lan truyền output (vấn đề 2) thì tạo thêm dòng lúc được thêm.

Ví dụ khi mở UC-04 lúc deployment đang chạy:

```text
10:00  Tầng 0  postgresql  Infrastructure Ready    SUCCEEDED
10:08  Tầng 1  backend     Configuration Resolved  SUCCEEDED
10:09  Tầng 1  backend     Manifest Generated      SUCCEEDED
10:10  Tầng 1  backend     CD Synced               SUCCEEDED
10:10  Tầng 1  backend     Application Ready       RUNNING   ← đang chờ pod healthy
       Tầng 2  frontend    Configuration Resolved  PENDING
```

### Câu hỏi cần chốt khi giải quyết

1. Deployment Orchestrator ghi từng bước ngay khi bắt đầu/kết thúc, thay vì ghi một lần ở cuối?
2. Tạo sẵn các bước `PENDING` khi xác nhận (và `SKIPPED` khi thất bại), hay chỉ tạo dòng khi bước bắt đầu chạy?
3. Danh sách bước theo loại thành phần như bảng trên có đủ chưa?

### Lưu ý khi sửa tài liệu cho các vấn đề đã chốt

Quyết định 10 của vấn đề 2 đã nói tiến trình chi tiết theo tầng/thành phần/bước nằm ở `deployment_step`. Khi sửa tài liệu cho vấn đề 2, mô tả điều đó ở mức khái niệm (tiến trình theo tầng/thành phần được ghi lại và UC-04 đọc được); chi tiết ai ghi, ghi lúc nào, tập bước và trạng thái `PENDING`/`SKIPPED` để lại cho mục này.

### Giá trị ENUM dự kiến (từ vấn đề 9, chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| `deployment_step.status` | `PENDING`, `RUNNING`, `SUCCEEDED`, `FAILED`, `SKIPPED` |
| `deployment_step.step_name` (đổi từ VARCHAR sang ENUM) | `INFRASTRUCTURE_READY`, `CONFIGURATION_RESOLVED`, `MANIFEST_GENERATED`, `CD_SYNCED`, `APPLICATION_READY`, `REMOVED`, `DESTROYED`, `UNLINKED` |

### Điều kiện đóng

Sequence UC-03 thể hiện rõ thời điểm ghi từng bước; có operation/contract cho việc ghi step (kể cả `CD_SYNCED`, `APPLICATION_READY`); state machine README không còn câu "không có operation sau `saveDeploymentRecord`" cho các bước này; traceability chỉ ra writer của mọi bước UC-04 hiển thị.

## D5 — Trạng thái vòng đời của deployment bị trộn với trạng thái của hệ thống CD

**Tương ứng:** issue #8 trong commit `88585cc` ("Lẫn lifecycle với CD delivery status").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

Có hai loại trạng thái khác nhau:

- **Trạng thái vòng đời của deployment** — do IDP quyết định (chờ xác nhận, đang triển khai, thành công, thất bại), được state machine và các contract dùng để chặn thao tác.
- **Trạng thái giao hàng (delivery status)** — do hệ thống CD báo về, ví dụ Argo CD báo `Synced`, `OutOfSync`, `Progressing`, `Degraded`; Fleet lại báo bằng `GitRepo.status.summary` (`ready`, `notReady`, `errApplied`, `outOfSync`, `modified`) và condition `Ready`. Hai sản phẩm, hai tập giá trị hoàn toàn khác nhau.

Contract 9 (`saveDeploymentRecord`, `04_operation_contracts/operation_contracts.md`) trộn hai loại này:

- "Ở success path, `deployment_record.status` phản ánh delivery status đã nhận" — status của record lấy theo trạng thái CD báo về.
- "`Deployment.status` … được cập nhật đồng nhất với trạng thái current/final của Deployment Record" — status của deployment chép theo record.

Tức là: **CD báo gì → `deployment_record.status` → `deployment.status`**.

Hệ quả:

1. **Trạng thái của IDP bị trộn giá trị của CD:** `deployment.status` có thể mang giá trị như `Progressing`, `OutOfSync` — không nằm trong state machine nào, làm các guard (ví dụ chỉ xác nhận được khi `AWAITING_CONFIRMATION`) mất ý nghĩa.
2. **Phá lớp trừu tượng CD:** UC-03 quy định không phụ thuộc trực tiếp vào Argo CD, Flux…; nhưng lưu nguyên giá trị của Argo CD thì khi đổi CD system, dữ liệu status đổi theo. Rủi ro này không còn là giả định: vấn đề 14 đã đổi mặc định sang Fleet, nơi không có giá trị nào tên `Synced` hay `Progressing`.
3. **Hai nơi lưu cùng một thứ:** `deployment.status` và `deployment_record.status` luôn phải bằng nhau, dễ lệch nhau.

### Liên quan tới các quyết định/mục khác

- **Vấn đề 2 (đã chốt):** `deployment.status` là `AWAITING_CONFIRMATION → CONFIRMED → DEPLOYING → SUCCEEDED | FAILED`, và IDP tự quyết `SUCCEEDED` dựa trên việc chờ pod healthy — không cần lấy từ CD.
- **D4:** trạng thái CD của từng workload thuộc bước `CD_SYNCED`.
- **D2:** trạng thái CD xem trực tiếp ở UC-04.

### Hướng giải quyết đã đề xuất (chưa chốt)

1. `deployment.status` chỉ chứa trạng thái vòng đời của IDP, không bao giờ chép giá trị từ CD.
2. Bỏ `deployment_record.status` (record 1–1 với deployment, lấy trạng thái từ `deployment.status`).
3. Trạng thái CD, nếu cần lưu, lưu ở trường riêng (ví dụ `delivery_status`) và quy về tập giá trị trung lập do CD abstraction chuyển đổi (ví dụ `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED`), không lưu nguyên tên của Argo CD, Fleet hay Flux.

### Câu hỏi cần chốt khi giải quyết

1. `deployment.status` chỉ là trạng thái của IDP, tách hẳn khỏi trạng thái CD?
2. Bỏ `deployment_record.status`, hay giữ cả hai nhưng bắt buộc luôn bằng nhau?
3. Tập giá trị trung lập của trạng thái CD là gì, và lưu ở đâu (gộp với D4)?

### Lưu ý khi sửa tài liệu cho các vấn đề đã chốt

Khi sửa tài liệu theo vấn đề 2, không được mô tả `deployment.status` hay `deployment_record.status` nhận giá trị từ CD; contract 9 cần bỏ câu "phản ánh delivery status đã nhận" khỏi `deployment.status`. Phần cấu trúc lưu trạng thái CD để lại cho mục này.

### Giá trị ENUM dự kiến (từ vấn đề 9, chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| `deployment_record.status` | Bỏ cột, dùng `deployment.status` |
| `delivery_status` (trường riêng, nếu lưu) | `ACCEPTED`, `SYNCING`, `SYNCED`, `FAILED` |

### Điều kiện đóng

Contract 8–9, schema/ERD, domain model và state machine thống nhất: `deployment.status` chỉ nhận giá trị lifecycle của IDP; trạng thái CD (nếu lưu) nằm ở trường riêng với tập giá trị trung lập; không còn hai nơi lưu cùng một trạng thái mà không có quy tắc đồng bộ.

## D6 — Phục hồi khi Deployment Worker chết giữa chừng

**Tương ứng:** một phần của issue #10 trong commit `88585cc` ("Confirm chạy tác vụ dài trong HTTP request"). Phần tách nhận việc / làm việc đã được chốt ở vấn đề 10 (xem `design_decisions.md`); mục này chỉ gồm phần phục hồi.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Bối cảnh

Theo quyết định của vấn đề 10:

- `confirmDeployment` đổi `deployment.status` thành `CONFIRMED` và tạo job trong bảng DB trong cùng một transaction, rồi trả lời UI ngay.
- Deployment Worker chạy nền lấy job, chạy chuỗi triển khai theo tầng (vấn đề 2), đổi status sang `DEPLOYING` rồi `SUCCEEDED`/`FAILED`.

Job lưu trong DB nên không mất khi server khởi động lại. Nhưng thiết kế chưa nói gì về trường hợp **worker chết khi đang chạy dở một job** (server restart, crash, mất kết nối tới DB hoặc cluster).

### Vấn đề

Ví dụ deployment `shop-app` gồm ba tầng:

```text
Tầng 0: postgresql  → đã tạo xong trên AWS
Tầng 1: backend     → đã gửi sang CD, đang chờ pod healthy
        ← worker chết ở đây
Tầng 2: frontend    → chưa làm
```

Các câu hỏi chưa có câu trả lời:

1. **Ai phát hiện job bị bỏ dở?** Job vẫn ở trạng thái "đang chạy" nhưng không còn worker nào làm; deployment kẹt ở `DEPLOYING`.
2. **Làm tiếp hay làm lại từ đầu?** Làm lại từ đầu thì tầng 0 phải nhận ra database đã tồn tại để không tạo trùng; làm tiếp thì phải biết chính xác đã xong tới đâu (liên quan D4 — các bước tiến trình).
3. **Chống hai worker cùng chạy một job:** worker cũ chỉ bị treo chứ chưa chết, worker mới lại lấy job → hai worker cùng tạo hạ tầng, cùng ghi status. Cần cơ chế khóa hoặc thời hạn giữ job (lease) và cách chặn worker cũ ghi đè.
4. **Không tạo trùng hạ tầng khi chạy lại:** provisioner phải idempotent (chạy lại không tạo thêm), ví dụ Terraform refresh + plan trên cùng state.
5. **Thử lại bao nhiêu lần, lỗi nào đáng thử lại:** lỗi mạng tạm thời khác với lỗi cấu hình sai.
6. **Dữ liệu đầu vào đã thay đổi khi chạy lại:** nếu trong lúc worker chết có người lưu phiên bản định nghĩa mới hoặc sửa cấu hình, lần chạy lại dùng dữ liệu nào (liên quan D3 — fingerprint của plan; vấn đề 5 — deployment gắn với một phiên bản định nghĩa).

### Hướng có thể cân nhắc (chưa chốt)

- Job có thời hạn giữ (lease) và worker gia hạn định kỳ (heartbeat); hết hạn thì worker khác được lấy lại job.
- Mọi lần ghi status/step của worker kèm điều kiện "vẫn đang giữ job" để chặn worker cũ ghi đè.
- Làm tiếp theo tầng: tầng nào đã ghi `SUCCEEDED` (D4) thì bỏ qua; tầng đang dở thì chạy lại với provisioner idempotent.
- Giới hạn số lần thử lại; quá giới hạn thì deployment `FAILED` kèm lỗi rõ ràng.

### Giá trị ENUM dự kiến (chốt khi giải quyết mục này)

| ENUM | Giá trị dự kiến |
|---|---|
| Trạng thái job | `QUEUED`, `RUNNING`, `COMPLETED`, `FAILED` |

Có thể cần thêm giá trị khi chốt cơ chế lease/thử lại.

### Câu hỏi cần chốt khi giải quyết

1. Phát hiện job bỏ dở bằng lease/heartbeat hay cách khác?
2. Làm tiếp theo tầng hay làm lại từ đầu?
3. Chặn worker cũ ghi đè bằng cách nào?
4. Chính sách thử lại: số lần, loại lỗi được thử lại?
5. Lần chạy lại dùng dữ liệu đầu vào lúc xác nhận hay dữ liệu hiện tại?

### Điều kiện đóng

Sequence của Deployment Worker thể hiện được kịch bản "worker chết giữa chừng → job được lấy lại → hoàn tất hoặc thất bại rõ ràng"; không kịch bản nào tạo trùng hạ tầng, để deployment kẹt vĩnh viễn ở `DEPLOYING`, hoặc có hai worker cùng ghi một deployment; contract và state machine của job/deployment phản ánh đúng cơ chế đã chọn.

## D7 — Secret bị bỏ rơi trong Secret Store

**Tương ứng:** issue #11 trong commit `88585cc` ("Secret bị orphan khi save lỗi").

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Bối cảnh

Để DB của IDP không bao giờ chứa secret dạng chữ thường, UC-02 làm như sau: gửi giá trị secret vào **Secret Store**, nhận về một **secret reference**, và DB chỉ lưu reference đó (`secret.secret_ref`).

### Vấn đề

Trong `sequence_digrams/uc_02_configure_application_environment.puml`, secret được ghi vào Secret Store **ngay lúc Developer nhập**, còn reference chỉ được lưu vào DB **khi bấm Save**:

```text
Developer nhập DB_PASSWORD
  → setDirectConfigurationValue(secret, secretValue)
  → storeSecret(applicationId, environment, secretValue)   ← ghi vào Secret Store ngay
  → nhận secret reference

... Developer tiếp tục điền các biến khác ...

Developer bấm Save
  → validateEnvironmentConfiguration(configuration)
  → save(configuration with value references)              ← lúc này mới lưu reference vào DB
```

Mọi trường hợp không đi tới được bước lưu DB đều để lại một secret không ai trỏ tới (orphan):

| Tình huống | Kết quả |
|---|---|
| Developer nhập secret rồi đóng tab, không bấm Save | Secret nằm lại trong Secret Store |
| Bấm Save nhưng validate thất bại rồi bỏ đi | Như trên |
| Lưu DB lỗi (ví dụ DB mất kết nối) | Như trên |
| Developer nhập lại secret nhiều lần trước khi Save | Các bản trước nằm lại |
| Đổi secret ở lần cấu hình sau | Bản cũ vẫn nằm trong Secret Store |

Hậu quả:

- **Rủi ro bảo mật:** secret thật tồn tại mà không ai biết, không ai quản lý, không ai thu hồi.
- **Rác tích tụ** theo thời gian, có thể tốn chi phí (nhiều dịch vụ secret tính tiền theo số secret).
- Contract 3 (`saveEnvironmentConfiguration`) chỉ bảo đảm plaintext không vào DB, không nói gì về việc dọn các secret không được dùng.

### Liên quan

- **D1:** câu hỏi "secret nằm ở đâu trong lúc Developer chưa bấm Save" là một phần của câu hỏi bản nháp được giữ ở đâu.
- **Vấn đề 5 (đã chốt):** cấu hình UC-02 chưa có phiên bản; khi đổi secret, bản cũ không còn được cấu hình nào trỏ tới.

### Hướng có thể cân nhắc (chưa chốt)

**A. Chỉ ghi vào Secret Store lúc bấm Save:** validate xong mới ghi secret, rồi lưu reference vào DB; lưu DB lỗi thì xóa ngay secret vừa ghi; lưu thành công mà secret bị thay thì xóa bản cũ. Đơn giản, loại được gần hết các tình huống; nhưng secret nằm trong form tới lúc Save (dính D1), và nếu server chết đúng giữa "ghi Secret Store" và "lưu DB" thì vẫn có thể sót.

**B. Lưu tạm có hạn dùng** (hướng của `88585cc`): lúc nhập, secret vào Secret Store dưới dạng tạm, tự hết hạn; Save thành công thì chuyển thành chính thức; không Save thì tự biến mất. Secret rời trình duyệt ngay, không có rác lâu dài; nhưng phức tạp hơn (tạm/chính thức/thu hồi/hết hạn) và có kẽ hở khi DB đã lưu mà chưa kịp chuyển thành chính thức thì bản tạm hết hạn.

**C. Tác vụ dọn rác định kỳ:** định kỳ so danh sách secret trong Secret Store với reference trong DB, xóa cái không ai trỏ tới. Bắt được mọi trường hợp sót, kể cả server chết giữa chừng; nhưng giữa hai lần dọn, secret orphan vẫn tồn tại.

Đề xuất lúc bàn: A, kèm C làm lưới an toàn.

### Câu hỏi cần chốt khi giải quyết

1. Chọn hướng nào (A, B, C hoặc kết hợp)?
2. Khi đổi secret: xóa bản cũ ngay sau khi lưu thành công, hay giữ lại một thời gian để có thể quay lại?
3. Có giải quyết cùng lúc với D1 (bản nháp) không?

### Điều kiện đóng

Sequence UC-02, contract 3 và VOPC (Secret Store) thể hiện rõ thời điểm ghi secret, cách xử lý khi validate/lưu DB thất bại và khi secret bị thay; không tình huống nào trong bảng trên để lại secret không ai quản lý mà không có cơ chế dọn.

## D8 — Cơ chế cụ thể để đọc Workload Output

**Tương ứng:** quyết định 5 của vấn đề 2 trong `design_decisions.md`.

**Quyết định:** để ở mức trừu tượng, chưa chốt cơ chế.

### Bối cảnh

Theo vấn đề 2, UC-03 triển khai theo tầng: sau khi workload của một tầng healthy, IDP thu thập Workload Output (ví dụ `backend.endpoint`) để resolve configuration của các tầng sau; khi deploy một phần, IDP đọc output từ workload phụ thuộc đang chạy ngoài phạm vi. Tài liệu hiện chỉ quy định:

- `Workload Output Collector.collectWorkloadOutputs(target, workloads)` (contract 10) trả về tập `Workload Output` transient.
- Collector gọi `Workload Status Provider / Kubernetes Adapter.readWorkloadOutputs(target, workloads)`, adapter đọc dữ liệu runtime từ Kubernetes Cluster (`readWorkloadRuntimeData`).
- `Workload.exposedOutputs` chỉ là danh sách tên output.

### Vấn đề chưa giải quyết

- Mỗi output trong `exposedOutputs` được lấy từ đâu (Service DNS/endpoint, Ingress, annotation, trạng thái của resource Kubernetes…) và ai khai báo cách lấy.
- Output có cần khai báo thêm thông tin (ví dụ kiểu, port, scheme) để đọc được một cách tất định không.
- Cách chuẩn hóa output trước khi tính dấu vân tay, để không báo "thay đổi" giả (ví dụ thứ tự, định dạng).

### Điều kiện đóng

Domain model/ERD mô tả đủ thông tin để đọc từng loại output; sequence UC-03 và contract 10 chỉ rõ nguồn đọc; cách chuẩn hóa output trước khi hash được đặc tả.

## D9 — Resource dùng chung và resource riêng theo workload

**Tương ứng:** mục "Hoãn" của vấn đề 3 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này, để lại xử lý sau.

### Vấn đề

1. **Output của resource dùng chung thay đổi không lan sang application khác.** Resource Definition loại `EXISTING` cho nhiều application/environment trỏ cùng một resource thật. Khi output của resource đó thay đổi (ví dụ host mới), `propagateOutputChanges` (contract 11) chỉ lan truyền trong phạm vi application đang deploy; các application khác chỉ phát hiện qua dấu vân tay output ở lần deploy sau của chính chúng. IDP không tự deploy lại các application đó.
2. **Chưa có resource riêng theo từng workload.** Thiết kế hiện tại đặt Resource Requirement ở mức application (theo phiên bản); nhiều workload cùng depends on một requirement thì dùng chung một Resource Instance. Mức "resource riêng của một workload" (private theo workload như Humanitec) chưa được mô hình hóa.

### Câu hỏi cần chốt khi giải quyết

1. Khi output của resource `EXISTING` thay đổi, có cần thông báo hoặc tự tạo deployment cho các application đang dùng chung không? Ai phát hiện thay đổi đó (platform hay IDP)?
2. Có cần resource riêng theo workload không, và nếu có thì khóa chủ sở hữu của Resource Instance thêm workload như thế nào?

### Điều kiện đóng

Contract 11 và use case UC-03 mô tả rõ phạm vi lan truyền với resource dùng chung; domain model/ERD thể hiện quyết định về resource riêng theo workload.

## D10 — Khóa hoặc ngừng hỗ trợ phiên bản catalog cũ

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này; mọi phiên bản catalog đều được chọn khi deploy.

### Vấn đề

Theo vấn đề 12, catalog có phiên bản bất biến và Developer chọn phiên bản khi deploy, kể cả phiên bản cũ hơn phiên bản đang chạy. Platform chưa có cách:

- Cấm dùng một phiên bản có lỗi hoặc lỗ hổng (vd module Terraform tạo cụm với cấu hình không an toàn).
- Buộc application chuyển khỏi phiên bản cũ trước một thời hạn.
- Biết application/environment nào còn chạy trên phiên bản nào để thông báo.

### Câu hỏi cần chốt khi giải quyết

1. Phiên bản catalog có trạng thái (vd `ACTIVE`, `DEPRECATED`, `BLOCKED`) không, và ai đổi trạng thái (platform administration nằm ngoài năm Developer use case)?
2. Deployment đang chạy trên phiên bản bị khóa thì deploy một phần có bị chặn không, hay chỉ chặn chọn phiên bản đó cho deployment mới?
3. Có cần màn hình cho platform xem application nào đang dùng phiên bản nào không?

### Điều kiện đóng

ERD và domain model thể hiện trạng thái phiên bản catalog (nếu có); UC-03 A1 và contract 4 mô tả hành vi khi chọn hoặc đang chạy phiên bản bị khóa.

## D11 — Phiên bản catalog mới đổi hẳn công thức của một resource đang chạy

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này.

### Vấn đề

Ví dụ: shop-app staging chạy trên cụm nội bộ với catalog v1, PostgreSQL dựng bằng công thức `postgres-k8s` (pod Postgres trong cụm, dữ liệu nằm trong đó). Platform ra catalog v2, trong đó PostgreSQL trên cụm nội bộ đổi sang công thức khác, vd `postgres-shared` (database dùng chung có sẵn). Developer chọn catalog v2 và deploy.

Nếu v2 chỉ đổi tham số trong cùng công thức (vd dung lượng 10GB → 20GB) thì IDP cập nhật resource bình thường; mục này chỉ nói trường hợp đổi sang công thức khác.

### Câu hỏi cần chốt khi giải quyết

1. IDP báo lỗi và không cho deploy (Developer/platform tự chuyển dữ liệu hoặc gỡ app trước), hay tự hủy resource cũ rồi dựng lại theo công thức mới sau khi Developer xác nhận cảnh báo mất dữ liệu?
2. Nếu tự dựng lại: thứ tự hủy/tạo thế nào để workload đang dùng resource không bị gián đoạn lâu?

### Điều kiện đóng

Đặc tả UC-03 (A1 hoặc plan), contract 4 và contract 6 mô tả rõ hành vi khi công thức của một resource đang chạy thay đổi giữa hai phiên bản catalog.

## D12 — UC-02 lấy danh sách output từ phiên bản catalog nào

**Tương ứng:** mục "Hoãn" của vấn đề 12 trong `design_decisions.md`.

**Quyết định:** chưa giải quyết lúc này; UC-02 và contract 3 giữ nguyên.

### Vấn đề

Ở UC-02, Developer gán biến vào output của resource (vd `DB_HOST ← postgresql.host`) và IDP chỉ cho chọn output mà Resource Definition có. Catalog nay có nhiều phiên bản nhưng cấu hình không gắn với phiên bản catalog nào.

Ví dụ: catalog v1 có output `host`, `port`, `username`, `password`; catalog v2 thêm `reader_host`. Developer gán `READ_DB_HOST ← postgresql.reader_host` rồi deploy với catalog v1.

### Câu hỏi cần chốt khi giải quyết

1. UC-02 hiển thị output theo phiên bản catalog mới nhất và UC-03 kiểm tra lại theo phiên bản được chọn (A1 nếu output không có), hay cho Developer chọn phiên bản catalog ngay ở UC-02?
2. Resource cùng loại có thể dùng công thức khác nhau tùy nơi triển khai (vd Aurora trên AWS, Postgres trong cụm nội bộ), với danh sách output khác nhau; UC-02 chưa biết nơi triển khai thì hiển thị output nào?

### Điều kiện đóng

Đặc tả UC-02, contract 3 và UC-03 A1 thống nhất nguồn danh sách output và thời điểm kiểm tra.

## D13 — Teardown đánh dấu workload đã gỡ dù chưa xác minh được cụm

**Tương ứng:** rà soát UC-05 sau commit `8302586`.

**Quyết định:** chưa sửa code lúc này. Đặc tả UC-05 mô tả hành vi đúng (chỉ đánh dấu đã gỡ sau khi xác minh); code hiện chưa bảo đảm điều đó trong mọi trường hợp.

### Vấn đề

Ở bước gỡ workload, worker lấy kết nối cụm rồi chỉ báo lỗi khi đây là deployment loại triển khai:

```go
cluster, err := ex.clusterAccess(ctx)
if err != nil && d.Kind == domain.KindDeploy {
    return ex.fail(...)
}
if cluster != nil { /* publish, chờ CD, xác minh biến mất, xóa CD object, xóa namespace */ }
```

Với teardown, lỗi bị bỏ qua: cả khối xác minh bị nhảy qua vì `cluster == nil`, nhưng đoạn ngay sau đó vẫn đánh dấu mọi Workload Instance là đã gỡ, rồi các tầng sau vẫn hủy resource.

Ví dụ: bản ghi kết nối cụm nội bộ đã cũ (cụm được dựng lại sau khi đăng ký), hoặc cụm tạm thời không truy cập được. Teardown vẫn báo thành công, database ghi workload đã gỡ và resource đã hủy, trong khi Deployment và Service vẫn đang chạy trong cụm. Lần deploy sau lên cùng chủ sở hữu sẽ chồng lên phần còn sót đó.

Điều này mâu thuẫn với quy tắc nghiệp vụ của UC-05 và với nguyên tắc đã áp dụng cho UC-03: chỉ đánh dấu đã gỡ sau khi xác minh thành phần đã biến mất khỏi cụm.

### Câu hỏi cần chốt khi giải quyết

1. Có trường hợp nào được phép bỏ qua xác minh không? Trường hợp hợp lý duy nhất nhìn thấy: chính cụm nằm trong danh sách gỡ của lần này và đã ở trạng thái đã hủy hoặc đã gỡ liên kết — khi đó không còn nơi nào để xác minh.
2. Khi không lấy được cụm ngoài trường hợp trên thì dừng hẳn theo A2, hay đánh dấu bằng một trạng thái riêng (ví dụ đã gỡ nhưng chưa xác minh) để lần sau dọn tiếp?
3. Có cần bước dọn phần còn sót khi deploy lại lên cùng application + environment + nơi triển khai không?

### Điều kiện đóng

Worker chỉ đánh dấu workload đã gỡ sau khi xác minh, hoặc ngoại lệ được phép được ghi rõ trong cả code lẫn đặc tả UC-05; có test cho nhánh không lấy được kết nối cụm.

## D14 — Teardown không dọn CD object, desired state và namespace khi plan chỉ còn resource

**Tương ứng:** rà soát UC-05 sau commit `8302586`.

**Quyết định:** chưa sửa code lúc này.

### Vấn đề

Việc xóa đối tượng của application khỏi CD system, xóa thư mục `<nơi triển khai>/<environment>` trong Delivery Repository và xóa namespace nằm bên trong điều kiện "tầng gỡ này có workload":

```go
if len(workloadIDs) > 0 {
    ...
    if d.Kind == domain.KindTeardown {
        w.CD.RemoveApplication(...)
        w.Kube.DeleteNamespace(...)
    }
}
```

Nhưng contract 12 cho phép gỡ khi chỉ còn Resource Instance, không còn workload nào — ví dụ lần teardown trước đã gỡ xong workload rồi thất bại ở bước hủy resource. Lần gỡ thứ hai sẽ hủy nốt resource nhưng **không bao giờ** xóa đối tượng CD, desired state và namespace: CD system tiếp tục theo dõi một đường dẫn rỗng, namespace ở lại trong cụm.

### Câu hỏi cần chốt khi giải quyết

1. Đưa ba bước dọn đó ra ngoài điều kiện, chạy đúng một lần cho mỗi teardown, kể cả khi không còn workload nào?
2. Nếu namespace còn object không do IDP tạo thì xóa hay giữ và báo?
3. Chạy lại teardown lần hai có được coi là cách xử lý chính thức cho lần đầu thất bại giữa chừng không? Nếu có thì nó phải dọn được mọi thứ còn sót.

### Điều kiện đóng

Teardown luôn xóa đối tượng CD, desired state của environment đó và namespace đúng một lần, kể cả khi plan chỉ còn resource; có kiểm chứng thật cho trường hợp teardown lần hai sau một lần thất bại.
