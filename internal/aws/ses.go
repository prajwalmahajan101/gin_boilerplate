package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SES sends transactional email from a fixed verified sender address.
type SES struct {
	client *sesv2.Client
	from   string
}

// NewSES builds an SES helper that sends as from.
func NewSES(cfg aws.Config, from string) *SES {
	return &SES{client: sesv2.NewFromConfig(cfg), from: from}
}

// SendEmail sends a plain-text email to a single recipient.
func (s *SES) SendEmail(ctx context.Context, to, subject, body string) error {
	_, err := s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: &s.from,
		Destination:      &types.Destination{ToAddresses: []string{to}},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: &subject},
				Body:    &types.Body{Text: &types.Content{Data: &body}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ses: send to %q: %w", to, err)
	}
	return nil
}
