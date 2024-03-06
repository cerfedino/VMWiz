FROM certbot/certbot:latest

RUN mkdir /renew
WORKDIR /renew
EXPOSE 80

COPY renew.sh /renew/renew.sh
COPY store-env.sh /renew/store-env.sh
COPY crontab /etc/crontab

RUN mkdir -p /var/www/letsencrypt/.well-known/acme-challenge/

ENTRYPOINT ["sh"]
CMD ["-c", "/renew/store-env.sh && /renew/renew.sh && crond -f"]
