run:
	go build ./cmd/main
	./main

run-docker:
	docker build --tag 'crypto_ws' .
	docker run -p 8080:8080 'crypto_ws'