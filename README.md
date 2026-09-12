# IDP — MVP deployment design

Mốc hiện tại: **UC-03 happy path deploy frontend + backend Go + PostgreSQL lên AWS thật** bằng Terraform và Argo CD, kiểm thử CRUD rồi cleanup tài nguyên demo. Source application/configuration chuẩn bị bằng fixture; chỉ cần query tối thiểu để nghiệm thu UC-03.

**Kiểm thử xong phải lưu bằng chứng và xóa ngay tài nguyên AWS do task tạo để tránh phát sinh phí, không giữ demo chạy chờ bàn giao.** Nếu thất bại giữa chừng vẫn phải teardown an toàn và báo rõ tài nguyên còn sót; không xóa tài nguyên có sẵn của người dùng.

IDP/công cụ có thể chạy local; kind chỉ hỗ trợ phát triển/test, **không phải target nghiệm thu**. Chưa chốt EKS hay Kubernetes trên EC2; phải ghi cấu hình cloud và chi phí trước khi provision. PostgreSQL mẫu chạy trên Kubernetes target AWS, chưa yêu cầu RDS.

## Đọc theo thứ tự

1. [MVP scope](MVP_SCOPE.md): phạm vi và tiêu chí demo.
   [Prompt code UC3](plab_mvp.md): nhiệm vụ triển khai mốc happy-path AWS, giới hạn và bàn giao.
2. [Deployment design](MVP_DEPLOYMENT_DESIGN.md): snapshot, confirm, worker, resource state, Argo publish/query và recovery.
3. [Operation contracts](04_operation_contracts/operation_contracts.md) và [schema](03_database_erd/schema.md).
4. [Domain model](02_domain_model/domain_objects.md), [VOPC](01_vopc_design_class_diagram/README.md), [state machines](05_state_machines/README.md).
5. [Traceability](06_traceability/traceability_matrix.md) và [MVP design review](06_traceability/mvp_design_review.md).

Sequence diagrams: [deploy](sequence_digrams/uc_03_deploy_application.puml), [query](sequence_digrams/uc_04_view_deployment_result.puml), [manual recovery](sequence_digrams/mvp_recover_deployment.puml).

## Trạng thái

Thiết kế luồng làm đầu vào implementation; cấu hình dịch vụ AWS cụ thể cần chốt trước provision. Chưa có mã ứng dụng/migration và chưa nghiệm thu runtime/cloud. Tám finding được xử lý ở mức thiết kế rộng; không có nghĩa mốc happy-path đã implement/test mọi recovery/redeploy scenario. Secret staging (R3) và use case không configuration (R9) hoãn theo phạm vi. [Lịch sử review](06_traceability/review_open_issues.md) giữ bằng chứng baseline.

Sơ đồ editor UC-01/02 là reference tương lai, không phải yêu cầu code trong mốc deploy-first. Source fixture C1 dùng permanent Secret reference; không triển khai staging/shared resource/automatic replay.
