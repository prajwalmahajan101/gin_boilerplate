package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func loadCloudSecrets(ctx context.Context) error {
	secretID := os.Getenv("SECRETS_MANAGER_SECRET_ID")
	if secretID == "" {
		return nil
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-south-1"
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("config: aws config for secrets manager: %w", err)
	}

	client := secretsmanager.NewFromConfig(awsCfg)
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &secretID})
	if err != nil {
		return fmt.Errorf("config: get secret %q: %w", secretID, err)
	}
	if out.SecretString == nil {
		return fmt.Errorf("config: secret %q has no string value", secretID)
	}

	var bundle map[string]string
	if err := json.Unmarshal([]byte(*out.SecretString), &bundle); err != nil {
		return fmt.Errorf("config: secret %q is not a flat JSON object: %w", secretID, err)
	}

	for k, v := range bundle {
		if _, present := os.LookupEnv(k); present {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("config: setenv %q from secret bundle: %w", k, err)
		}
	}
	return nil
}
