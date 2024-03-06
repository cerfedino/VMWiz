FROM certbot/certbot:latest

RUN mkdir /renew
WORKDIR /renew
EXPOSE 80

COPY renew.sh /renew/renew.sh
COPY crontab /etc/crontab

RUN echo "CERTBOT_EMAIL=${CERTBOT_EMAIL}" > /renew/.env && \
    echo "CERTBOT_DOMAIN=${CERTBOT_DOMAIN}" >> /renew/.env

ENTRYPOINT sh
CMD ["cron", "-f"]
