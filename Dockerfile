FROM scratch

COPY ./devctl /devctl

ENTRYPOINT ["/devctl"]