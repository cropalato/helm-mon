# syntax=docker/dockerfile:1


##
## Build
##
FROM golang:1.16-alpine As build
ENV CGO_ENABLED 0

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN apk add --no-cache libc6-compat build-base

RUN go build -gcflags "all=-N -l" -o /helm-mon

##
## Deploy
##
FROM golang:1.16-alpine

WORKDIR /

COPY --from=build /helm-mon /helm-mon

RUN apk add --no-cache strace

EXPOSE 8080

CMD [ "/helm-mon" ]
