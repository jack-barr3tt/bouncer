FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS node
RUN apk add --no-cache nodejs npm

FROM node AS hello
WORKDIR /src
COPY client/package.json client/package-lock.json client/tsconfig.json client/
COPY client/src client/src
RUN npm ci --prefix client && npm run build --prefix client
COPY scaffold/apps/hello scaffold/apps/hello
RUN npm ci --prefix scaffold/apps/hello \
  && rm -rf scaffold/apps/hello/node_modules/@jack-barr3tt/bouncer-client \
  && mkdir -p scaffold/apps/hello/node_modules/@jack-barr3tt/bouncer-client \
  && cp client/package.json scaffold/apps/hello/node_modules/@jack-barr3tt/bouncer-client/package.json \
  && cp -R client/dist scaffold/apps/hello/node_modules/@jack-barr3tt/bouncer-client/dist \
  && npm run build --prefix scaffold/apps/hello

FROM node AS hub
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

FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS builder
COPY --from=server /bouncer /usr/local/bin/bouncer
ENV WORKSPACE=/workspace
EXPOSE 8081
ENTRYPOINT ["bouncer", "builder"]

FROM node
RUN apk add --no-cache git openssh-client docker-cli docker-cli-compose
COPY --from=server /bouncer /usr/local/bin/bouncer
WORKDIR /opt/bouncer/hub
COPY hub/package.json hub/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=hub /src/hub/dist ./dist
COPY scaffold /opt/bouncer/scaffold
COPY --from=hello /src/scaffold/apps/hello/dist /opt/bouncer/scaffold/apps/hello/dist
COPY deploy/docker-compose.yml /opt/bouncer/docker-compose.yml
COPY deploy/stack.yml /opt/bouncer/stack.yml
COPY deploy/env.yaml /opt/bouncer/env.yaml
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 755 /usr/local/bin/docker-entrypoint.sh /usr/local/bin/bouncer
ARG VERSION=dev
ENV BOUNCER_VERSION=$VERSION
ENV SITE_ROOT=/site
ENV HUB_UPSTREAM=http://127.0.0.1:3000
ENV LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["docker-entrypoint.sh"]
