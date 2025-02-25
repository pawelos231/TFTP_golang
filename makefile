.PHONY: run

run_server: 
	go run server/main.go
	
run_client:
	go run client/main.go