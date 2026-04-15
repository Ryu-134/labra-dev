package store

type App struct {
	ID                int64  `json:"id"`
	UserID            int64  `json:"user_id"`
	Name              string `json:"name"`
	RepoFullName      string `json:"repo_full_name"`
	Branch            string `json:"branch"`
	BuildType         string `json:"build_type"`
	OutputDir         string `json:"output_dir"`
	RootDir           string `json:"root_dir,omitempty"`
	SiteURL           string `json:"site_url,omitempty"`
	AutoDeployEnabled bool   `json:"auto_deploy_enabled"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

type Deployment struct {
	ID            int64  `json:"id"`
	AppID         int64  `json:"app_id"`
	UserID        int64  `json:"user_id"`
	Status        string `json:"status"`
	TriggerType   string `json:"trigger_type"`
	CommitSHA     string `json:"commit_sha,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
	CommitAuthor  string `json:"commit_author,omitempty"`
	Branch        string `json:"branch,omitempty"`
	SiteURL       string `json:"site_url,omitempty"`
	FailureReason string `json:"failure_reason,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
	StartedAt     int64  `json:"started_at,omitempty"`
	FinishedAt    int64  `json:"finished_at,omitempty"`
}

type DeploymentLog struct {
	ID           int64  `json:"id"`
	DeploymentID int64  `json:"deployment_id"`
	LogLevel     string `json:"log_level"`
	Message      string `json:"message"`
	CreatedAt    int64  `json:"created_at"`
}

type AWSConnection struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	RoleARN         string `json:"role_arn"`
	ExternalID      string `json:"external_id"`
	Region          string `json:"region"`
	AccountID       string `json:"account_id"`
	Status          string `json:"status"`
	LastValidatedAt int64  `json:"last_validated_at,omitempty"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

type CreateAppInput struct {
	UserID            int64
	Name              string
	RepoFullName      string
	Branch            string
	BuildType         string
	OutputDir         string
	RootDir           string
	SiteURL           string
	AutoDeployEnabled bool
}

type UpdateAppInput struct {
	Name              string
	Branch            string
	BuildType         string
	OutputDir         string
	RootDir           string
	SiteURL           string
	AutoDeployEnabled bool
}

type CreateDeploymentInput struct {
	AppID         int64
	UserID        int64
	Status        string
	TriggerType   string
	CommitSHA     string
	CommitMessage string
	CommitAuthor  string
	Branch        string
	SiteURL       string
	FailureReason string
	CorrelationID string
	StartedAt     int64
	FinishedAt    int64
}

type UpsertAWSConnectionInput struct {
	UserID          int64
	RoleARN         string
	ExternalID      string
	Region          string
	AccountID       string
	Status          string
	LastValidatedAt int64
}

type AuditEventInput struct {
	ActorUserID int64
	EventType   string
	TargetType  string
	TargetID    string
	Status      string
	Message     string
	Metadata    string
}
