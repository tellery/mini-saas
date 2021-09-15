FROM golang:1.16 AS builder
WORKDIR /build
ADD . .
ENV CGO_ENABLED=0
RUN go build -o xds-server cmd/xds-server/main.go


FROM golang:1.16-alpine
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/xds-server ./xds-server
ENTRYPOINT ["./xds-server"]
