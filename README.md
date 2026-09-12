# IDP — MVP deployment design

Mốc đầu tiên: deploy ứng dụng frontend + backend Go + PostgreSQL lên kind bằng Terraform và Argo CD. Source application/configuration chuẩn bị bằng fixture; ưu tiên UC-03 và query UC-04.

## Đọc theo thứ tự

1. [MVP scope](MVP_SCOPE.md): phạm vi và tiêu chí demo.
2. [Deployment design](MVP_DEPLOYMENT_DESIGN.md): snapshot, confirm, worker, resource state, Argo publish/query và recovery.
3. [Operation contracts](04_operation_contracts/operation_contracts.md) và [schema](03_database_erd/schema.md).
4. [Domain model](02_domain_model/domain_objects.md), [VOPC](01_vopc_design_class_diagram/README.md), [state machines](05_state_machines/README.md).
5. [Traceability](06_traceability/traceability_matrix.md) và [MVP design review](06_traceability/mvp_design_review.md).

Sequence diagrams: [deploy](sequence_digrams/uc_03_deploy_application.puml), [query](sequence_digrams/uc_04_view_deployment_result.puml), [manual recovery](sequence_digrams/mvp_recover_deployment.puml).

## Trạng thái

Thiết kế MVP sẵn sàng làm đầu vào implementation; chưa có mã ứng dụng/migration và chưa nghiệm thu runtime. Tám finding được xử lý ở mức thiết kế MVP; secret staging (R3) và use case không configuration (R9) hoãn theo phạm vi. [Lịch sử review](06_traceability/review_open_issues.md) giữ bằng chứng baseline.

Sơ đồ editor UC-01/02 là reference tương lai, không phải yêu cầu code trong mốc deploy-first. Source fixture C1 dùng permanent Secret reference; không triển khai staging/shared resource/automatic replay.
