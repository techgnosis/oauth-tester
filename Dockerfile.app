FROM scratch

COPY oauth-tester /oauth-tester/oauth-tester

COPY oauth-tester.oauth-tester.svc.cluster.local-key.pem /oauth-tester/key.pem
COPY oauth-tester.oauth-tester.svc.cluster.local.pem /oauth-tester/cert.pem

ENTRYPOINT ["/oauth-tester/oauth-tester"]