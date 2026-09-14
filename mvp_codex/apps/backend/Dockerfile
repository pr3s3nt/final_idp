FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY apps/backend ./apps/backend
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/backend ./apps/backend

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /out/backend /backend
EXPOSE 8080
ENTRYPOINT ["/backend"]
