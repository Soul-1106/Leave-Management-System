package services

import (
	"testing"

	"leave-management-backend/internal/models"
)

func TestValidateAdminUser(t *testing.T) {
	valid := models.AdminUserInput{
		Name: "Sarah Employee", Email: "sarah@example.com", Password: "temporary-password",
		Role: "employee", EmployeeID: "EMP-100", Designation: "Engineer",
	}
	if err := validateAdminUser(valid, true); err != nil {
		t.Fatalf("valid employee was rejected: %v", err)
	}

	valid.Password = "short"
	if err := validateAdminUser(valid, true); err == nil {
		t.Fatal("expected a short password to be rejected")
	}

	valid.Password = "temporary-password"
	valid.Role = "admin"
	if err := validateAdminUser(valid, true); err == nil {
		t.Fatal("expected direct admin creation to be rejected")
	}
}

func TestValidateAdminBalance(t *testing.T) {
	valid := models.AdminBalanceInput{
		UserID: "user", LeaveTypeID: "annual", Year: 2026,
		TotalAllocated: 20, Used: 5,
	}
	if err := validateAdminBalance(valid); err != nil {
		t.Fatalf("valid balance was rejected: %v", err)
	}
	valid.Used = 21
	if err := validateAdminBalance(valid); err == nil {
		t.Fatal("expected used days above allocation to be rejected")
	}
}
