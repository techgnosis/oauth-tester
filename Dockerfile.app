FROM golang:1.25 AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY vendor/ ./vendor/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --mod=vendor .

FROM scratch
WORKDIR /oauth-tester
COPY --from=builder /app/oauth-tester ./oauth-tester
COPY oauth-tester.oauth-tester.svc.cluster.local-key.pem ./key.pem
COPY oauth-tester.oauth-tester.svc.cluster.local.pem ./cert.pem


CMD ["/oauth-tester/oauth-tester"]