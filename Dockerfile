FROM ubuntu AS base

WORKDIR /app

EXPOSE 3030

RUN apt-get update && apt-get install -y supervisor && apt-get install -y logrotate

COPY --chmod=0644 deployment/logrotate          /etc/logrotate.d/item-composition-service
COPY deployment/supervisord.conf                /etc/supervisor/supervisord.conf
COPY --chmod=0755 deployment/pre_start.sh       /opt/bin/pre_start

COPY --chmod=0755 cmd/app/main                  /opt/bin/item-composition-service
COPY api                                        /app/api
COPY proto                                      /app/proto
COPY config/config.yaml                         /etc/item-composition-service/config.yaml

RUN /opt/bin/pre_start

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/supervisord.conf"]
