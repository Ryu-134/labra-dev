package handlers

import "net/http"

type ServiceStatus struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Mode        string `json:"mode"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

var serviceStatuses = []ServiceStatus{
	{
		Name:        "control-api",
		Tier:        "api",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "User-facing API gateway and metadata endpoints",
	},
	{
		Name:        "deploy-orchestrator",
		Tier:        "worker",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "Deployment queueing and execution orchestration",
	},
	{
		Name:        "webhook-ingestor",
		Tier:        "ingestion",
		Mode:        "in-process",
		Status:      "healthy",
		Description: "GitHub webhook normalization and routing",
	},
}

func GetSystemServicesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"services": serviceStatuses,
		"architecture": map[string]any{
			"pattern": "3-tier + microservice control-plane",
			"tiers":   []string{"frontend", "api", "metadata"},
		},
	})
}
