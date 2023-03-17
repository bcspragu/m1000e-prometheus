FROM golang:1.20-alpine as build

WORKDIR /build

RUN apk add --update --no-cache git ca-certificates tzdata && update-ca-certificates

COPY ./go.mod ./go.sum ./

RUN go mod download && go mod verify

COPY main.go ./
COPY racadm/ ./racadm/
COPY ipmi/ ./ipmi/

RUN go test ./... && GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/server

FROM scratch

WORKDIR /app

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/server /app/server

ENTRYPOINT ["/app/server"]
