DB_DOCKER_COMPOSE := docker compose -f docker-compose.db.yml

db-up:
	@$(DB_DOCKER_COMPOSE) up -d

db-up-build:
	@$(DB_DOCKER_COMPOSE) up --build -d

db-down:
	@$(DB_DOCKER_COMPOSE) down -v

db-logs:
	@$(DB_DOCKER_COMPOSE) logs -f

db-ps:
	@$(DB_DOCKER_COMPOSE) ps

run:
	go run cmd/server/main.go

help:
	@echo ""
	@echo "Available targets:"
	@echo "  make db-up         - Start the database container (detached mode)"
	@echo "  make db-up-build   - Build and start the database container"
	@echo "  make db-down       - Stop and remove the database container and volumes"
	@echo "  make db-logs       - View live database container logs"
	@echo "  make db-ps         - Show running database container(s)"
	@echo "  make run           - Starts the scheduler and HTTP server"
	@echo ""