package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"leave-management-backend/internal/database"
	"leave-management-backend/internal/middleware"
	"leave-management-backend/internal/models"
	"leave-management-backend/internal/services"
)

func Readiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if database.DB == nil || database.DB.PingContext(r.Context()) != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func respond(w http.ResponseWriter, value any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		status, message := services.PublicError(err)
		log.Printf("request_failed method=%s status=%d error=%q", statusText(status), status, err)
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

func statusText(status int) string {
	return strings.ReplaceAll(strings.ToLower(http.StatusText(status)), " ", "_")
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetStats(r.Context(), identity)
	respond(w, data, err)
}

func GetMyLeaves(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var input models.CreateLeaveRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		identity, _ := middleware.IdentityFrom(r)
		id, err := services.CreateLeave(r.Context(), identity.EmployeeID, input)
		respond(w, map[string]string{"id": id, "status": "pending"}, err)
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetLeaves(r.Context(), identity.EmployeeID)
	respond(w, data, err)
}

func DeleteMyLeave(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/leaves/my/"), "/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "leave ID is required", http.StatusBadRequest)
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	err := services.DeletePendingLeave(r.Context(), identity.EmployeeID, id)
	respond(w, map[string]string{"id": id, "status": "deleted"}, err)
}

func GetApprovals(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetManagerLeaves(r.Context(), identity)
	respond(w, data, err)
}

func GetApprovalHistory(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetManagerLeaveHistory(r.Context(), identity)
	respond(w, data, err)
}

func DecideLeave(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPatch) {
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/leaves/"), "/decision")
	var input struct {
		Status string `json:"status"`
	}
	if id == "" || json.NewDecoder(r.Body).Decode(&input) != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	err := services.DecideLeave(r.Context(), id, strings.ToLower(input.Status), identity)
	respond(w, map[string]string{"id": id, "status": input.Status}, err)
}

func GetMe(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	respond(w, identity, nil)
}

func GetEmployees(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetEmployees(r.Context(), identity)
	respond(w, data, err)
}

func GetLeaveBalances(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	data, err := services.GetBalances(r.Context(), identity)
	respond(w, data, err)
}
