FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps/frontend ./apps/frontend
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/frontend ./apps/frontend

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /out/frontend /frontend
EXPOSE 8080
ENTRYPOINT ["/frontend"]
