package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"labra-backend/internal/api/store"
)

var appStore *store.Store

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type createAppRequest struct {
	Name              string `json:"name"`
	RepoFullName      string `json:"repo_full_name"`
	Branch            string `json:"branch"`
	BuildType         string `json:"build_type"`
	OutputDir         string `json:"output_dir"`
	RootDir           string `json:"root_dir"`
	SiteURL           string `json:"site_url"`
	AutoDeployEnabled *bool  `json:"auto_deploy_enabled"`
}

type updateAppRequest struct {
	Name              *string `json:"name"`
	Branch            *string `json:"branch"`
	BuildType         *string `json:"build_type"`
	OutputDir         *string `json:"output_dir"`
	RootDir           *string `json:"root_dir"`
	SiteURL           *string `json:"site_url"`
	AutoDeployEnabled *bool   `json:"auto_deploy_enabled"`
}

func InitAppStore(db *sql.DB) {
	appStore = store.New(db)
}

func CreateAppHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	var body createAppRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	in, err := normalizeCreateApp(body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	in.UserID = userID

	app, err := appStore.CreateApp(r.Context(), in)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			writeJSONError(w, http.StatusConflict, "app already exists for repo+branch")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to create app")
		return
	}

	writeJSON(w, http.StatusCreated, app)
}

func ListAppsHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	apps, err := appStore.ListAppsByUser(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to list apps")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
}

func GetAppHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	app, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func PatchAppHandler(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSONError(w, http.StatusInternalServerError, "store not initialized")
		return
	}

	userID, ok := readUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing user id: pass X-User-ID header")
		return
	}

	appID, err := readIDFromPathOrQuery(r, "apps")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body updateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	current, err := appStore.GetAppByIDForUser(r.Context(), appID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to load app")
		return
	}

	next, err := mergeAppUpdate(current, body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	app, err := appStore.UpdateAppForUser(r.Context(), appID, userID, next)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "app not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to update app")
		return
	}

	writeJSON(w, http.StatusOK, app)
}

func normalizeCreateApp(req createAppRequest) (store.CreateAppInput, error) {
	name := strings.TrimSpace(req.Name)
	repo := strings.TrimSpace(req.RepoFullName)
	branch := strings.TrimSpace(req.Branch)
	buildType := strings.TrimSpace(req.BuildType)
	outputDir := strings.TrimSpace(req.OutputDir)
	rootDir := strings.TrimSpace(req.RootDir)
	siteURL := strings.TrimSpace(req.SiteURL)

	if name == "" {
		return store.CreateAppInput{}, fmt.Errorf("name is required")
	}
	if repo == "" {
		return store.CreateAppInput{}, fmt.Errorf("repo_full_name is required")
	}
	if !repoPattern.MatchString(repo) {
		return store.CreateAppInput{}, fmt.Errorf("repo_full_name must look like owner/repo")
	}
	if branch == "" {
		branch = "main"
	}
	if buildType == "" {
		buildType = "static"
	}
	if buildType != "static" {
		return store.CreateAppInput{}, fmt.Errorf("build_type must be static for MVP")
	}
	if outputDir == "" {
		outputDir = "dist"
	}

	autoDeploy := true
	if req.AutoDeployEnabled != nil {
		autoDeploy = *req.AutoDeployEnabled
	}

	return store.CreateAppInput{
		Name:              name,
		RepoFullName:      strings.ToLower(repo),
		Branch:            branch,
		BuildType:         buildType,
		OutputDir:         outputDir,
		RootDir:           rootDir,
		SiteURL:           siteURL,
		AutoDeployEnabled: autoDeploy,
	}, nil
}

func mergeAppUpdate(current store.App, req updateAppRequest) (store.UpdateAppInput, error) {
	next := store.UpdateAppInput{
		Name:              current.Name,
		Branch:            current.Branch,
		BuildType:         current.BuildType,
		OutputDir:         current.OutputDir,
		RootDir:           current.RootDir,
		SiteURL:           current.SiteURL,
		AutoDeployEnabled: current.AutoDeployEnabled,
	}

	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.Branch != nil {
		next.Branch = strings.TrimSpace(*req.Branch)
	}
	if req.BuildType != nil {
		next.BuildType = strings.TrimSpace(*req.BuildType)
	}
	if req.OutputDir != nil {
		next.OutputDir = strings.TrimSpace(*req.OutputDir)
	}
	if req.RootDir != nil {
		next.RootDir = strings.TrimSpace(*req.RootDir)
	}
	if req.SiteURL != nil {
		next.SiteURL = strings.TrimSpace(*req.SiteURL)
	}
	if req.AutoDeployEnabled != nil {
		next.AutoDeployEnabled = *req.AutoDeployEnabled
	}

	if next.Name == "" {
		return store.UpdateAppInput{}, fmt.Errorf("name cannot be empty")
	}
	if next.Branch == "" {
		return store.UpdateAppInput{}, fmt.Errorf("branch cannot be empty")
	}
	if next.BuildType == "" {
		next.BuildType = "static"
	}
	if next.BuildType != "static" {
		return store.UpdateAppInput{}, fmt.Errorf("build_type must be static for MVP")
	}
	if next.OutputDir == "" {
		next.OutputDir = "dist"
	}

	return next, nil
}

func readUserID(r *http.Request) (int64, bool) {
	v := strings.TrimSpace(r.Header.Get("X-User-ID"))
	if v == "" {
		v = strings.TrimSpace(r.URL.Query().Get("user_id"))
	}
	if v == "" {
		return 0, false
	}

	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func readIDFromPathOrQuery(r *http.Request, base string) (int64, error) {
	if raw := strings.TrimSpace(r.URL.Query().Get("id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("id must be a positive integer")
		}
		return id, nil
	}

	prefix := "/v1/" + strings.Trim(base, "/") + "/"
	path := strings.TrimSpace(r.URL.Path)
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("id not found in path")
	}

	raw := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if raw == "" {
		return 0, fmt.Errorf("id is required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	return id, nil
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"status":  status,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
