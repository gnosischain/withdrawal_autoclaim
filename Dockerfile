FROM golang:1.20-alpine3.18 AS build

WORKDIR /build
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 go build .

#
FROM alpine:latest

WORKDIR /app

COPY --from=build /build/withdrawal_autoclaim .

ENTRYPOINT ["./withdrawal_autoclaim"]