# Local development

This file is for working on this repository. Deploying a site that uses the published image is in [README.md](README.md).

Run `make` from the repository root to list targets. Copy `.env.example` to `.env` there before starting. The server reads that file from the root, and leaves a variable that is already set. A relative `SITE_ROOT` is resolved from that same directory.

`make dev` starts Postgres and a tmux session named `bouncer`. The `server` window runs this checkout on `http://127.0.0.1:8080`. The `hub` window is the homepage dev server, and it proxies `/api` and `/apps` to that port. Open the URL Vite prints. If the session already exists, `make dev` attaches to it.

In the session, switch windows with `Ctrl-b` then `n` or `p`, or `Ctrl-b` then `0` or `1` (`server`, `hub`). `Ctrl-b` then `d` detaches and leaves both running. `make stop-tmux` kills the session. `make down` stops Compose. `make stop` does both. `make server` and `make hub` run those processes without tmux.

Postgres listens on `127.0.0.1:5436`. The user, password, and database are `bouncer`. The first start of an empty database creates the admin from `AUTH_BOOTSTRAP_USERNAME` and `AUTH_BOOTSTRAP_PASSWORD`. Sign in with those, then change the password. The server applies migrations as it starts. `make migrate` applies them on their own.

`SITE_ROOT` in the example is `../apps`, a directory next to this repository. Point it at the site you want to serve. `make dev` writes `apps.yaml` and the Hello sample into a site that does not already have `apps.yaml`, and builds Hello when `dist` is missing. A site that already has `apps.yaml` is left alone. The container entrypoint copies the same sample, including the Hello build from the image.

`docker compose up` runs Postgres, the builder, and `ghcr.io/jack-barr3tt/bouncer`. That server is the published image, so it does not include changes in this checkout.

The homepage lists the apps that account is allowed to open. It loads them from `GET /api/apps`. An admin is allowed every app in `apps.yaml`.

`make dev` runs the hub with Vite, which renders each page on the server. Port 8080 proxies hub paths to `HUB_UPSTREAM` (`http://127.0.0.1:5173` unless you set it). `npm start` in `hub/` is the production process, on port 3000.

`make builder` is the git-deploy builder on port 8081. Set `BUILDER_TOKEN` in `.env`. Leave it stopped when you are not exercising deploy.

`make generate` rewrites the server API from `server/openapi.yaml` and the hub client from that file. `make schema` refreshes `server/db/schema.sql` after a migration change.

`make lint` runs golangci-lint on the server and oxlint on the JavaScript and TypeScript. `make check` runs that, then the client, create, script, server, hub, and Hello checks. Server tests that talk to Postgres need `TEST_DATABASE_URL`. Point that at a database you can wipe. The tests truncate its tables.
