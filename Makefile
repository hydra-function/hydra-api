run:
	go run main.go

setup: 
	go get
	go mod tidy

run-db:
	@docker-compose up -d $(SVC_DB)