FROM golang:1.26 AS go-builder

WORKDIR /src

COPY go.mod .
COPY main.go .
COPY config/ ./config/
COPY handler/ ./handler/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /audyn .


FROM python:3.12-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends ffmpeg \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /streamrip
COPY streamrip/ .
RUN pip install --no-cache-dir .

WORKDIR /app
COPY --from=go-builder /audyn /usr/local/bin/audyn
COPY config/ ./config/

COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

RUN mkdir -p /downloads/incomplete /downloads/complete

ENV AUDYN_INCOMPLETE_PATH=/downloads/incomplete
ENV AUDYN_COMPLETE_PATH=/downloads/complete
ENV AUDYN_COMPLETE_PATH_MAPPING=/downloads/complete
ENV AUDYN_PORT=5000
# DEEZER_ARL
# ENV DEEZER_ARL=

EXPOSE 5000

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["audyn"]
