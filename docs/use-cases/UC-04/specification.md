---
id: UC-04
artifact: use-case-specification
status: current
delivery_status: partially-implemented
last_reviewed: 2026-09-18
---

# UC-04 — View Deployment Result


## Mục tiêu

Cho phép Developer xem tiến trình và kết quả của deployment, bao gồm image version thực tế đang được triển khai.

## Actor

**Primary Actor:** Developer

## Tiền điều kiện

Developer có Auth Session hợp lệ và Local User Account còn `ACTIVE` theo UC-06.

Application có ít nhất một deployment.

## Hậu điều kiện

Developer xem được trạng thái deployment.

Không làm thay đổi application hoặc infrastructure.

## Luồng chính

Developer mở application và chọn Deployments.

**IDP** hiển thị lịch sử deployment.

Developer chọn một deployment.

**IDP** hiển thị:

Environment.

Deployment target.

Phiên bản Application Definition đã deploy.

Phiên bản catalog đã dùng.

Image repository và version của từng workload, và workload nào được triển khai lại tự động.

Các thành phần đã được gỡ, hủy hoặc gỡ liên kết trong deployment, nếu có.

Deployment status.

Tiến trình theo từng tầng và từng thành phần.

Infrastructure status.

CD status.

Workload health.

Endpoint nếu có.

Ví dụ:

Deployment #42 Production — Phiên bản 5 — Catalog v2 — Succeeded

Tầng 0 postgresql ✓ Infrastructure Ready

Tầng 1 backend registry.company.local/shop-backend:v1.4.3 Healthy ✓ Configuration Resolved ✓ Manifest Generated ✓ CD Synced ✓ Application Ready

Tầng 2 frontend registry.company.local/shop-frontend:v2.1.0 Healthy (triển khai lại tự động) ✓ Configuration Resolved ✓ Manifest Generated ✓ CD Synced ✓ Application Ready

## Luồng ngoại lệ

A1 – Deployment thất bại

**IDP** hiển thị tầng, failed step, workload/resource liên quan và error summary.

A2 – Workload chưa Healthy

**IDP** hiển thị workload và image version gặp vấn đề.

## Quy tắc nghiệp vụ

UC-04 chỉ cung cấp trạng thái và kết quả deployment.

Phải hiển thị image/version thực tế đã sử dụng trong deployment.

Phải phân biệt workload được Developer chọn với workload được triển khai lại tự động.

Logs, metrics, traces và diagnostics sâu nằm ngoài phạm vi UC
