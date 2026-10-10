FROM node:22-alpine AS hello
WORKDIR /src
COPY scaffold/apps/hello scaffold/apps/hello
RUN npm ci --prefix scaffold/apps/hello && npm run build --prefix scaffold/apps/hello

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

FROM node:22-alpine AS builder
COPY --from=server /bouncer /usr/local/bin/bouncer
ENV WORKSPACE=/workspace
EXPOSE 8081
ENTRYPOINT ["bouncer", "builder"]

FROM node:22-alpine
RUN apk add --no-cache git openssh-client
COPY --from=server /bouncer /usr/local/bin/bouncer
WORKDIR /opt/bouncer/hub
COPY hub/package.json hub/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=hub /src/hub/dist ./dist
COPY scaffold /opt/bouncer/scaffold
COPY --from=hello /src/scaffold/apps/hello/dist /opt/bouncer/scaffold/apps/hello/dist
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 755 /usr/local/bin/docker-entrypoint.sh /usr/local/bin/bouncer
ENV SITE_ROOT=/site
ENV HUB_UPSTREAM=http://127.0.0.1:3000
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
