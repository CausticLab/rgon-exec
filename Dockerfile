FROM alpine:edge
MAINTAINER Eric McNiece <emcniece@gmail.com>

RUN apk add --no-cache ca-certificates

ENV RANCHER_GEN_RELEASE v1.1.0

ADD https://github.com/emcniece/rgon-exec/releases/download/${RANCHER_GEN_RELEASE}/rgon-exec-linux-amd64.tar.gz /tmp/rancher-gen.tar.gz
RUN tar -zxvf /tmp/rgon-exec.tar.gz -C /usr/local/bin \
  && chmod +x /usr/local/bin/rgon-exec

ENTRYPOINT ["/usr/local/bin/rgon-exec"]