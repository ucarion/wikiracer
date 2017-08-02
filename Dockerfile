FROM golang:latest

RUN mkdir -p /go/src/github.com/ucarion/wikiracer
ADD . /go/src/github.com/ucarion/wikiracer
WORKDIR /go/src/github.com/ucarion/wikiracer

CMD ["./integration_test.sh"]
