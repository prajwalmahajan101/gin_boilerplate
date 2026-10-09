// Package aws holds thin helpers over the AWS SDK v2 (S3, SES). It centralises
// config loading so every AWS client in the app shares one region/credentials
// resolution path.
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

// Config loads the default AWS config (credentials chain, env, IAM role) pinned
// to region. An empty region falls back to the SDK's own resolution.
func Config(ctx context.Context, region string) (aws.Config, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("aws: load config: %w", err)
	}
	return cfg, nil
}
