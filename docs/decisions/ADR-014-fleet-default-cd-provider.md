---
id: ADR-014
artifact: architecture-decision-record
status: current
outcome: accepted
last_reviewed: 2026-09-17
source_record: ../archive/consolidated/design-decisions-log.md
---

# ADR-014 — Fleet là CD provider mặc định

**Current status (2026-09-17):** accepted and implemented for `kind-local`. AWS intentionally continues to use Argo CD as recorded in the decision.

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

**Sẽ ảnh hưởng:** chỉ các dòng ví dụ trong [các use case](../use-cases/README.md), [`design-classes.md`](../architecture/design-classes.md) và [backlog](../backlog/README.md); phần còn lại là code và tài liệu vận hành.

**Đã áp dụng (nhánh `uc03-impl`, 16/09/2026):** tài liệu và code đều xong, kiểm chứng thật trên cụm nội bộ ([Fleet verification](../verification/2026-09-16-fleet-provider.md)). Hai application chạy hai CD system khác nhau trên cùng một cụm.

**Một hệ quả phát hiện khi chạy thật:** manifest do IDP sinh ra từng đánh dấu bằng nhãn chuẩn `app.kubernetes.io/managed-by: idp`. Fleet triển khai bundle qua Helm, mà Helm luôn đặt nhãn đó thành `Helm`, nên object trong cụm lệch Git vĩnh viễn và Fleet không bao giờ báo xong. Bài học chung, không riêng Fleet: **IDP chỉ được đánh dấu bằng nhãn thuộc không gian tên của mình** (`idp.dev/managed-by`), vì nhãn `app.kubernetes.io/managed-by` thuộc về công cụ trực tiếp áp manifest xuống cụm, và công cụ đó thay đổi theo CD system.
