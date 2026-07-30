package services

import "testing"

func TestEscapeObjectPath(t *testing.T) {
	got := escapeObjectPath("user id/leave-id/report final.pdf")
	want := "user%20id/leave-id/report%20final.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAttachmentTypesAreRestricted(t *testing.T) {
	if _, ok := allowedAttachmentTypes["application/pdf"]; !ok {
		t.Fatal("PDF should be allowed")
	}
	if _, ok := allowedAttachmentTypes["text/html"]; ok {
		t.Fatal("HTML must not be allowed")
	}
}
