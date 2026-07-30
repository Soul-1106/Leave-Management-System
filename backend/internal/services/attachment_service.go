package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"leave-management-backend/internal/middleware"
	"leave-management-backend/internal/models"
)

const attachmentBucket = "leave-attachments"

var allowedAttachmentTypes = map[string]string{
	"application/pdf": ".pdf",
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
}

func UploadLeaveAttachment(ctx context.Context, identity middleware.Identity, leaveID, originalName, contentType string, data []byte) error {
	extension, ok := allowedAttachmentTypes[contentType]
	if !ok {
		return clientError(400, "only PDF, JPEG, and PNG attachments are allowed")
	}
	if len(data) == 0 || len(data) > 5*1024*1024 {
		return clientError(400, "attachment must be between 1 byte and 5 MB")
	}
	conn, err := db()
	if err != nil {
		return err
	}
	var currentPath string
	err = conn.QueryRowContext(ctx, `
		SELECT COALESCE(l.attachment_path,'')
		FROM leaves l
		JOIN employees e ON e.id=l.employee_id
		WHERE l.id=$1 AND e.employee_id=$2 AND l.status='pending'`,
		leaveID, identity.EmployeeID).Scan(&currentPath)
	if err != nil {
		return clientError(404, "pending leave request not found")
	}
	randomID, err := attachmentID()
	if err != nil {
		return err
	}
	objectPath := path.Join(identity.UserID, leaveID, randomID+extension)
	if err = storageRequest(ctx, http.MethodPost, objectPath, contentType, bytes.NewReader(data), nil); err != nil {
		return err
	}
	result, err := conn.ExecContext(ctx, `
		UPDATE leaves
		SET attachment_path=$2, attachment_name=$3, attachment_type=$4, attachment_size=$5
		WHERE id=$1`,
		leaveID, objectPath, originalName, contentType, len(data))
	if err != nil {
		_ = storageRequest(ctx, http.MethodDelete, objectPath, "", nil, nil)
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		_ = storageRequest(ctx, http.MethodDelete, objectPath, "", nil, nil)
		return clientError(404, "leave request not found")
	}
	if currentPath != "" && currentPath != objectPath {
		_ = storageRequest(ctx, http.MethodDelete, currentPath, "", nil, nil)
	}
	return nil
}

func GetLeaveAttachmentLink(ctx context.Context, identity middleware.Identity, leaveID string) (models.AttachmentLink, error) {
	conn, err := db()
	if err != nil {
		return models.AttachmentLink{}, err
	}
	var result models.AttachmentLink
	var objectPath string
	query := `
		SELECT COALESCE(l.attachment_path,''), COALESCE(l.attachment_name,''),
		       COALESCE(l.attachment_type,''), COALESCE(l.attachment_size,0)
		FROM leaves l
		JOIN employees e ON e.id=l.employee_id
		JOIN users employee_user ON employee_user.id=e.user_id
		WHERE l.id=$1`
	args := []any{leaveID}
	if identity.Role != "admin" {
		query += ` AND employee_user.manager_id=$2`
		args = append(args, identity.UserID)
	}
	if err = conn.QueryRowContext(ctx, query, args...).Scan(&objectPath, &result.Name, &result.Type, &result.Size); err != nil || objectPath == "" {
		return models.AttachmentLink{}, clientError(404, "attachment not found or access denied")
	}
	var signed struct {
		SignedURL string `json:"signedURL"`
	}
	payload, _ := json.Marshal(map[string]int{"expiresIn": 120})
	if err = storageRequest(ctx, http.MethodPost, "sign/"+objectPath, "application/json", bytes.NewReader(payload), &signed); err != nil {
		return models.AttachmentLink{}, err
	}
	if signed.SignedURL == "" {
		return models.AttachmentLink{}, fmt.Errorf("storage did not return a signed URL")
	}
	baseURL := strings.TrimRight(firstNonEmpty(os.Getenv("SUPABASE_URL"), os.Getenv("VITE_SUPABASE_URL")), "/")
	if strings.HasPrefix(signed.SignedURL, "http") {
		result.URL = signed.SignedURL
	} else {
		result.URL = baseURL + "/storage/v1" + signed.SignedURL
	}
	return result, nil
}

func storageRequest(ctx context.Context, method, objectPath, contentType string, body io.Reader, result any) error {
	baseURL := strings.TrimRight(firstNonEmpty(os.Getenv("SUPABASE_URL"), os.Getenv("VITE_SUPABASE_URL")), "/")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if baseURL == "" || serviceKey == "" {
		return unavailableError("Supabase Storage is not configured", nil)
	}
	endpoint := baseURL + "/storage/v1/object/"
	if strings.HasPrefix(objectPath, "sign/") {
		endpoint += "sign/" + attachmentBucket + "/" + escapeObjectPath(strings.TrimPrefix(objectPath, "sign/"))
	} else {
		endpoint += attachmentBucket + "/" + escapeObjectPath(objectPath)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+serviceKey)
	request.Header.Set("apikey", serviceKey)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if method == http.MethodPost && !strings.Contains(endpoint, "/sign/") {
		request.Header.Set("x-upsert", "true")
	}
	response, err := adminHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return &Error{Status: 502, Message: "attachment storage request failed", Err: fmt.Errorf("Supabase Storage: %s", strings.TrimSpace(string(message)))}
	}
	if result != nil {
		return json.NewDecoder(response.Body).Decode(result)
	}
	return nil
}

func escapeObjectPath(value string) string {
	segments := strings.Split(value, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

func attachmentID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
