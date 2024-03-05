.PHONY: minio
minio:
	@echo "\n✅ Starting Minio\n"
	docker compose -f docker-compose.yml up -d minio mc

.PHONY: rest-catalog
rest-catalog:
	@echo "\n✅ Starting REST Catalog\n"
	docker compose -f docker-compose.yml up -d rest postgres

.PHONY: trino
trino:
	@echo "\n✅ Starting Trino\n"
	docker compose -f docker-compose.yml up -d trino

.PHONY: kafka
kafka:
	@echo "\n✅ Starting Kafka\n"
	docker compose -f docker-compose.yml up kafka-1 kafka-2 kafka-3 schema-registry kafka-connect connect-ui schema-registry-ui
