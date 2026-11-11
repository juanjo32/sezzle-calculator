.PHONY: test test-backend test-frontend coverage run-backend run-frontend docker

test: test-backend test-frontend

test-backend:
	cd backend && go test ./... -race

test-frontend:
	cd frontend && npm test

coverage:
	cd backend && go test ./calculator ./api -coverprofile=coverage.out && go tool cover -func=coverage.out && go tool cover -html=coverage.out -o coverage.html
	cd frontend && npm run coverage

run-backend:
	cd backend && go run .

run-frontend:
	cd frontend && npm run dev

docker:
	docker compose up --build
