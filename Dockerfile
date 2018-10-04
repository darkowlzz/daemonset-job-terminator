FROM golang:1.10.3
WORKDIR /go/src/github.com/darkowlzz/daemonset-job-terminator/
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -o out ./app

FROM scratch
COPY --from=0 /go/src/github.com/darkowlzz/daemonset-job-terminator/out /
CMD ["/out"]
