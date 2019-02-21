FROM ubuntu:bionic as build

ENV GOPATH /go

RUN apt-get update \
    && apt-get install -y golang-glide golang-go mercurial git \
    && mkdir -p $GOPATH/src/github.com/flexshopper/newrelic-custom-metrics

COPY . $GOPATH/src/github.com/flexshopper/newrelic-custom-metrics

RUN cd $GOPATH/src/github.com/flexshopper/newrelic-custom-metrics \
    && glide install -v \
    && echo "Building adapter..." \
    && CGO_ENABLED=0 GOARCH=amd64 go build -o /tmp/adapter .

FROM ubuntu:bionic

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=build /tmp/adapter /

CMD ["/adapter"]