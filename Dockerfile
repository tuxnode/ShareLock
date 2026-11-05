FROM golang:1.20-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sharelock-server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 sharelock \
    && mkdir -p /data \
    && chown -R sharelock:sharelock /home/sharelock /data
COPY --from=build /out/sharelock-server /usr/local/bin/sharelock-server

USER sharelock
WORKDIR /home/sharelock

ENV SHARELOCK_DIR=/data \
    SHARELOCK_ADDRESS=0.0.0.0:8080

VOLUME /data
EXPOSE 8080

ENTRYPOINT ["sharelock-server"]
