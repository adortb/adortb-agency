package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/adortb/adortb-agency/internal/agency"
	"github.com/adortb/adortb-agency/internal/api"
	"github.com/adortb/adortb-agency/internal/batch"
	"github.com/adortb/adortb-agency/internal/commission"
	"github.com/adortb/adortb-agency/internal/metrics"
	"github.com/adortb/adortb-agency/internal/reporting"
	"github.com/adortb/adortb-agency/internal/user"
)

func main() {
	dsn := envOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/adortb_agency?sslmode=disable")
	port := envOrDefault("PORT", "8109")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("ping database", "err", err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)

	metrics.Register()

	agencyRepo := agency.NewRepo(db)
	hierarchyRepo := agency.NewHierarchyRepo(db)
	subaccRepo := user.NewSubaccountRepo(db)
	permRepo := user.NewPermissionRepo(db)
	aggregator := reporting.NewAggregator(db)
	wlRepo := reporting.NewWhiteLabelRepo(db)
	calc := commission.NewCalculator(db)
	settlement := commission.NewSettlementService(db, calc)
	batchOp := batch.NewOperator(&batch.NoopCampaignClient{}, 10)

	h := api.NewHandler(agencyRepo, hierarchyRepo, subaccRepo, permRepo,
		aggregator, wlRepo, calc, settlement, batchOp)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf(":%s", port)
	slog.Info("agency service starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
