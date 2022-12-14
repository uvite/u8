FROM golang:1.19-alpine as builder
WORKDIR $GOPATH/src/github.com/uvite/u8
ADD . .
RUN apk --no-cache add git
RUN CGO_ENABLED=0 go install -a -trimpath -ldflags "-s -w -X github.com/uvite/u8/lib/consts.VersionDetails=$(date -u +"%FT%T%z")/$(git describe --tags --always --long --dirty)"

FROM alpine:3.16
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 k6
COPY --from=builder /go/bin/k6 /usr/bin/k6

USER 12345
WORKDIR /home/k6
ENTRYPOINT ["k6"]
