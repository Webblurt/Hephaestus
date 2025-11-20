package controllers

import (
	"encoding/json"
	"errors"
	services "hephaestus/internal/services"
	utils "hephaestus/internal/utils"
	"net/http"
)

type Controller struct {
	Service services.ServiceInterface
	Cfg     *utils.Config
	log     *utils.Logger
}

func NewController(service services.ServiceInterface, cfg *utils.Config, log *utils.Logger) *Controller {
	return &Controller{
		Service: service,
		Cfg:     cfg,
		log:     log,
	}
}

func fetchAuthorizationHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is missing")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) <= len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("invalid authorization header format")
	}
	token := authHeader[len(bearerPrefix):]

	return token, nil
}

func (c *Controller) withAuth(handler func(w http.ResponseWriter, r *http.Request, token string, id string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := fetchAuthorizationHeader(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		id, err := c.Service.Validate(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		handler(w, r, token, id)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
