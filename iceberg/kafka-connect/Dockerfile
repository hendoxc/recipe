FROM confluentinc/cp-kafka-connect:latest

ENV CONNECT_PLUGIN_PATH: "/usr/share/java,/usr/share/confluent-hub-components"

RUN confluent-hub install --no-prompt tabular/iceberg-kafka-connect:0.6.5
