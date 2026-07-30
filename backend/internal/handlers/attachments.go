package handlers

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"leave-management-backend/internal/middleware"
	"leave-management-backend/internal/services"
)

const maxAttachmentBytes = 5 * 1024 * 1024

func UploadAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	leaveID := attachmentLeaveID(r.URL.Path)
	if leaveID == "" {
		http.Error(w, "leave ID is required", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentBytes+1024*1024)
	if err := r.ParseMultipartForm(maxAttachmentBytes); err != nil {
		http.Error(w, "attachment must be 5 MB or smaller", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("attachment")
	if err != nil {
		http.Error(w, "attachment file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAttachmentBytes+1))
	if err != nil || len(data) > maxAttachmentBytes {
		http.Error(w, "attachment must be 5 MB or smaller", http.StatusBadRequest)
		return
	}
	contentType := http.DetectContentType(data)
	name := strings.TrimSpace(filepath.Base(header.Filename))
	if name == "" {
		name = "attachment"
	}
	if len(name) > 255 {
		name = name[:255]
	}
	identity, _ := middleware.IdentityFrom(r)
	err = services.UploadLeaveAttachment(r.Context(), identity, leaveID, name, contentType, data)
	respond(w, map[string]string{"status": "uploaded", "name": name}, err)
}

func GetAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	leaveID := attachmentLeaveID(r.URL.Path)
	if leaveID == "" {
		http.Error(w, "leave ID is required", http.StatusBadRequest)
		return
	}
	identity, _ := middleware.IdentityFrom(r)
	link, err := services.GetLeaveAttachmentLink(r.Context(), identity, leaveID)
	respond(w, link, err)
}

func attachmentLeaveID(requestPath string) string {
	return strings.Trim(strings.TrimPrefix(requestPath, "/api/attachments/"), "/")
}
