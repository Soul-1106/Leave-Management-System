package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"leave-management-backend/internal/models"
	"leave-management-backend/internal/services"
)

func AdminUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := services.GetAdminUsers(r.Context())
		respond(w, users, err)
	case http.MethodPost:
		var input models.AdminUserInput
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		id, err := services.CreateAdminUser(r.Context(), input)
		respondAdmin(w, http.StatusCreated, map[string]string{"id": id}, err)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func AdminUser(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPatch) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/")
	if id == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}
	var input models.AdminUserInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	err := services.UpdateAdminUser(r.Context(), id, input)
	respondAdmin(w, http.StatusOK, map[string]string{"id": id}, err)
}

func AdminDepartments(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	departments, err := services.GetDepartments(r.Context())
	respond(w, departments, err)
}

func AdminBalances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		balances, err := services.GetAdminBalances(r.Context())
		respond(w, balances, err)
	case http.MethodPatch:
		var input models.AdminBalanceInput
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		err := services.UpdateAdminBalance(r.Context(), input)
		respondAdmin(w, http.StatusOK, map[string]string{"status": "updated"}, err)
	default:
		w.Header().Set("Allow", "GET, PATCH")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func respondAdmin(w http.ResponseWriter, successStatus int, value any, err error) {
	if err != nil {
		respond(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(successStatus)
	_ = json.NewEncoder(w).Encode(value)
}
