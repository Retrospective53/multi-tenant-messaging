migrateup:
	migrate -path db/migrations/ -database "postgresql://user:pass@localhost:5432/app?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migrations/ -database "postgresql://user:pass@localhost:5432/app?sslmode=disable" -verbose down

sqlc:
	sqlc generate

.PHONY: postgres migrateup migratedown sqlc