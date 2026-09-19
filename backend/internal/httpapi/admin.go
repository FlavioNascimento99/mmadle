package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// Admin interface (metrics). Admins are bootstrapped from the ADMIN_USERNAMES
// env allowlist: matching accounts promote to role='admin' on register and
// login. The column is the source of truth afterwards — removing an address
// from the list never demotes (demote with SetUserRole / SQL).

// userRoleAdmin is the role gating /api/admin/*.
const userRoleAdmin = "admin"

// maybePromote gives allowlisted accounts the admin role. Best-effort: auth
// never fails because promotion did.
func (s *Server) maybePromote(ctx context.Context, user *store.User) {
	if s.Auth == nil || len(s.AdminUsernames) == 0 || user.Role == userRoleAdmin {
		return
	}
	if !s.AdminUsernames[domain.NormalizeUsername(user.Username)] {
		return
	}
	if err := s.Auth.SetUserRole(ctx, user.ID, userRoleAdmin); err != nil {
		s.Logger.Error("admin promotion failed", "user_id", user.ID)
		return
	}
	user.Role = userRoleAdmin
}

// requireAdmin enforces the admin role for the metrics interface. Guests get
// 401 (like /api/me/*); signed-in non-admins get 404, not 403, so the
// interface's existence is not advertised to regular players.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if user.Role != userRoleAdmin {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return store.User{}, false
	}
	return user, true
}

// adminDays parses the ?days= window (default 30, clamped 1..90).
func adminDays(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("days"))
	if raw == "" {
		return 30, true
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > 90 {
		writeError(w, http.StatusBadRequest, "invalid_days", "days must be an integer between 1 and 90")
		return 0, false
	}
	return days, true
}

// GET /api/admin/metrics/overview?days= — product metrics from Postgres.
// Game semantics only exist here; Cloudflare sees requests, not guesses.
func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	// Guests get 401, players get 404 (the interface stays unadvertised);
	// only admins can observe whether metrics are configured.
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if s.AdminMetrics == nil {
		writeError(w, http.StatusInternalServerError, "metrics_unavailable", "metrics are not configured")
		return
	}
	days, ok := adminDays(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	overview, err := s.AdminMetrics.MetricsOverview(ctx, days)
	if err != nil {
		s.Logger.Error("metrics overview failed")
		writeError(w, http.StatusInternalServerError, "metrics_failed", "could not load metrics")
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// --- Cloudflare infrastructure metrics (GraphQL proxy) ---

// cfGraphQLEndpoint is the Analytics API; overridden in tests.
const cfGraphQLEndpoint = "https://api.cloudflare.com/client/v4/graphql"

type cfDay struct {
	Date        string  `json:"date"`
	Requests    int64   `json:"requests"`
	Errors      int64   `json:"errors"`
	Subrequests int64   `json:"subrequests"`
	CPUTimeP50  float64 `json:"cpu_time_p50_us"`
	CPUTimeP99  float64 `json:"cpu_time_p99_us"`
}

type cfWorkersReport struct {
	Days     int     `json:"days"`
	Since    string  `json:"since"`
	Script   string  `json:"script"`
	Requests int64   `json:"requests"`
	Errors   int64   `json:"errors"`
	ByDay    []cfDay `json:"by_day"`
}

type cfGraphQLResponse struct {
	Data *struct {
		Viewer struct {
			Accounts []struct {
				Workers []struct {
					Dimensions struct {
						Date string `json:"date"`
					} `json:"dimensions"`
					Sum *struct {
						Requests    *int64 `json:"requests"`
						Errors      *int64 `json:"errors"`
						Subrequests *int64 `json:"subrequests"`
					} `json:"sum"`
					Quantiles *struct {
						CPUTimeP50 *float64 `json:"cpuTimeP50"`
						CPUTimeP99 *float64 `json:"cpuTimeP99"`
					} `json:"quantiles"`
				} `json:"workersInvocationsAdaptive"`
			} `json:"accounts"`
		} `json:"viewer"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// GET /api/admin/cloudflare/workers?days= — Worker requests/errors/CPU from
// the Cloudflare GraphQL Analytics API, proxied so the API token never
// reaches the browser. Needs CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID.
func (s *Server) handleCloudflareWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if s.CloudflareToken == "" || s.CloudflareAccount == "" {
		writeError(w, http.StatusNotImplemented, "cloudflare_not_configured",
			"set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID secrets to enable infrastructure metrics")
		return
	}
	days, ok := adminDays(w, r)
	if !ok {
		return
	}
	now := s.Clock.Now().UTC()
	since := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")

	query := `query($accountTag: string!, $from: Time!, $to: Time!, $script: string!) {
viewer { accounts(filter: {accountTag: $accountTag}) {
workersInvocationsAdaptive(limit: 10000,
filter: {datetime_geq: $from, datetime_leq: $to, scriptName: $script}) {
dimensions { date } sum { requests errors subrequests }
quantiles { cpuTimeP50 cpuTimeP99 } } } } }`
	body, _ := json.Marshal(map[string]any{
		"query": query,
		"variables": map[string]any{
			"accountTag": s.CloudflareAccount,
			"from":       since + "T00:00:00Z",
			"to":         now.Format("2006-01-02T15:04:05Z"),
			"script":     "mmadle",
		},
	})

	endpoint := s.cfGraphQLURL
	if endpoint == "" {
		endpoint = cfGraphQLEndpoint
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cloudflare_failed", "could not query Cloudflare")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.CloudflareToken)
	client := s.cfHTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		s.Logger.Error("cloudflare graphql request failed")
		writeError(w, http.StatusBadGateway, "cloudflare_failed", "could not reach Cloudflare analytics")
		return
	}
	defer resp.Body.Close()
	var parsed cfGraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		s.Logger.Error("cloudflare graphql decode failed")
		writeError(w, http.StatusBadGateway, "cloudflare_failed", "invalid response from Cloudflare analytics")
		return
	}
	if len(parsed.Errors) > 0 || parsed.Data == nil {
		s.Logger.Error("cloudflare graphql returned errors")
		writeError(w, http.StatusBadGateway, "cloudflare_failed", "Cloudflare analytics rejected the query")
		return
	}
	report := cfWorkersReport{Days: days, Since: since, Script: "mmadle", ByDay: []cfDay{}}
	for _, acct := range parsed.Data.Viewer.Accounts {
		for _, row := range acct.Workers {
			day := cfDay{Date: row.Dimensions.Date}
			if row.Sum != nil {
				if row.Sum.Requests != nil {
					day.Requests = *row.Sum.Requests
				}
				if row.Sum.Errors != nil {
					day.Errors = *row.Sum.Errors
				}
				if row.Sum.Subrequests != nil {
					day.Subrequests = *row.Sum.Subrequests
				}
			}
			if row.Quantiles != nil {
				if row.Quantiles.CPUTimeP50 != nil {
					day.CPUTimeP50 = *row.Quantiles.CPUTimeP50
				}
				if row.Quantiles.CPUTimeP99 != nil {
					day.CPUTimeP99 = *row.Quantiles.CPUTimeP99
				}
			}
			report.ByDay = append(report.ByDay, day)
			report.Requests += day.Requests
			report.Errors += day.Errors
		}
	}
	writeJSON(w, http.StatusOK, report)
}
