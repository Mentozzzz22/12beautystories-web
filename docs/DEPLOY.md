# Deployment

One VPS runs everything with Docker Compose:

```
Internet ──► Caddy :443 (HTTPS, auto certificates) ──► app :8080 (Go API + web app) ──► Postgres
                                                        │
                                              uploads volume ◄── backup (daily pg_dump + photos)
```

Only Caddy is exposed. The database and the app are reachable inside the Docker network only.

## 1. Server

- Ubuntu 24.04, 2 GB RAM, 20+ GB disk. A Kazakhstan provider (ps.kz, hoster.kz) keeps
  customers' personal data in the country as local law requires.
- Point the domain's DNS `A` records for `@` and `www` to the server IP before the first start,
  otherwise Caddy cannot obtain certificates.

```bash
# as root on a fresh server
apt update && apt upgrade -y
curl -fsSL https://get.docker.com | sh
adduser deploy && usermod -aG docker deploy
ufw allow OpenSSH && ufw allow 80 && ufw allow 443 && ufw enable
```

## 2. Application

```bash
su - deploy
git clone https://github.com/Mentozzzz22/12beautystories-web.git shop
cd shop
cp .env.example .env
```

Edit `.env`:

| Variable | Value |
|---|---|
| `DOMAIN` | `12-beautystories.com` |
| `POSTGRES_USER` / `POSTGRES_DB` | `shop` |
| `POSTGRES_PASSWORD` | long random string: `openssl rand -base64 24` |
| `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID` | optional, see below |

The other variables in `.env.example` are for local development; the production compose file sets
its own values (`COOKIE_SECURE=true`, `TRUST_PROXY=true`, `PUBLIC_URL=https://$DOMAIN`).

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps        # app should become "healthy"
```

Create the owner's admin account (the password is read from the environment):

```bash
docker compose -f docker-compose.prod.yml exec -e ADMIN_PASSWORD='…' app /app/admin create -email owner@example.com
```

The site is live at `https://$DOMAIN`, the admin panel at `https://$DOMAIN/admin`.

## 3. Telegram notifications (optional)

1. In Telegram open @BotFather, send `/newbot`, copy the token.
2. Create a group (for example "Заказы 12 Beauty Stories"), add the owner and the bot.
3. Send any message in the group, then open
   `https://api.telegram.org/bot<TOKEN>/getUpdates` and copy `chat.id` (a negative number for groups).
4. Put both values into `.env` and run `docker compose -f docker-compose.prod.yml up -d`.

## 4. Updates

```bash
cd ~/shop
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

Migrations run automatically when the app starts.

## 5. Backups

The `backup` service writes to `~/shop/backups` every 24 hours and keeps 14 days:

- `db_<date>.dump` – PostgreSQL custom-format dump
- `uploads_<date>.tar.gz` – product and banner photos

Copy them off the server regularly (another machine or cloud storage), a backup on the same disk
does not survive a lost server.

Restore:

```bash
docker compose -f docker-compose.prod.yml stop app
docker compose -f docker-compose.prod.yml exec -T db pg_restore --clean --if-exists -U shop -d shop < backups/db_<date>.dump
# the volume is named <folder>_uploads, i.e. shop_uploads for a checkout in ~/shop
docker run --rm -v shop_uploads:/uploads -v "$PWD/backups:/backups" alpine \
  tar -xzf /backups/uploads_<date>.tar.gz -C /uploads
docker compose -f docker-compose.prod.yml start app
```

## Logs

```bash
docker compose -f docker-compose.prod.yml logs -f app      # JSON logs, one line per request
docker compose -f docker-compose.prod.yml logs -f caddy
docker compose -f docker-compose.prod.yml logs backup
```
