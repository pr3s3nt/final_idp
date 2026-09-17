---
id: UC03-RUNBOOK
artifact: operations-runbook
status: current
last_reviewed: 2026-09-17
---

# RUNBOOK – UC-03 Deploy Application

Mọi lệnh chạy trong thư mục `idp/backend/`.

## 1. Yêu cầu máy

| Công cụ | Đã dùng khi kiểm chứng |
|---|---|
| Go | 1.27 |
| Docker | 29.x (có mạng `kind`) |
| kind CLI (chỉ để xem/xóa cụm thủ công) | 0.32 |
| Terraform | 1.9.8 |
| score-k8s | 0.15.0 |
| kubectl, jq, git, ssh | bất kỳ bản gần đây |
| gh (chỉ dùng một lần để lấy token nạp vào Secret Store) | 2.62 |
| aws CLI + credentials (chỉ target `aws`) | 2.35 |

Internet cần tới: registry.terraform.io, argoproj.github.io và rancher.github.io (Helm chart của Argo CD, Fleet), quay.io/ghcr.io/docker.io (image Argo CD, postgres, redis), github.com:22 và api.github.com (delivery repository riêng của từng application).

## 2. Biến môi trường

| Biến | Bắt buộc | Ý nghĩa |
|---|---|---|
| `IDP_SECRET_KEY` | có | Khóa Secret Store; phải giống nhau giữa import, serve, worker |
| `IDP_DATABASE_URL` | không | mặc định `postgres://idp:idp@127.0.0.1:55433/idp?sslmode=disable` |
| `IDP_DELIVERY_REPO_PATTERN` | worker | quy ước đặt tên delivery repository, ví dụ `pr3s3nt/idp-<app>-gitops`; `<app>` được thay bằng tên application |
| `IDP_DELIVERY_BRANCH` | không | nhánh chứa desired state, mặc định `main` |
| `IDP_GIT_HOSTING_TOKEN_SECRET` | không | secret reference của token tạo repo, mặc định `idpsecret://platform/git-hosting-token` |
| `IDP_GIT_HOSTING_API` | không | mặc định `https://api.github.com` |
| `IDP_GIT_HOSTING_SSH_HOST` | không | mặc định `github.com` |
| `IDP_GIT_KNOWN_HOSTS` | không | mặc định `<IDP_DATA_DIR>/delivery/known_hosts`; tự ghi bằng `ssh-keyscan` ở lần dùng đầu |
| `IDP_CD_PROVIDER` | không | hệ thống CD đồng bộ cụm: `fleet` (mặc định) hoặc `argocd`. UC-03 chỉ làm việc với CD abstraction nên đổi giá trị này không đổi luồng deploy |
| `IDP_AWS_ECR_REGISTRY` | import | registry ECR của account, điền vào `eks-cluster.image_registry_mirror`. Phải đặt **trước** `import-fixtures`: phiên bản catalog đã tạo không sửa được |
| `IDP_HEALTH_TIMEOUT` | không | thời gian chờ sync/rollout, mặc định 6m |
| `IDP_LISTEN` | không | mặc định `127.0.0.1:8088` |
| `IDP_FRONTEND_DIR` | không | thư mục bundle React (UC-01) được phục vụ dưới `/ui/`, mặc định `../frontend/dist` |

Mỗi application có một delivery repository riêng: IDP tạo repo theo `IDP_DELIVERY_REPO_PATTERN` ở lần deploy đầu tiên của application, sinh một cặp khóa chỉ dùng cho application đó (khóa ghi cho IDP, khóa đọc cho hệ thống CD — Fleet hoặc Argo CD tùy `IDP_CD_PROVIDER`) và lưu cả hai vào Secret Store. Database chỉ giữ secret reference. Gỡ application khỏi một environment chỉ xóa thư mục của environment đó, repo và cặp khóa được giữ lại.

## 3. Chuẩn bị một lần

