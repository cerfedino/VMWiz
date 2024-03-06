FROM certbot/certbot:latest

RUN mkdir /renewal
WORKDIR /renewal
EXPOSE 80

COPY renew.sh /renewal
COPY crontab /etc/crontab

ENTRYPOINT sh
CMD ["cron", "-f"]
