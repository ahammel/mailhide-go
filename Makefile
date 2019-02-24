handler.zip: ./cmd/main.go
	GOOS=linux go build cmd/main.go
	zip handler.zip ./main

