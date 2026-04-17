package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/adortb/adortb-agency/internal/agency"
	"github.com/adortb/adortb-agency/internal/batch"
	"github.com/adortb/adortb-agency/internal/commission"
	"github.com/adortb/adortb-agency/internal/reporting"
	"github.com/adortb/adortb-agency/internal/user"
)

type Handler struct {
	agencyRepo    *agency.Repo
	hierarchyRepo *agency.HierarchyRepo
	subaccRepo    *user.SubaccountRepo
	permRepo      *user.PermissionRepo
	aggregator    *reporting.Aggregator
	wlRepo        *reporting.WhiteLabelRepo
	calc          *commission.Calculator
	settlement    *commission.SettlementService
	batchOp       *batch.Operator
}

func NewHandler(
	agencyRepo *agency.Repo,
	hierarchyRepo *agency.HierarchyRepo,
	subaccRepo *user.SubaccountRepo,
	permRepo *user.PermissionRepo,
	aggregator *reporting.Aggregator,
	wlRepo *reporting.WhiteLabelRepo,
	calc *commission.Calculator,
	settlement *commission.SettlementService,
	batchOp *batch.Operator,
) *Handler {
	return &Handler{
		agencyRepo:    agencyRepo,
		hierarchyRepo: hierarchyRepo,
		subaccRepo:    subaccRepo,
		permRepo:      permRepo,
		aggregator:    aggregator,
		wlRepo:        wlRepo,
		calc:          calc,
		settlement:    settlement,
		batchOp:       batchOp,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agencies", h.routeAgencies)
	mux.HandleFunc("/v1/agencies/", h.routeAgency)
	mux.HandleFunc("/v1/auth/login", h.login)
	mux.HandleFunc("/v1/auth/switch-advertiser", requireAuth(h.switchAdvertiser))
	mux.HandleFunc("/health", h.health)
}

// ── /v1/agencies ──────────────────────────────────────────────────

func (h *Handler) routeAgencies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createAgency(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) createAgency(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name           string  `json:"name"`
		LegalEntity    string  `json:"legal_entity"`
		ContactEmail   string  `json:"contact_email"`
		CommissionRate float64 `json:"commission_rate"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	a, err := h.agencyRepo.Create(agency.CreateAgencyReq{
		Name:           req.Name,
		LegalEntity:    req.LegalEntity,
		ContactEmail:   req.ContactEmail,
		CommissionRate: req.CommissionRate,
	})
	if err != nil {
		if err == agency.ErrDuplicate {
			writeError(w, http.StatusConflict, "agency already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create agency failed")
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// ── /v1/agencies/:id/** ───────────────────────────────────────────

func (h *Handler) routeAgency(w http.ResponseWriter, r *http.Request) {
	segs := pathSegments(r.URL.Path, "/v1/agencies/")
	if len(segs) == 0 {
		writeError(w, http.StatusBadRequest, "missing agency id")
		return
	}
	agencyID, err := parseInt64(segs[0])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid agency id")
		return
	}

	if len(segs) == 1 {
		// GET /v1/agencies/:id
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h.getAgency(w, r, agencyID)
		return
	}

	sub := segs[1]
	switch {
	case sub == "users" && len(segs) == 2:
		h.routeUsers(w, r, agencyID)
	case sub == "advertisers" && len(segs) == 2:
		h.routeAdvertisers(w, r, agencyID)
	case sub == "advertisers" && len(segs) == 4 && segs[3] == "permissions":
		h.routePermissions(w, r, agencyID, segs[2])
	case sub == "campaigns" && len(segs) == 2:
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.listCampaigns(w, r, agencyID, c)
		})(w, r)
	case sub == "batch" && len(segs) == 3 && segs[2] == "pause-campaigns":
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.batchPause(w, r, agencyID, c)
		})(w, r)
	case sub == "batch" && len(segs) == 3 && segs[2] == "budget-update":
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.batchBudgetUpdate(w, r, agencyID, c)
		})(w, r)
	case sub == "reports" && len(segs) == 3 && segs[2] == "aggregated":
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.aggregatedReport(w, r, agencyID, c)
		})(w, r)
	case sub == "commissions" && len(segs) == 2:
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.listCommissions(w, r, agencyID, c)
		})(w, r)
	case sub == "commissions" && len(segs) == 3 && segs[2] == "settle":
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.settleCommission(w, r, agencyID, c)
		})(w, r)
	case sub == "commissions" && len(segs) == 3:
		requireAuth(func(w http.ResponseWriter, r *http.Request, c *Claims) {
			h.getCommission(w, r, agencyID, segs[2], c)
		})(w, r)
	case sub == "white-label-config" && len(segs) == 2:
		h.routeWhiteLabel(w, r, agencyID)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (h *Handler) getAgency(w http.ResponseWriter, r *http.Request, id int64) {
	a, err := h.agencyRepo.GetByID(id)
	if err != nil {
		if err == agency.ErrNotFound {
			writeError(w, http.StatusNotFound, "agency not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "get agency failed")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// ── Users ─────────────────────────────────────────────────────────

func (h *Handler) routeUsers(w http.ResponseWriter, r *http.Request, agencyID int64) {
	switch r.Method {
	case http.MethodPost:
		h.createUser(w, r, agencyID)
	case http.MethodGet:
		h.listUsers(w, r, agencyID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request, agencyID int64) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Role == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email, role, password are required")
		return
	}
	if !validRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid role: must be agency_admin, media_buyer, or analyst")
		return
	}
	u, err := h.subaccRepo.Create(user.CreateUserReq{
		AgencyID: agencyID,
		Email:    req.Email,
		Name:     req.Name,
		Role:     req.Role,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create user failed")
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request, agencyID int64) {
	users, err := h.subaccRepo.ListByAgency(agencyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list users failed")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func validRole(role string) bool {
	return role == user.RoleAgencyAdmin || role == user.RoleMediaBuyer || role == user.RoleAnalyst
}

// ── Advertisers ───────────────────────────────────────────────────

func (h *Handler) routeAdvertisers(w http.ResponseWriter, r *http.Request, agencyID int64) {
	switch r.Method {
	case http.MethodPost:
		h.addAdvertiser(w, r, agencyID)
	case http.MethodGet:
		h.listAdvertisers(w, r, agencyID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) addAdvertiser(w http.ResponseWriter, r *http.Request, agencyID int64) {
	var req struct {
		AdvertiserID int64  `json:"advertiser_id"`
		Role         string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AdvertiserID <= 0 {
		writeError(w, http.StatusBadRequest, "advertiser_id is required")
		return
	}
	aa, err := h.hierarchyRepo.AddAdvertiser(agencyID, req.AdvertiserID, req.Role)
	if err != nil {
		if err == agency.ErrDuplicate {
			writeError(w, http.StatusConflict, "advertiser already linked")
			return
		}
		writeError(w, http.StatusInternalServerError, "add advertiser failed")
		return
	}
	writeJSON(w, http.StatusCreated, aa)
}

func (h *Handler) listAdvertisers(w http.ResponseWriter, r *http.Request, agencyID int64) {
	list, err := h.hierarchyRepo.ListAdvertisers(agencyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list advertisers failed")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ── Permissions ───────────────────────────────────────────────────

func (h *Handler) routePermissions(w http.ResponseWriter, r *http.Request, agencyID int64, advIDStr string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	advID, err := parseInt64(advIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid advertiser id")
		return
	}
	var req struct {
		AgencyUserID int64  `json:"agency_user_id"`
		Permission   string `json:"permission"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.permRepo.Grant(req.AgencyUserID, &advID, req.Permission); err != nil {
		writeError(w, http.StatusInternalServerError, "grant permission failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "granted"})
}

// ── Campaigns (proxy list) ────────────────────────────────────────

func (h *Handler) listCampaigns(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	advs, err := h.hierarchyRepo.ListAdvertisers(agencyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list advertisers failed")
		return
	}
	ids := make([]int64, len(advs))
	for i, a := range advs {
		ids[i] = a.AdvertiserID
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agency_id":      agencyID,
		"advertiser_ids": ids,
		"note":           "fetch campaigns per advertiser_id from adortb-dsp",
	})
}

// ── Batch Ops ─────────────────────────────────────────────────────

func (h *Handler) batchPause(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CampaignIDs []int64 `json:"campaign_ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	results := h.batchOp.Execute(context.Background(), batch.BatchRequest{
		CampaignIDs: req.CampaignIDs,
		Action:      batch.ActionPause,
	})
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) batchBudgetUpdate(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CampaignIDs []int64 `json:"campaign_ids"`
		NewBudget   float64 `json:"new_budget"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	results := h.batchOp.Execute(context.Background(), batch.BatchRequest{
		CampaignIDs: req.CampaignIDs,
		Action:      batch.ActionBudgetUpdate,
		NewBudget:   &req.NewBudget,
	})
	writeJSON(w, http.StatusOK, results)
}

// ── Reports ───────────────────────────────────────────────────────

func (h *Handler) aggregatedReport(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	from, _ := time.Parse("2006-01-02", q.Get("from"))
	to, _ := time.Parse("2006-01-02", q.Get("to"))
	if from.IsZero() {
		from = time.Now().AddDate(0, -1, 0)
	}
	if to.IsZero() {
		to = time.Now()
	}
	report, err := h.aggregator.GetAggregatedReport(agencyID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get report failed")
		return
	}
	writeJSON(w, http.StatusOK, report)
}

// ── Commissions ───────────────────────────────────────────────────

func (h *Handler) listCommissions(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	list, err := h.calc.ListHistory(agencyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list commissions failed")
		return
	}
	est, err := h.calc.EstimateCurrentMonth(agencyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "estimate commission failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"current_estimate": est,
		"history":          list,
	})
}

func (h *Handler) settleCommission(w http.ResponseWriter, r *http.Request, agencyID int64, _ *Claims) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		TotalSpend float64 `json:"total_spend"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rec, err := h.settlement.Settle(agencyID, req.TotalSpend)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settle failed")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (h *Handler) getCommission(w http.ResponseWriter, r *http.Request, agencyID int64, periodStr string, _ *Claims) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// periodStr format: "2024-01"
	t, err := time.Parse("2006-01", periodStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid period format, use YYYY-MM")
		return
	}
	rec, err := h.calc.GetByPeriod(agencyID, t)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// ── White Label ───────────────────────────────────────────────────

func (h *Handler) routeWhiteLabel(w http.ResponseWriter, r *http.Request, agencyID int64) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := h.wlRepo.GetConfig(agencyID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cfg)
	case http.MethodPut:
		var req struct {
			Domain       string `json:"domain"`
			LogoURL      string `json:"logo_url"`
			PrimaryColor string `json:"primary_color"`
		}
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := h.wlRepo.UpdateConfig(agencyID, req.Domain, req.LogoURL, req.PrimaryColor); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ── Auth ──────────────────────────────────────────────────────────

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		AgencyID int64  `json:"agency_id"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := h.subaccRepo.Authenticate(req.AgencyID, req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := generateToken(Claims{
		AgencyUserID: u.ID,
		AgencyID:     u.AgencyID,
		Role:         u.Role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":    token,
		"user":     u,
		"role":     u.Role,
		"agency_id": u.AgencyID,
	})
}

func (h *Handler) switchAdvertiser(w http.ResponseWriter, r *http.Request, c *Claims) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		AdvertiserID int64 `json:"advertiser_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ok, err := h.hierarchyRepo.HasAccess(c.AgencyID, req.AdvertiserID)
	if err != nil || !ok {
		writeError(w, http.StatusForbidden, "no access to this advertiser")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agency_id":     c.AgencyID,
		"advertiser_id": req.AdvertiserID,
		"switched":      true,
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 请求日志
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/health") && !strings.HasPrefix(r.URL.Path, "/metrics") {
			// minimal logging
		}
		next.ServeHTTP(w, r)
	})
}
