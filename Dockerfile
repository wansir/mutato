ARG GO_VERSION=1.23.6
ARG GO_IMAGE=docker.io/golang:${GO_VERSION}

FROM ${GO_IMAGE} AS build
WORKDIR /go/src/mutato
ADD . /go/src/mutato/
RUN make mutato-webhook-server

FROM alpine:3.20.3 AS final

WORKDIR /home/mutato

COPY --link --from=build --chmod=555 /go/src/mutato/bin/cmd/mutato-webhook-server /usr/local/bin/mutato-webhook-server

EXPOSE 8443

CMD [ "mutato-webhook-server" ]
