FROM scratch

COPY ./devctl /devctl
RUN chmod +x /devctl
ENTRYPOINT ["/devctl"]