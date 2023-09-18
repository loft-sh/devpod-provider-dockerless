FROM alpine:latest


RUN apk add --update-cache shadow shadow-subids iproute2 && rm -rf /var/cache/apk/*

RUN useradd --shell /bin/sh --home /home/rootless --add-subids-for-system --uid 1000 --password "" rootless && \
    mkdir -p /home/rootless && chown -R rootless:rootless /home/rootless
