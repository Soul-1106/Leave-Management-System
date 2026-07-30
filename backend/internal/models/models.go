package models

type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Delta string `json:"delta"`
	Tone  string `json:"tone"`
}

type Leave struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Status   string `json:"status"`
	Dates    string `json:"dates,omitempty"`
	Reason   string `json:"reason"`
	Approver string `json:"approver,omitempty"`
	Days     int    `json:"days,omitempty"`
}

type Approval struct {
	LeaveID        string `json:"leaveId"`
	Name           string `json:"name"`
	ID             string `json:"id"`
	Role           string `json:"role"`
	Dept           string `json:"dept"`
	Leave          string `json:"leave"`
	Dates          string `json:"dates"`
	Reason         string `json:"reason"`
	Requested      string `json:"requested"`
	Days           int    `json:"days"`
	Status         string `json:"status"`
	HasAttachment  bool   `json:"hasAttachment"`
	AttachmentName string `json:"attachmentName,omitempty"`
}

type AttachmentLink struct {
	URL  string `json:"url"`
	Name string `json:"name"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

type CreateLeaveRequest struct {
	LeaveType string `json:"leaveType"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Reason    string `json:"reason"`
}

type Employee struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Role  string `json:"role"`
	Dept  string `json:"dept"`
	Email string `json:"email,omitempty"`
}

type AdminUser struct {
	UserID       string `json:"userId"`
	EmployeeID   string `json:"employeeId,omitempty"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Designation  string `json:"designation,omitempty"`
	DepartmentID string `json:"departmentId,omitempty"`
	Department   string `json:"department,omitempty"`
	ManagerID    string `json:"managerId,omitempty"`
	ManagerName  string `json:"managerName,omitempty"`
}

type AdminUserInput struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Password     string `json:"password,omitempty"`
	Role         string `json:"role"`
	EmployeeID   string `json:"employeeId,omitempty"`
	Designation  string `json:"designation,omitempty"`
	DepartmentID string `json:"departmentId,omitempty"`
	ManagerID    string `json:"managerId,omitempty"`
}

type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AdminBalance struct {
	UserID         string `json:"userId"`
	EmployeeID     string `json:"employeeId"`
	LeaveTypeID    string `json:"leaveTypeId"`
	LeaveType      string `json:"leaveType"`
	Year           int    `json:"year"`
	TotalAllocated int    `json:"totalAllocated"`
	Used           int    `json:"used"`
	Remaining      int    `json:"remaining"`
}

type AdminBalanceInput struct {
	UserID         string `json:"userId"`
	LeaveTypeID    string `json:"leaveTypeId"`
	Year           int    `json:"year"`
	TotalAllocated int    `json:"totalAllocated"`
	Used           int    `json:"used"`
}

type LeaveBalance struct {
	Label string `json:"label"`
	Used  int    `json:"used"`
	Total int    `json:"total"`
	Color string `json:"color"`
}
