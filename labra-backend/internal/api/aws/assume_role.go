package aws

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type AssumeRoleInput struct {
	RoleARN    string
	ExternalID string
	Region     string
}

type AssumeRoleVerifier interface {
	Verify(ctx context.Context, in AssumeRoleInput) (accountID string, err error)
}

type LocalAssumeRoleVerifier struct{}

var roleARNPattern = regexp.MustCompile(`^arn:aws:iam::([0-9]{12}):role\/[A-Za-z0-9+=,.@_\/-]+$`)

func (LocalAssumeRoleVerifier) Verify(_ context.Context, in AssumeRoleInput) (string, error) {
	roleARN := strings.TrimSpace(in.RoleARN)
	externalID := strings.TrimSpace(in.ExternalID)
	region := strings.TrimSpace(in.Region)

	if roleARN == "" {
		return "", fmt.Errorf("role_arn is required")
	}
	matches := roleARNPattern.FindStringSubmatch(roleARN)
	if len(matches) != 2 {
		return "", fmt.Errorf("role_arn must match arn:aws:iam::<account-id>:role/<role-name>")
	}
	if len(externalID) < 8 || len(externalID) > 128 {
		return "", fmt.Errorf("external_id must be between 8 and 128 characters")
	}
	if region == "" {
		return "", fmt.Errorf("region is required")
	}

	return matches[1], nil
}
