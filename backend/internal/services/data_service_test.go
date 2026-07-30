package services

import (
	"testing"

	"leave-management-backend/internal/models"
)

func TestValidateLeaveRequestRejectsCrossYear(t *testing.T) {
	input := models.CreateLeaveRequest{
		LeaveType: "Annual Leave",
		StartDate: "2026-12-30",
		EndDate:   "2027-01-02",
		Reason:    "Holiday",
	}
	if err := validateLeaveRequest(input); err == nil {
		t.Fatal("expected a cross-year request to be rejected")
	}
}

func TestValidateLeaveRequestAcceptsSingleYear(t *testing.T) {
	input := models.CreateLeaveRequest{
		LeaveType: "Annual Leave",
		StartDate: "2026-08-10",
		EndDate:   "2026-08-12",
		Reason:    "Holiday",
	}
	if err := validateLeaveRequest(input); err != nil {
		t.Fatalf("valid request was rejected: %v", err)
	}
}

func TestValidateLeaveDeletionAllowsOnlyPendingRequests(t *testing.T) {
	if err := validateLeaveDeletion("pending"); err != nil {
		t.Fatalf("pending request was rejected: %v", err)
	}
	for _, status := range []string{"approved", "rejected"} {
		if err := validateLeaveDeletion(status); err == nil {
			t.Fatalf("%s request should not be deletable", status)
		}
	}
}
