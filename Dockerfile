# Run linter(s). 
FROM docker.io/golangci/golangci-lint:latest as linter

WORKDIR /src/hello-zedcloud

COPY go.mod go.sum main.go version ./

# NOTE: If this stage doesn't produce any files that are then used in the
# final container image then it will be optimized out (won't be run).
RUN golangci-lint run -v server.go | tee ./linter.logs

# Build the server executable.
FROM docker.io/golang:alpine as builder

WORKDIR /src/hello-zedcloud

COPY go.mod go.sum main.go version ./

ENV GOTOOLCHAIN=local
RUN go build -o hello-zedcloud

# Build the final container image.
FROM docker.io/alpine

WORKDIR /

COPY --from=linter /src/hello-zedcloud/linter.logs /var/cache/hello-zedcloud-linter.logs 
COPY --from=builder /src/hello-zedcloud/hello-zedcloud /hello-zedcloud

COPY ./static /var/www/static

CMD ["/hello-zedcloud", "-static", "/var/www/static"]
