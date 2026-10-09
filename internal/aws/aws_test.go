package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// emailSender mirrors the interface the auth module depends on; this asserts
// *SES satisfies that shape without importing auth (no cycle).
type emailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

func TestSES_ConstructsAndConforms(t *testing.T) {
	var _ emailSender = NewSES(aws.Config{}, "noreply@example.com")
}

func TestS3_Constructs(t *testing.T) {
	if NewS3(aws.Config{}, "bucket") == nil {
		t.Fatal("NewS3 returned nil")
	}
}