```bash
# DB của IDP và test DB
docker run -d --name idp-uc03-db --restart unless-stopped -e POSTGRES_USER=idp -e POSTGRES_PASSWORD=idp -e POSTGRES_DB=idp \
  -p 127.0.0.1:55433:5432 postgres:17-alpine
docker exec idp-uc03-db psql -U idp -d idp -c "CREATE DATABASE idp_test"

# Registry image cho target kind-local (đóng vai registry của CI)
docker run -d --name idp-uc03-registry --restart unless-stopped -p 127.0.0.1:5055:5000 registry:2
docker network connect kind idp-uc03-registry   # tạo mạng bằng `docker network create kind` nếu chưa có

# Khóa Secret Store: giữ trong idp/backend/.env (0600, đã gitignore) để không mất giữa các phiên
umask 077; printf 'IDP_SECRET_KEY=%s\n' "$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')" > .env
set -a; . ./.env; set +a

go build -o bin/idp ./cmd/idp

# Token tạo delivery repository: cần quyền tạo repo private và gắn deploy key.
# Token đi thẳng từ file vào Secret Store, không qua biến môi trường hay log.
umask 077; T=$(mktemp); trap 'rm -f "$T"' EXIT
printf '%s' "$(gh auth token)" > "$T"        # hoặc dán một fine-grained token có Administration: write
./bin/idp secret-put platform/git-hosting-token "$T"

export IDP_AWS_ECR_REGISTRY=<account>.dkr.ecr.ap-southeast-1.amazonaws.com
export IDP_DELIVERY_REPO_PATTERN='pr3s3nt/idp-<app>-gitops'   # nháy đơn: `<app>` là ký tự chuyển hướng của shell

./bin/idp migrate
./bin/idp import-fixtures              # catalog v1, v2; shop-app v1–v3, reporting-app v1–v2; configuration
./prerequisites/build-push-images.sh localhost:5055
./prerequisites/shared-postgres.sh     # PostgreSQL dùng chung cho definition EXISTING
./prerequisites/kind-internal-cluster.sh   # cụm Kubernetes nội bộ có sẵn (kind + Argo CD + Fleet), đăng ký vào Secret Store
```

Hai loại nơi triển khai:

- `kind-local` là **cụm nội bộ có sẵn**: platform dựng một lần bằng `kind-internal-cluster.sh` (cụm `idp-internal`); catalog khai báo `k8s-cluster` là `EXISTING` trỏ tới `idpsecret://platform/kind-internal-cluster`. IDP chỉ liên kết, không tạo/xóa cụm. Postgres, Redis của từng app vẫn do deployment tạo trong cụm.
- `aws` là **cloud**: deployment đầu tiên dựng VPC, EKS + Argo CD, Aurora, ElastiCache; gỡ app thì xóa. Target này vẫn dùng Argo CD; Fleet mới chỉ chạy trên `kind-local`.

Catalog có phiên bản (`fixtures/catalog/v<N>.yaml`, bất biến). Muốn sửa catalog thì thêm file phiên bản mới rồi chạy lại `import-fixtures`; phiên bản đã có bị bỏ qua. Deploy chọn phiên bản catalog (`catalogVersion` trong API, ô chọn trên form, `IDP_CATALOG_VERSION` với `idpctl.sh`); để trống thì dùng phiên bản đang chạy, hoặc của staging khi promote, hoặc mới nhất.

## 4. Chạy

```bash
./bin/idp serve    # http://127.0.0.1:8088 – UI và JSON API
./bin/idp worker   # Deployment Worker (một tiến trình)
```

### Web frontend (UC-01)

Editor UC-01 là ứng dụng React trong `idp/frontend/` ([ADR-017](../../decisions/ADR-017-react-web-frontend.md)). Các trang Go của UC-03 đến UC-05 không cần bước này.

```bash
# Build một lần; `idp serve` phục vụ bundle tại http://127.0.0.1:8088/ui/applications
(cd ../frontend && npm ci && npm run build)
./bin/idp serve

# Hoặc khi phát triển giao diện: Vite tại http://127.0.0.1:5173/ui/applications, proxy /api sang `idp serve`
(cd ../frontend && npm run dev)
```

