FROM gcr.io/distroless/base:latest
LABEL maintainer="mgencur@redhat.com"

ADD creds-init /usr/bin/creds-init
ENTRYPOINT ["/usr/bin/creds-init"]
