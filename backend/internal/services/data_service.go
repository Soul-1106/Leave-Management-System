package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"leave-management-backend/internal/database"
	"leave-management-backend/internal/middleware"
	"leave-management-backend/internal/models"
)

func db() (*sql.DB, error) {
	if database.DB == nil {
		return nil, unavailableError("database is unavailable", nil)
	}
	return database.DB, nil
}

func GetStats(ctx context.Context, identity middleware.Identity) ([]models.Stat, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	if identity.Role == "admin" {
		var employees, pending, approved, rejected, today int
		err = conn.QueryRowContext(ctx, `
			SELECT
				(SELECT count(*) FROM employees),
				count(*) FILTER (WHERE status = 'pending'),
				count(*) FILTER (WHERE status = 'approved'),
				count(*) FILTER (WHERE status = 'rejected'),
				count(*) FILTER (WHERE status = 'approved' AND CURRENT_DATE BETWEEN start_date AND end_date)
			FROM leaves`).Scan(&employees, &pending, &approved, &rejected, &today)
		if err != nil {
			return nil, err
		}
		return []models.Stat{
			{Label: "Total employees", Value: fmt.Sprint(employees), Delta: "Current workforce", Tone: "primary"},
			{Label: "Pending requests", Value: fmt.Sprint(pending), Delta: "Needs review", Tone: "warning"},
			{Label: "Approved requests", Value: fmt.Sprint(approved), Delta: "All time", Tone: "success"},
			{Label: "Rejected requests", Value: fmt.Sprint(rejected), Delta: "All time", Tone: "danger"},
			{Label: "Employees on leave", Value: fmt.Sprint(today), Delta: "Today", Tone: "teal"},
		}, nil
	}
	if identity.Role == "manager" {
		var employees, pending, approved, rejected int
		err = conn.QueryRowContext(ctx, `
			SELECT count(DISTINCT e.id),
			       count(DISTINCT l.id) FILTER (WHERE l.status='pending'),
			       count(DISTINCT l.id) FILTER (WHERE l.status='approved'),
			       count(DISTINCT l.id) FILTER (WHERE l.status='rejected')
			FROM users u
			LEFT JOIN employees e ON e.user_id=u.id
			LEFT JOIN leaves l ON l.employee_id=e.id
			WHERE u.manager_id=$1`, identity.UserID).
			Scan(&employees, &pending, &approved, &rejected)
		if err != nil {
			return nil, err
		}
		return []models.Stat{
			{Label: "My team", Value: fmt.Sprint(employees), Delta: "Assigned employees", Tone: "primary"},
			{Label: "Pending", Value: fmt.Sprint(pending), Delta: "Needs review", Tone: "warning"},
			{Label: "Approved", Value: fmt.Sprint(approved), Delta: "All time", Tone: "success"},
			{Label: "Rejected", Value: fmt.Sprint(rejected), Delta: "All time", Tone: "danger"},
		}, nil
	}

	var pending, approved, rejected int
	err = conn.QueryRowContext(ctx, `
		SELECT count(*) FILTER (WHERE status = 'pending'),
		       count(*) FILTER (WHERE status = 'approved'),
		       count(*) FILTER (WHERE status = 'rejected')
		FROM leaves l JOIN employees e ON e.id=l.employee_id
		WHERE e.employee_id=$1`, identity.EmployeeID).Scan(&pending, &approved, &rejected)
	if err != nil {
		return nil, err
	}
	return []models.Stat{
		{Label: "Pending requests", Value: fmt.Sprint(pending), Delta: "Awaiting review", Tone: "warning"},
		{Label: "Approved leaves", Value: fmt.Sprint(approved), Delta: "All time", Tone: "success"},
		{Label: "Rejected leaves", Value: fmt.Sprint(rejected), Delta: "All time", Tone: "danger"},
	}, nil
}

