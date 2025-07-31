migrateup:
	migrate -path db/migrations/ -database "postgresql://appuser:apppass@localhost:5432/messaging?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migrations/ -database "postgresql://appuser:apppass@localhost:5432/messaging?sslmode=disable" -verbose down

sqlc:
	sqlc generate

.PHONY: postgres migrateup migratedown sqlc