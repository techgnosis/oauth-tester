FROM scratch

COPY oauth-tester /oauth-tester

COPY oauth-tester.oauth-tester.svc.cluster.local-key.pem /key.pem
COPY oauth-tester.oauth-tester.svc.cluster.local.pem /cert.pem

ENTRYPOINT ["/oauth-tester"]