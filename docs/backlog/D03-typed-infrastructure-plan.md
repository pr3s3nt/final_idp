---
id: D03
artifact: deferred-issue
status: deferred
last_reviewed: 2026-09-17
source_record: "git:f58765e:docs/archive/consolidated/deferred-issues-log.md"
---

# D03 — Infrastructure Plan chưa có cấu trúc typed rõ ràng

**Tương ứng:** issue #6 trong commit `88585cc` ("Infrastructure Plan chưa có typed model").

**Quyết định:** xem xét sau. Vấn đề chỉ gây hại khi dữ liệu đầu vào của plan bị thay đổi trong khoảng thời gian giữa lúc lập plan (`createDeployment`) và lúc Developer xác nhận (`confirmDeployment`), ví dụ có người khác sửa cấu hình hoặc định nghĩa trong lúc đó. Trường hợp này chưa cần lo ở giai đoạn hiện tại.

### Vấn đề

Trước khi đụng vào hạ tầng thật, UC-03 lập một plan để: hiển thị cho Developer những gì sắp làm; cho Developer override một số tham số được phép; và tính dấu vân tay (fingerprint). Lúc tạo deployment, IDP lưu fingerprint của plan; lúc Developer xác nhận, IDP dựng lại plan, tính lại fingerprint, nếu khác thì báo `PLAN_CHANGED`.

Plan được nhắc ở nhiều nơi nhưng không nơi nào định nghĩa nó gồm những gì:

| Nơi | Plan được mô tả thế nào |
|---|---|
| `docs/use-cases/UC-03/sequence.puml` | Chỉ là dòng chữ `Infrastructure plan (create/update/reuse)` |
| `docs/use-cases/UC-03/vopc.puml`, `docs/architecture/design-class-diagram.puml` | `Infrastructure Planner` có `-plan: Object`, `-allowedOverrides: Map` |
| `docs/architecture/contracts/operation-contracts.md` (Contract 4, 5) | Một đoạn văn liệt kê "những thứ đưa vào fingerprint" |
| `docs/architecture/database/schema.md` | Chỉ có cột `plan_fingerprint`, `plan_fingerprint_algo` |
| `docs/architecture/domain/domain-model.puml` | Không có class; chỉ có ghi chú "Infrastructure Plan remains TRANSIENT" |
| `docs/architecture/domain/persistence-classification.md` | Tự nhận bao phủ toàn bộ domain object nhưng không có dòng Infrastructure Plan |

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