func GetLeaves(ctx context.Context, employeeID string) ([]models.Leave, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT l.id, lt.name, l.status::text,
		       to_char(l.start_date, 'Mon DD, YYYY') || ' - ' || to_char(l.end_date, 'Mon DD, YYYY'),
		       l.reason, COALESCE(approver.full_name, ''), (l.end_date-l.start_date)+1
		FROM leaves l
		JOIN leave_types lt ON lt.id=l.leave_type_id
		LEFT JOIN users approver ON approver.id=l.approved_by_id
		JOIN employees e ON e.id=l.employee_id
		WHERE e.employee_id=$1
		ORDER BY l.created_at DESC`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Leave
	for rows.Next() {
		var item models.Leave
		if err := rows.Scan(&item.ID, &item.Type, &item.Status, &item.Dates, &item.Reason, &item.Approver, &item.Days); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetManagerLeaves(ctx context.Context, identity middleware.Identity) ([]models.Approval, error) {
	return getManagerLeavesByState(ctx, identity, false)
}

func GetManagerLeaveHistory(ctx context.Context, identity middleware.Identity) ([]models.Approval, error) {
	return getManagerLeavesByState(ctx, identity, true)
}

func getManagerLeavesByState(ctx context.Context, identity middleware.Identity, history bool) ([]models.Approval, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT l.id, u.full_name, e.employee_id, e.designation, COALESCE(d.name,''), lt.name,
		       to_char(l.start_date,'Mon DD, YYYY') || ' - ' || to_char(l.end_date,'Mon DD, YYYY'),
		       l.reason, to_char(l.created_at,'Mon DD, YYYY'), (l.end_date-l.start_date)+1, l.status::text,
		       l.attachment_path IS NOT NULL, COALESCE(l.attachment_name,''),
		       COALESCE(to_char(l.approval_date,'Mon DD, YYYY'), '')
		FROM leaves l
		JOIN employees e ON e.id=l.employee_id
		JOIN users u ON u.id=e.user_id
		LEFT JOIN departments d ON d.id=e.department_id
		JOIN leave_types lt ON lt.id=l.leave_type_id
		WHERE TRUE`
	args := []any{}
	if identity.Role != "admin" {
		query += ` AND u.manager_id=$1`
		args = append(args, identity.UserID)
	}
	if history {
		query += ` AND l.status IN ('approved', 'rejected')`
		query += ` ORDER BY l.approval_date DESC, l.created_at DESC`
	} else {
		query += ` AND l.status='pending'`
		query += ` ORDER BY l.created_at DESC`
	}
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Approval
	for rows.Next() {
		var item models.Approval
		if err := rows.Scan(&item.LeaveID, &item.Name, &item.ID, &item.Role, &item.Dept, &item.Leave, &item.Dates, &item.Reason, &item.Requested, &item.Days, &item.Status, &item.HasAttachment, &item.AttachmentName, &item.DecisionDate); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetEmployees(ctx context.Context, identity middleware.Identity) ([]models.Employee, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT u.full_name, e.employee_id, e.designation, COALESCE(d.name,''), u.email
		FROM employees e JOIN users u ON u.id=e.user_id
		LEFT JOIN departments d ON d.id=e.department_id`
	args := []any{}
	if identity.Role != "admin" {
		query += ` WHERE u.manager_id=$1`
		args = append(args, identity.UserID)
	}
	query += ` ORDER BY u.full_name`
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Employee
	for rows.Next() {
		var item models.Employee
		if err := rows.Scan(&item.Name, &item.ID, &item.Role, &item.Dept, &item.Email); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetBalances(ctx context.Context, identity middleware.Identity) ([]models.LeaveBalance, error) {
	conn, err := db()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT lt.name, sum(lb.used), sum(lb.total_allocated)
		FROM leave_balances lb JOIN leave_types lt ON lt.id=lb.leave_type_id
		JOIN employees e ON e.id=lb.employee_id
		JOIN users u ON u.id=e.user_id
		WHERE lb.year=EXTRACT(YEAR FROM CURRENT_DATE)`
	args := []any{}
	if identity.Role == "employee" {
		query += ` AND e.employee_id=$1`
		args = append(args, identity.EmployeeID)
	} else if identity.Role == "manager" {
		query += ` AND u.manager_id=$1`
		args = append(args, identity.UserID)
	}
	query += ` GROUP BY lt.name ORDER BY lt.name`
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	colors := []string{"teal", "orange", "blue", "purple"}
	var result []models.LeaveBalance
	for rows.Next() {
		var item models.LeaveBalance
		if err := rows.Scan(&item.Label, &item.Used, &item.Total); err != nil {
			return nil, err
		}
		item.Color = colors[len(result)%len(colors)]
		result = append(result, item)
	}
	return result, rows.Err()
}

func CreateLeave(ctx context.Context, employeeID string, input models.CreateLeaveRequest) (string, error) {
	if err := validateLeaveRequest(input); err != nil {
		return "", err
	}
	conn, err := db()
	if err != nil {
		return "", err
	}
	var overlaps bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM leaves l
			JOIN employees e ON e.id=l.employee_id
			WHERE e.employee_id=$1
			  AND l.status IN ('pending','approved')
			  AND daterange(l.start_date, l.end_date, '[]') &&
			      daterange($2::date, $3::date, '[]')
		)`, employeeID, input.StartDate, input.EndDate).Scan(&overlaps)
	if err != nil {
		return "", err
	}
	if overlaps {
		return "", clientError(409, "leave dates overlap an existing pending or approved request")
	}
	var id string
	err = conn.QueryRowContext(ctx, `
		INSERT INTO leaves(employee_id, leave_type_id, start_date, end_date, reason)
		SELECT e.id, lt.id, $2, $3, $4 FROM employees e CROSS JOIN leave_types lt
		WHERE e.employee_id=$1 AND lt.name=$5
		RETURNING id`,
		employeeID, input.StartDate, input.EndDate, input.Reason, input.LeaveType).Scan(&id)
	if err == sql.ErrNoRows {
		return "", clientError(404, "employee or leave type not found")
	}
	return id, err
}

func validateLeaveRequest(input models.CreateLeaveRequest) error {
	start, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return clientError(400, "invalid start date")
	}
	end, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		return clientError(400, "invalid end date")
	}
	if end.Before(start) {
		return clientError(400, "end date cannot be before start date")
	}
	if start.Year() != end.Year() {
		return clientError(400, "leave requests cannot span calendar years")
	}
	if strings.TrimSpace(input.Reason) == "" {
		return clientError(400, "reason is required")
	}
	return nil
}

func DeletePendingLeave(ctx context.Context, employeeID, leaveID string) error {
	conn, err := db()
	if err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status, attachmentPath string
	err = tx.QueryRowContext(ctx, `
		SELECT l.status::text, COALESCE(l.attachment_path, '')
		FROM leaves l
		JOIN employees e ON e.id=l.employee_id
		WHERE l.id=$1 AND e.employee_id=$2
		FOR UPDATE`, leaveID, employeeID).Scan(&status, &attachmentPath)
	if err == sql.ErrNoRows {
		return clientError(404, "leave request not found")
	}
	if err != nil {
		return err
	}
	if err = validateLeaveDeletion(status); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM leaves WHERE id=$1`, leaveID); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	// Remote cleanup is best-effort: a temporary Storage failure must not make
	// a successfully deleted request appear to have failed.
	if attachmentPath != "" {
		if cleanupErr := storageRequest(ctx, http.MethodDelete, attachmentPath, "", nil, nil); cleanupErr != nil {
			log.Printf("attachment_cleanup_failed leave_id=%q error=%q", leaveID, cleanupErr)
		}
	}
	return nil
}

