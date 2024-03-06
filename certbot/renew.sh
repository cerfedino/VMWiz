#! /bin/sh

source /renew/.env

certbot certonly \
--text \
--email "$CERTBOT_EMAIL" \
--agree-tos \
--non-interactive \
--server https://acme-v02.api.letsencrypt.org/directory \
--webroot \
--webroot-path /var/www/letsencrypt \
--cert-name "$CERTBOT_DOMAIN" -d "$CERTBOT_DOMAIN"
