package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"leave-management-backend/internal/models"
)

var adminHTTPClient = &http.Client{Timeout: 10 * time.Second}

func GetAdminUsers(ctx context.Context) ([]models.AdminUser, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT u.id, COALESCE(e.employee_id,''), u.full_name, u.email, u.role::text,
		       COALESCE(e.designation,''), COALESCE(e.department_id::text,''),
		       COALESCE(d.name,''), COALESCE(u.manager_id::text,''), COALESCE(m.full_name,'')
		FROM users u
		LEFT JOIN employees e ON e.user_id=u.id
		LEFT JOIN departments d ON d.id=e.department_id
		LEFT JOIN users m ON m.id=u.manager_id
		WHERE u.role IN ('employee','manager')
		ORDER BY u.role DESC, u.full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.AdminUser
	for rows.Next() {
		var item models.AdminUser
		if err := rows.Scan(&item.UserID, &item.EmployeeID, &item.Name, &item.Email, &item.Role,
			&item.Designation, &item.DepartmentID, &item.Department, &item.ManagerID, &item.ManagerName); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetDepartments(ctx context.Context) ([]models.Department, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `SELECT id, name FROM departments ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Department
	for rows.Next() {
		var item models.Department
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetAdminBalances(ctx context.Context) ([]models.AdminBalance, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT u.id, e.employee_id, lt.id, lt.name, EXTRACT(YEAR FROM CURRENT_DATE)::int,
		       COALESCE(lb.total_allocated, lt.max_days_per_year),
		       COALESCE(lb.used, 0),
		       COALESCE(lb.remaining, lt.max_days_per_year)
		FROM employees e
		JOIN users u ON u.id=e.user_id AND u.role='employee'
		CROSS JOIN leave_types lt
		LEFT JOIN leave_balances lb
		  ON lb.employee_id=e.id AND lb.leave_type_id=lt.id
		 AND lb.year=EXTRACT(YEAR FROM CURRENT_DATE)::int
		ORDER BY u.full_name, lt.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.AdminBalance
	for rows.Next() {
		var item models.AdminBalance
		if err := rows.Scan(&item.UserID, &item.EmployeeID, &item.LeaveTypeID, &item.LeaveType,
			&item.Year, &item.TotalAllocated, &item.Used, &item.Remaining); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func UpdateAdminBalance(ctx context.Context, input models.AdminBalanceInput) error {
	if err := validateAdminBalance(input); err != nil {
		return err
	}
	conn, err := db()
	if err != nil {
		return err
	}
	result, err := conn.ExecContext(ctx, `
		INSERT INTO leave_balances(employee_id, leave_type_id, year, total_allocated, used, remaining)
		SELECT e.id, $2, $3, $4, $5, $4-$5
		FROM employees e JOIN users u ON u.id=e.user_id
		WHERE u.id=$1 AND u.role='employee'
		ON CONFLICT(employee_id, leave_type_id, year) DO UPDATE
		SET total_allocated=EXCLUDED.total_allocated,
		    used=EXCLUDED.used,
		    remaining=EXCLUDED.remaining`,
		input.UserID, input.LeaveTypeID, input.Year, input.TotalAllocated, input.Used)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return clientError(404, "employee not found")
	}
	return nil
}

func validateAdminBalance(input models.AdminBalanceInput) error {
	if input.UserID == "" || input.LeaveTypeID == "" {
		return clientError(400, "employee and leave type are required")
	}
	if input.Year < 2000 || input.Year > 2100 {
		return clientError(400, "year must be between 2000 and 2100")
	}
	if input.TotalAllocated < 0 || input.Used < 0 {
		return clientError(400, "allocated and used days cannot be negative")
	}
	if input.Used > input.TotalAllocated {
		return clientError(400, "used days cannot exceed allocated days")
	}
	return nil
}

func CreateAdminUser(ctx context.Context, input models.AdminUserInput) (string, error) {
	if err := validateAdminUser(input, true); err != nil {
		return "", err
	}
	payload := map[string]any{
		"email": input.Email, "password": input.Password, "email_confirm": true,
		"app_metadata":  map[string]string{"role": input.Role, "employee_id": input.EmployeeID},
		"user_metadata": map[string]string{"full_name": input.Name, "designation": input.Designation},
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := supabaseAdminRequest(ctx, http.MethodPost, "/auth/v1/admin/users", payload, &created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", internalError("unable to create account", nil)
	}
	if err := updateApplicationUser(ctx, created.ID, input); err != nil {
		_ = supabaseAdminRequest(ctx, http.MethodDelete, "/auth/v1/admin/users/"+created.ID, nil, nil)
		return "", err
	}
	return created.ID, nil
}

func UpdateAdminUser(ctx context.Context, userID string, input models.AdminUserInput) error {
	if userID == "" {
		return clientError(400, "user ID is required")
	}
	if err := validateAdminUser(input, false); err != nil {
		return err
	}
	conn, err := db()
	if err != nil {
		return err
	}
	var currentRole string
	if err = conn.QueryRowContext(ctx, `SELECT role::text FROM users WHERE id=$1`, userID).Scan(&currentRole); err != nil {
		return clientError(404, "account not found")
	}
	if currentRole != input.Role {
		return clientError(409, "account role cannot be changed after creation")
	}
	authPayload := map[string]any{
		"email":         input.Email,
		"user_metadata": map[string]string{"full_name": input.Name, "designation": input.Designation},
		"app_metadata":  map[string]string{"role": input.Role, "employee_id": input.EmployeeID},
	}
	if input.Password != "" {
		if len(input.Password) < 8 {
			return clientError(400, "password must be at least 8 characters")
		}
		authPayload["password"] = input.Password
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := updateApplicationUserTx(ctx, tx, userID, input); err != nil {
		return err
	}
	if err := supabaseAdminRequest(ctx, http.MethodPut, "/auth/v1/admin/users/"+userID, authPayload, nil); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return internalError("account was updated in authentication but profile synchronization failed", err)
	}
	return nil
}

func updateApplicationUser(ctx context.Context, userID string, input models.AdminUserInput) error {
	conn, err := db()
	if err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = updateApplicationUserTx(ctx, tx, userID, input); err != nil {
		return err
	}
	return tx.Commit()
}

func updateApplicationUserTx(ctx context.Context, tx *sql.Tx, userID string, input models.AdminUserInput) error {
	var err error
	var department any
	if input.DepartmentID != "" {
		department = input.DepartmentID
	}
	var manager any
	if input.ManagerID != "" {
		if input.ManagerID == userID {
			return clientError(400, "a user cannot manage themselves")
		}
		var valid bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='manager')`, input.ManagerID).Scan(&valid); err != nil {
			return err
		}
		if !valid {
			return clientError(400, "selected manager does not exist or does not have the manager role")
		}
		manager = input.ManagerID
	}
	// Supabase may create the Auth user before app_metadata is visible to the
	// profile trigger, which temporarily provisions an employee profile. A new
	// manager is management-only, so remove that empty profile before changing
	// the role. Existing employee-to-manager role changes are rejected earlier.
	if input.Role == "manager" {
		if _, err = tx.ExecContext(ctx, `DELETE FROM employees WHERE user_id=$1`, userID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `
		UPDATE users SET full_name=$2, email=$3, role=$4, manager_id=$5
		WHERE id=$1`, userID, input.Name, input.Email, input.Role, manager); err != nil {
		return err
	}
	if input.Role == "employee" {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO employees(user_id, employee_id, designation, joining_date, department_id)
			VALUES($1,$2,$3,CURRENT_DATE,$4)
			ON CONFLICT(user_id) DO UPDATE
			SET employee_id=EXCLUDED.employee_id, designation=EXCLUDED.designation,
			    department_id=EXCLUDED.department_id`,
			userID, input.EmployeeID, input.Designation, department); err != nil {
			return err
		}
	}
	return nil
}

func validateAdminUser(input models.AdminUserInput, creating bool) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(input.Email)
	if input.Name == "" || input.Email == "" {
		return clientError(400, "name and email are required")
	}
	if input.Role != "employee" && input.Role != "manager" {
		return clientError(400, "role must be employee or manager")
	}
	if creating && len(input.Password) < 8 {
		return clientError(400, "password must be at least 8 characters")
	}
	if input.Role == "employee" && (strings.TrimSpace(input.EmployeeID) == "" || strings.TrimSpace(input.Designation) == "") {
		return clientError(400, "employee ID and designation are required for employees")
	}
	if input.Role == "manager" && input.ManagerID != "" {
		return clientError(400, "a manager cannot be assigned to another manager in this assessment scope")
	}
	return nil
}

func supabaseAdminRequest(ctx context.Context, method, path string, payload any, result any) error {
	baseURL := strings.TrimRight(firstNonEmpty(os.Getenv("SUPABASE_URL"), os.Getenv("VITE_SUPABASE_URL")), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if baseURL == "" || serviceKey == "" {
		return unavailableError("Supabase Admin API is not configured", nil)
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+serviceKey)
	request.Header.Set("apikey", serviceKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := adminHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return &Error{Status: 502, Message: "identity provider request failed", Err: fmt.Errorf("Supabase Admin API: %s", strings.TrimSpace(string(message)))}
	}
	if result != nil {
		return json.NewDecoder(response.Body).Decode(result)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
