package main

import (
	routes "hephaestus/internal/api/routes"
	clients "hephaestus/internal/clients"
	repositories "hephaestus/internal/repositories"
	services "hephaestus/internal/services"
	utils "hephaestus/internal/utils"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// loading .env
	_ = godotenv.Load()
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable is not set")
	}

	//loading configuration
	cfg, err := utils.LoadConfig(configPath)
	if err != nil {
		log.Fatal("Error loading config file", err)
	}

	// creating logger
	log := utils.NewLogger(cfg.Logger.LogLevel)

	// repository creation
	repo, err := repositories.NewRepository(cfg, log)
	if err != nil {
		log.Fatal("Error creating repository: ", err)
	}
	log.Info("Repository created successful")

	// start migrations
	if err := repo.RunMigrations(cfg); err != nil {
		log.Warn("Error running migrations: ", err)
	}
	log.Info("Migrations applied successfully")

	// creating clients for external apis
	clientsList, err := clients.CreateClients(cfg, log)
	if err != nil {
		log.Fatal("Error creating clients: ", err)
	}
	log.Info("Clients created successful")

	// creating service
	service, err := services.NewService(cfg, clientsList, repo, log)
	if err != nil {
		log.Fatal("Error creating service: ", err)
	}
	log.Info("Service created successful")

	// starting scheduler
	service.StartCertificateRenewalScheduler()
	log.Info("Certificate renewal scheduler started")

	// creating routes
	router, err := routes.CreateRoutes(service, cfg, log)
	if err != nil {
		log.Fatal("Error creating routes: ", err)
	}
	log.Info("Routes created successful")

	// starting http server
	log.Info("Starting the server on port ", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, router); err != nil {
		log.Fatal("Error starting server: ", err)
	}
	log.Info("Server started successful")
}
