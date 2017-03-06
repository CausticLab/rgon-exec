FROM alpine:edge
MAINTAINER Eric McNiece <emcniece@gmail.com>

RUN apk add --no-cache ca-certificates

ENV RGON_EXEC_RELEASE v1.1.0

ADD https://github.com/CausticLab/rgon-exec/releases/download/${RGON_EXEC_RELEASE}/rgon-exec-linux-amd64.tar.gz /tmp/rgon-exec.tar.gz
RUN tar -zxvf /tmp/rgon-exec.tar.gz -C /usr/local/bin \
  && mv /usr/local/bin/rgon-exec-linux-amd64 /usr/local/bin/rgon-exec \
  && chmod +x /usr/local/bin/rgon-exec \
  && rm /tmp/rgon-exec.tar.gz

ENTRYPOINT ["/usr/local/bin/rgon-exec"]