Chưa build thì `/ui/` trả 404 kèm hướng dẫn; các trang khác vẫn chạy.

### Chạy bằng Docker

Image chứa sẵn `terraform`, `score-k8s`, AWS CLI, Git và OpenSSH; cùng một image chạy được server, worker và các lệnh quản trị:

```bash
docker build -t final-idp:local .

# Ví dụ DB chạy trên host Linux; thay URL/secret theo môi trường thực tế.
docker run --rm --network host -v idp-data:/data \
  -e IDP_SECRET_KEY -e IDP_DATABASE_URL \
  final-idp:local migrate
docker run --rm --network host -v idp-data:/data \
  -e IDP_SECRET_KEY -e IDP_DATABASE_URL -e IDP_DELIVERY_REPO_PATTERN \
  final-idp:local serve
docker run --rm --network host -v idp-data:/data \
  -e IDP_SECRET_KEY -e IDP_DATABASE_URL -e IDP_DELIVERY_REPO_PATTERN \
  final-idp:local worker
```

Server là lệnh mặc định và lắng nghe `0.0.0.0:8088`. Image chưa chứa bundle React: muốn dùng `/ui/` thì build `idp/frontend`, mount `dist/` vào container và đặt `IDP_FRONTEND_DIR`. Worker dùng cùng volume `/data` để chia sẻ Secret Store, checkout delivery repository, Terraform state và provider cache. Khi chạy target AWS, truyền thêm AWS credentials theo cơ chế chuẩn của AWS CLI/SDK.

Theo dõi: UI trang `/deployments/<id>`, hoặc `scripts/idpctl.sh show <id>`; log Terraform ở `var/terraform/workspaces/<resource-instance-id>/terraform.log`.

## 5. Kiểm thử tự động

```bash
go test ./...                                   # unit, gồm pipeline với score-k8s thật
go test -tags integration ./internal/service/   # Postgres thật (idp_test), adapter hạ tầng giả lập
(cd ../frontend && npm run lint && npm test && npm run build)             # web frontend UC-01
```

## 6. Sự cố thường gặp

| Hiện tượng | Xử lý |
|---|---|
| Deployment kẹt `DEPLOYING` sau khi worker bị kill | `./bin/idp fail-orphaned-job <deployment-id> "worker stopped"` (phục hồi tự động chưa làm – D6) |
| `PLAN_CHANGED` khi confirm | Catalog/config/instance đổi sau khi tạo plan; xem plan dựng lại rồi confirm lại |
| `PLAN_CHANGED_BEFORE_EXECUTION` | Input đổi giữa confirm và lúc worker chạy; tạo deployment mới |
| Argo CD không sync | `kubectl -n argocd get applications.argoproj.io` trong cụm; kiểm tra deploy key đọc và kết nối github.com:22 |
| Fleet không sync | `kubectl -n fleet-local get gitrepo` xem `COMMIT` và cột trạng thái; `kubectl -n fleet-local get gitrepo <app>-<env> -o jsonpath='{.status.summary}'`. Báo `Modified` nghĩa là object trong cụm lệch Git — thường do manifest giành một nhãn mà Fleet/Helm tự đặt (xem `idp.dev/managed-by` trong `manifest/pipeline.go`) |
| `PARTIAL_DEPLOYMENT_CATALOG_VERSION_MISMATCH` | Deploy một phần phải dùng phiên bản catalog đang chạy; muốn đổi thì deploy toàn bộ app |
| Đổi thông tin kết nối cụm nội bộ | Chạy lại `kind-internal-cluster.sh` (ghi đè bản ghi trong Secret Store). Lần deploy sau, output của `k8s-cluster` đổi nên Postgres/Redis được apply lại và workload trên cụm được deploy lại |

## 7. Dọn dẹp

```bash
scripts/idpctl.sh teardown shop-app STAGING kind-local   # rồi confirm id trả về: xóa workload, Postgres/Redis; cụm nội bộ chỉ gỡ liên kết
./prerequisites/kind-internal-cluster.sh destroy         # platform xóa cụm nội bộ khi không còn dùng
```
