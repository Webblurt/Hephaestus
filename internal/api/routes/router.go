package routes

import (
	"errors"
	"net/http"

	controllers "hephaestus/internal/api/controllers"
	services "hephaestus/internal/services"
	utils "hephaestus/internal/utils"
)

func CreateRoutes(service services.ServiceInterface, cfg *utils.Config, log *utils.Logger) (http.Handler, error) {
	if service == nil {
		return nil, errors.New("service is nil")
	}

	domains := controllers.NewController(service, cfg, log)

	mux := http.NewServeMux()

	mux.Handle("/hephaestus/api/v1/domains", methodRouter(map[string]http.HandlerFunc{
		http.MethodGet:    domains.HandleGetDomains(),
		http.MethodPost:   domains.HandleCreateDomain(),
		http.MethodDelete: domains.HandleDeleteDomain(),
	}))

	return mux, nil
}

func methodRouter(routes map[string]http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h, ok := routes[r.Method]; ok {
			h(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}
