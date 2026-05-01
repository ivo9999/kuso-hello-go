# kuso-hello-go

Minimal Go HTTP server used to smoke-test the kuso build pipeline. Echoes
hello + a few env vars (so we can verify addon connection-secret injection).

Self-contained: `docker build .` works locally without any prior steps.
