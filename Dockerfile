FROM golang:1.15-buster as builder

RUN apt-get update && apt-get install git && apt-get install ca-certificates

WORKDIR /volume-admission-controller

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/volume-admission-controller

# Runtime image
FROM scratch AS base
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/volume-admission-controller /usr/bin/volume-admission-controller
ENTRYPOINT ["/usr/bin/volume-admission-controller"]
