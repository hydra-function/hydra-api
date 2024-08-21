run:
	go run main.go

setup: 
	go mod tidy

run-db:
	@docker-compose up -d $(SVC_DB)