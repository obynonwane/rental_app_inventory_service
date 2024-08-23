FROM --platform=linux/amd64 alpine:latest

RUN mkdir /app

COPY inventoryApp /app

CMD [ "/app/inventoryApp" ]
