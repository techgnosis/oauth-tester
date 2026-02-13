FROM golang:1.25 AS builder
WORKDIR /app
COPY internal/ ./internal/
COPY go.mod .
COPY go.sum .
COPY main.go .
COPY vendor/ ./vendor/
RUN CGO_ENABLED=0 GOOS=linux go build -o oauth-tester --mod=vendor .

FROM scratch
WORKDIR /oauth-tester
COPY --from=builder /app/oauth-tester .
COPY oauth-tester.oauth-tester.svc.cluster.local-key.pem ./key.pem
COPY oauth-tester.oauth-tester.svc.cluster.local.pem ./cert.pem


CMD ["/oauth-tester/oauth-tester"]