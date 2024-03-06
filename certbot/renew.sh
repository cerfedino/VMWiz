#! /bin/sh

certbot certonly \
--text \
--email "$CERTBOT_EMAIL" \
--agree-tos \
--non-interactive \
--server https://acme-v02.api.letsencrypt.org/directory \
--standalone \
--cert-name "$CERTBOT_DOMAIN" -d "$CERTBOT_DOMAIN"
