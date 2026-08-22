FROM alpine:3.18

# Install necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Set the working directory to /libredesk
WORKDIR /libredesk

# Copy necessary files
COPY libredesk .
COPY config.sample.toml config.toml
# 新增这一行：把i18n整个目录复制进镜像
COPY ./i18n ./i18n

# Expose port 9000 for the application
EXPOSE 9000

# Set the default command to run the libredesk binary
CMD ["./libredesk"]