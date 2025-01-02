FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o vkftpd ./cmd/vkftpd

FROM alpine:latest

RUN adduser -D ftpuser && \
    mkdir -p /mud /etc/vkftpd && \
    chown -R ftpuser:ftpuser /mud

COPY --from=builder /app/vkftpd /usr/local/bin/

# Default configuration paths
ENV FTP_ROOT_DIR=/mud \
    CONFIG_FILE=/etc/vkftpd/config.json

VOLUME ["/mud", "/etc/vkftpd"]
EXPOSE 21 2121-2130

USER ftpuser
ENTRYPOINT ["vkftpd"]
CMD ["-config", "/etc/vkftpd/config.json"]
