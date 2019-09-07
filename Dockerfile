FROM golang:1.12.0-alpine3.9 as builder

LABEL maintainer="Ben Tucker <ben_tucker@hotmail.com"
RUN apk update && apk upgrade && apk add --no-cache git dep

WORKDIR $GOPATH/src/github.com/bluebenno/RDS-snapshot-copier
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only

COPY . ./

WORKDIR $GOPATH/src/github.com/bluebenno/RDS-snapshot-copier/cmd/rds-snapshot-copier
RUN CGO_ENABLED=0 GOOS=linux go build -o /app .


FROM alpine
COPY --from=builder /app ./
ENTRYPOINT ["./app"]
