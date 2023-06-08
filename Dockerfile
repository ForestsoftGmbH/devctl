FROM golang:alpine as builder
COPY ./devctl /devctl
RUN chmod 755 /devctl

FROM scratch

WORKDIR /
COPY --from=builder /devctl /devctl

ENTRYPOINT ["/devctl"]