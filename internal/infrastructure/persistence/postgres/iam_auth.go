package postgres

import (
	"context"
	"fmt"
	"net/url"

	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

// BuildIAMDSN generates an IAM auth token and returns a DSN suitable for pgxpool.
// - writer: the Aurora writer endpoint, e.g. cluster-xxxx.rds.amazonaws.com
// - db: the database name (e.g., "promptshield")
// - user: the database user granted rds_iam (e.g., "ps_app")
// - region: AWS region (e.g., "us-east-1")
func BuildIAMDSN(ctx context.Context, writer, db, user, region string) (string, error) {
	if writer == "" || db == "" || user == "" || region == "" {
		return "", fmt.Errorf("missing required parameters for IAM DSN")
	}

	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("load aws config: %w", err)
	}

	token, err := auth.BuildAuthToken(ctx, writer+":5432", region, user, cfg.Credentials)
	if err != nil {
		return "", fmt.Errorf("build auth token: %w", err)
	}

	userEsc := url.QueryEscape(user)
	passEsc := url.QueryEscape(token)
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=require", userEsc, passEsc, writer, db), nil
}
