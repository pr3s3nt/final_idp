# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.1
ARG TERRAFORM_VERSION=1.9.8
ARG AWS_CLI_VERSION=2.35.21

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations

ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/idp ./cmd/idp

FROM debian:bookworm-slim AS score-k8s
ARG SCORE_K8S_VERSION=0.15.0
ARG TARGETARCH
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates curl \
    && case "$TARGETARCH" in \
         amd64) checksum="7477feebc76dfec256b28d8eef9e5aca420c48bbeabc1ee120073a359214e8d7" ;; \
         arm64) checksum="882d14a7e3358c6d0bfa2c537f31363522b2140a2c5ef966d3f66cb493decbc6" ;; \
         *) echo "unsupported architecture: $TARGETARCH" >&2; exit 1 ;; \
       esac \
    && archive="score-k8s_${SCORE_K8S_VERSION}_linux_${TARGETARCH}.tar.gz" \
    && curl -fsSLo "/tmp/$archive" "https://github.com/score-spec/score-k8s/releases/download/${SCORE_K8S_VERSION}/${archive}" \
    && echo "$checksum  /tmp/$archive" | sha256sum -c - \
    && tar -xzf "/tmp/$archive" -C /usr/local/bin score-k8s

FROM hashicorp/terraform:${TERRAFORM_VERSION} AS terraform
FROM public.ecr.aws/aws-cli/aws-cli:${AWS_CLI_VERSION} AS aws-cli

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install --no-install-recommends -y ca-certificates git openssh-client \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 idp \
    && useradd --uid 10001 --gid idp --create-home --home-dir /data/home idp \
    && install -d -o idp -g idp /app /data

COPY --from=build /out/idp /usr/local/bin/idp
COPY --from=score-k8s /usr/local/bin/score-k8s /usr/local/bin/score-k8s
COPY --from=terraform /bin/terraform /usr/local/bin/terraform
COPY --from=aws-cli /usr/local/aws-cli /usr/local/aws-cli
RUN ln -s /usr/local/aws-cli/v2/current/bin/aws /usr/local/bin/aws

WORKDIR /app
COPY --chown=idp:idp fixtures ./fixtures
COPY --chown=idp:idp terraform/modules ./terraform/modules

ENV HOME=/data/home \
    IDP_DATA_DIR=/data \
    IDP_FIXTURES_DIR=/app/fixtures \
    IDP_MODULES_DIR=/app/terraform/modules \
    IDP_LISTEN=0.0.0.0:8088

USER idp
EXPOSE 8088
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/idp"]
CMD ["serve"]
