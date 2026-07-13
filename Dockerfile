FROM alpine:3.24
ARG TARGETPLATFORM
RUN adduser -D -H -u 10001 zirric
COPY $TARGETPLATFORM/zirric /usr/bin/zirric
USER 10001
ENTRYPOINT ["/usr/bin/zirric"]
