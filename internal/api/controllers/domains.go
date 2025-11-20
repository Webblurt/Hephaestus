package controllers

import (
	"encoding/json"
	models "hephaestus/internal/models"
	utils "hephaestus/internal/utils"
	"net/http"
)

func (c *Controller) HandleGetDomains() http.HandlerFunc {
	return c.withAuth(func(w http.ResponseWriter, r *http.Request, token string, userid string) {
		query := r.URL.Query()
		filters := models.GetDomainsReq{
			Status:     query.Get("status"),
			DomainName: query.Get("domain_name"),
			PageSize:   utils.GetDefaultIntegerQueryValue(query, "page_size", 10),
			Page:       utils.GetDefaultIntegerQueryValue(query, "page", 1),
		}
		filters.UserID = userid

		domains, err := c.Service.GetDomains(filters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, domains)
	})
}

func (c *Controller) HandleCreateDomain() http.HandlerFunc {
	return c.withAuth(func(w http.ResponseWriter, r *http.Request, token string, userid string) {
		var req models.CreateDomainReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.CreatedBy = userid

		domainID, err := c.Service.CreateDomain(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Domain created successfully", "domain_id": domainID})
	})
}

func (c *Controller) HandleDeleteDomain() http.HandlerFunc {
	return c.withAuth(func(w http.ResponseWriter, r *http.Request, token string, userid string) {
		query := r.URL.Query()
		domID := query.Get("domain_id")
		domName := query.Get("domain_name")
		if domID == "" && domName == "" {
			http.Error(w, "missing domain_id or domain_name", http.StatusBadRequest)
			return
		}
		var filters models.DeleteDomainReq
		if domID != "" {
			filters.DomainID = domID
		}
		if domName != "" {
			filters.DomainName = domName
		}
		filters.UserID = userid

		err := c.Service.DeleteDomain(filters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Domain deleted successfully"})
	})
}
