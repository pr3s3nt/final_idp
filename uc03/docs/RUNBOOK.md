# RUNBOOK – UC-03 Deploy Application

Mọi lệnh chạy trong thư mục `uc03/`.

## 1. Yêu cầu máy

| Công cụ | Đã dùng khi kiểm chứng |
|---|---|
| Go | 1.27 |
| Docker | 29.x (có mạng `kind`) |
| kind CLI (chỉ để xem/xóa cụm thủ công) | 0.32 |
| Terraform | 1.9.8 |
| score-k8s | 0.15.0 |
| kubectl, jq, git, ssh | bất kỳ bản gần đây |
| gh (tạo GitOps repo một lần) | 2.62 |
| aws CLI + credentials (chỉ target `aws`) | 2.35 |

Internet cần tới: registry.terraform.io, argoproj.github.io (Helm chart), quay.io/ghcr.io/docker.io (image Argo CD, postgres, redis), github.com:22 (GitOps).

## 2. Biến môi trường

| Biến | Bắt buộc | Ý nghĩa |
|---|---|---|
| `IDP_SECRET_KEY` | có | Khóa Secret Store; phải giống nhau giữa import, serve, worker |
| `IDP_DATABASE_URL` | không | mặc định `postgres://idp:idp@127.0.0.1:55433/idp?sslmode=disable` |
| `IDP_GITOPS_REPO` | worker | `git@github.com:<owner>/final-idp-gitops.git` |
| `IDP_GITOPS_SSH_KEY_FILE` | worker | deploy key có quyền ghi (IDP push) |
| `IDP_GITOPS_READ_SSH_KEY_FILE` | worker | deploy key chỉ đọc (Argo CD) |
| `IDP_AWS_ECR_REGISTRY` | import | registry ECR của account, điền vào `eks-cluster.image_registry_mirror`. Phải đặt **trước** `import-fixtures`: phiên bản catalog đã tạo không sửa được |
| `IDP_HEALTH_TIMEOUT` | không | thời gian chờ sync/rollout, mặc định 6m |
| `IDP_LISTEN` | không | mặc định `127.0.0.1:8088` |

`known_hosts` của GitHub được đọc từ cùng thư mục với `IDP_GITOPS_SSH_KEY_FILE`.

## 3. Chuẩn bị một lần

```bash
# DB của IDP và test DB
docker run -d --name idp-uc03-db --restart unless-stopped -e POSTGRES_USER=idp -e POSTGRES_PASSWORD=idp -e POSTGRES_DB=idp \
  -p 127.0.0.1:55433:5432 postgres:17-alpine
docker exec idp-uc03-db psql -U idp -d idp -c "CREATE DATABASE idp_test"

# Registry image cho target kind-local (đóng vai registry của CI)
docker run -d --name idp-uc03-registry --restart unless-stopped -p 127.0.0.1:5055:5000 registry:2
docker network connect kind idp-uc03-registry   # tạo mạng bằng `docker network create kind` nếu chưa có

# GitOps repo + deploy key (không dùng token gh trong IDP/cụm)
KEYDIR=~/.config/idp-uc03; mkdir -p $KEYDIR && chmod 700 $KEYDIR
ssh-keygen -q -t ed25519 -N '' -f $KEYDIR/gitops_write; ssh-keygen -q -t ed25519 -N '' -f $KEYDIR/gitops_read
ssh-keyscan -t ed25519,rsa,ecdsa github.com > $KEYDIR/known_hosts
gh repo create <owner>/final-idp-gitops --private --add-readme
gh repo deploy-key add $KEYDIR/gitops_write.pub --allow-write --title idp-uc03-write -R <owner>/final-idp-gitops
gh repo deploy-key add $KEYDIR/gitops_read.pub --title idp-uc03-argocd-read -R <owner>/final-idp-gitops

export IDP_SECRET_KEY=... IDP_AWS_ECR_REGISTRY=<account>.dkr.ecr.ap-southeast-1.amazonaws.com
export IDP_GITOPS_REPO=git@github.com:<owner>/final-idp-gitops.git
export IDP_GITOPS_SSH_KEY_FILE=$KEYDIR/gitops_write IDP_GITOPS_READ_SSH_KEY_FILE=$KEYDIR/gitops_read

go build -o bin/idp ./cmd/idp
./bin/idp migrate
./bin/idp import-fixtures              # catalog v1, v2; shop-app v1–v3, reporting-app v1–v2; configuration
./prerequisites/build-push-images.sh localhost:5055
./prerequisites/shared-postgres.sh     # PostgreSQL dùng chung cho definition EXISTING
./prerequisites/kind-internal-cluster.sh   # cụm Kubernetes nội bộ có sẵn (kind + Argo CD), đăng ký vào Secret Store
```