func validateLeaveDeletion(status string) error {
	if status != "pending" {
		return clientError(409, "only pending leave requests can be deleted")
	}
	return nil
}

func DecideLeave(ctx context.Context, id, status string, approver middleware.Identity) error {
	if status != "approved" && status != "rejected" {
		return clientError(400, "status must be approved or rejected")
	}
	conn, err := db()
	if err != nil {
		return err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var employeeID, leaveTypeID, currentStatus string
	var days, year int
	query := `
		SELECT l.employee_id, l.leave_type_id, (l.end_date-l.start_date)+1,
		       EXTRACT(YEAR FROM l.start_date)::int, l.status::text
		FROM leaves l
		JOIN employees e ON e.id=l.employee_id
		JOIN users employee_user ON employee_user.id=e.user_id
		WHERE l.id=$1
	`
	args := []any{id}
	if approver.Role != "admin" {
		query += ` AND employee_user.manager_id=$2`
		args = append(args, approver.UserID)
	}
	query += ` FOR UPDATE`
	err = tx.QueryRowContext(ctx, query, args...).
		Scan(&employeeID, &leaveTypeID, &days, &year, &currentStatus)
	if err == sql.ErrNoRows {
		return clientError(404, "leave request not found or not assigned to this manager")
	}
	if err != nil {
		return err
	}
	if currentStatus != "pending" {
		return clientError(409, "only pending requests can be decided")
	}

	if status == "approved" {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO leave_balances(employee_id, leave_type_id, year, total_allocated, used, remaining)
			SELECT $1, lt.id, $2, lt.max_days_per_year, 0, lt.max_days_per_year
			FROM leave_types lt WHERE lt.id=$3
			ON CONFLICT(employee_id, leave_type_id, year) DO NOTHING`,
			employeeID, year, leaveTypeID)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE leave_balances
			SET used=used+$1, remaining=total_allocated-(used+$1)
			WHERE employee_id=$2 AND leave_type_id=$3 AND year=$4 AND remaining >= $1`,
			days, employeeID, leaveTypeID, year)
		if err != nil {
			return err
		}
		changed, _ := result.RowsAffected()
		if changed == 0 {
			return clientError(409, "insufficient leave balance")
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE leaves SET status=$2, approved_by_id=$3, approval_date=$4
		WHERE id=$1`, id, status, approver.UserID, time.Now())
	if err != nil {
		return err
	}
	return tx.Commit()
}
