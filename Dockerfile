FROM golang:alpine as builder
COPY ./devctl /devctl
RUN chmod 755 /devctl

FROM scratch

COPY --from=builder /devctl /devctl

ENTRYPOINT ["/devctl"]