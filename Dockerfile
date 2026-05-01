# Single-stage Go build that copies sources, compiles statically, and
# produces a tiny final image. Self-contained — kuso's kaniko clones,
# this Dockerfile builds, no external dist/ needed.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY main.go ./
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/app .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/app /usr/local/bin/app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
