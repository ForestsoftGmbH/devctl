FROM golang:alpine as builder
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /usr/src

COPY . /usr/src/
RUN go build -ldflags="-w -s" -o devctl -v
RUN chmod 755 /usr/src/devctl
RUN go test -v ./...
RUN ls -al .


FROM scratch

WORKDIR /
COPY --from=builder /usr/src/devctl /

ENTRYPOINT ["/devctl"]