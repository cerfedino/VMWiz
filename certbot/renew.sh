#! /bin/sh

certbot certonly \
--text \
--email 'jmaier@sos.ethz.ch' \
--agree-tos \
--non-interactive \
--server https://acme-v02.api.letsencrypt.org/directory \
--standalone \
--cert-name app.vsos.ethz.ch -d app.vsos.ethz.ch
