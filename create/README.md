# @jack-barr3tt/create-bouncer

Writes the files for a [Bouncer](https://github.com/jack-barr3tt/bouncer) site: CI pipelines, build and deploy scripts, `deploy/docker-compose.yml`, and the Hello sample.

```bash
npx @jack-barr3tt/create-bouncer
```

Run it in the git repository that will hold your apps. It asks which CI to use (GitHub Actions, Woodpecker, or Forgejo), the public URL, the absolute path of that repository on the server, and the host port. The path is saved as `SITE_PATH`. The port is saved as `PORT` and defaults to 8080.

Pass the same answers as flags:

```bash
npx @jack-barr3tt/create-bouncer . \
  --ci github \
  --public-url https://apps.example.com \
  --site-path /var/www/apps \
  --port 8080
```

Existing files are left in place. Clone the repository on the server at the path you entered, then from `deploy/`:

```bash
cp .env.example .env
# set AUTH_BOOTSTRAP_USERNAME and AUTH_BOOTSTRAP_PASSWORD
docker compose up -d
```

The compose file starts a reconciler from the Bouncer image. That container mounts the Docker socket and applies the stack shipped in the image. It pins `IMAGE_TAG` to a published version. A newer version that needs env values which are still blank is left unapplied, and an admin sees the missing keys in Bouncer. The pin moves once those values are set. An admin can turn automatic updates off from the admin page. A compose file that still lists those services itself is rewritten when that server starts. Run `docker compose up -d` once more and commit `deploy/docker-compose.yml`.

How to run the server, add apps, and publish from git is in the [Bouncer README](https://github.com/jack-barr3tt/bouncer#readme).

## License

[MIT](LICENSE)