Hai loại nơi triển khai:

- `kind-local` là **cụm nội bộ có sẵn**: platform dựng một lần bằng `kind-internal-cluster.sh` (cụm `idp-internal`); catalog khai báo `k8s-cluster` là `EXISTING` trỏ tới `idpsecret://platform/kind-internal-cluster`. IDP chỉ liên kết, không tạo/xóa cụm. Postgres, Redis của từng app vẫn do deployment tạo trong cụm.
- `aws` là **cloud**: deployment đầu tiên dựng VPC, EKS + Argo CD, Aurora, ElastiCache; gỡ app thì xóa.

Catalog có phiên bản (`fixtures/catalog/v<N>.yaml`, bất biến). Muốn sửa catalog thì thêm file phiên bản mới rồi chạy lại `import-fixtures`; phiên bản đã có bị bỏ qua. Deploy chọn phiên bản catalog (`catalogVersion` trong API, ô chọn trên form, `IDP_CATALOG_VERSION` với `idpctl.sh`); để trống thì dùng phiên bản đang chạy, hoặc của staging khi promote, hoặc mới nhất.

## 4. Chạy

```bash
./bin/idp serve    # http://127.0.0.1:8088 – UI và JSON API
./bin/idp worker   # Deployment Worker (một tiến trình)
```

Theo dõi: UI trang `/deployments/<id>`, hoặc `scripts/idpctl.sh show <id>`; log Terraform ở `var/terraform/workspaces/<resource-instance-id>/terraform.log`.

## 5. Kiểm thử tự động

```bash
go test ./...                                   # unit, gồm pipeline với score-k8s thật
go test -tags integration ./internal/service/   # Postgres thật (idp_test), adapter hạ tầng giả lập
```

## 6. Sự cố thường gặp

| Hiện tượng | Xử lý |
|---|---|
| Deployment kẹt `DEPLOYING` sau khi worker bị kill | `./bin/idp fail-orphaned-job <deployment-id> "worker stopped"` (phục hồi tự động chưa làm – D6) |
| `PLAN_CHANGED` khi confirm | Catalog/config/instance đổi sau khi tạo plan; xem plan dựng lại rồi confirm lại |
| `PLAN_CHANGED_BEFORE_EXECUTION` | Input đổi giữa confirm và lúc worker chạy; tạo deployment mới |
| Argo CD không sync | `kubectl -n argocd get applications.argoproj.io` trong cụm; kiểm tra deploy key đọc và kết nối github.com:22 |
| `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH` | Deploy một phần phải dùng phiên bản catalog đang chạy; muốn đổi thì deploy toàn bộ app |
| Đổi thông tin kết nối cụm nội bộ | Chạy lại `kind-internal-cluster.sh` (ghi đè bản ghi trong Secret Store). Lần deploy sau, output của `k8s-cluster` đổi nên Postgres/Redis được apply lại và workload trên cụm được deploy lại |

## 7. Dọn dẹp

```bash
scripts/idpctl.sh teardown shop-app STAGING kind-local   # rồi confirm id trả về: xóa workload, Postgres/Redis; cụm nội bộ chỉ gỡ liên kết
./prerequisites/kind-internal-cluster.sh destroy         # platform xóa cụm nội bộ khi không còn dùng
```
