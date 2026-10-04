FROM node:22-alpine AS hello
WORKDIR /src
COPY client client
COPY scaffold/apps/hello scaffold/apps/hello
RUN npm ci --prefix client && npm run build --prefix client && npm ci --prefix scaffold/apps/hello && npm run build --prefix scaffold/apps/hello

FROM node:22-alpine AS hub
WORKDIR /src
COPY scaffold/apps.yaml apps.yaml
COPY hub/package.json hub/package-lock.json hub/
RUN npm ci --prefix hub
COPY hub hub
COPY scaffold scaffold
RUN npm run build --prefix hub

FROM golang:1.26-bookworm AS server
WORKDIR /src
COPY server/go.mod server/go.sum server/
RUN cd server && go mod download
COPY server server
RUN cd server && CGO_ENABLED=0 go build -o /bouncer ./cmd/bouncer

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=server /bouncer /usr/local/bin/bouncer
COPY --from=hub /src/hub/dist /opt/bouncer/hub
COPY scaffold /opt/bouncer/scaffold
COPY --from=hello /src/scaffold/apps/hello/dist /opt/bouncer/scaffold/apps/hello/dist
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 755 /usr/local/bin/docker-entrypoint.sh /usr/local/bin/bouncer
ENV SITE_ROOT=/site
ENV HUB_DIR=/opt/bouncer/hub
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
