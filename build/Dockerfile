FROM alpine:3.16

ENV \
  PATH="/app/mediarr:${PATH}" \
  MEDIARR_CONFIG_DIR="/config"

# Binary
COPY ["dist/build_linux_linux_amd64/mediarr", "/app/mediarr/mediarr"]

VOLUME ["/config"]

ENTRYPOINT ["/app/mediarr/mediarr"]
