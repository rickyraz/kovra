package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/shopspring/decimal"

	"kovra/internal/cache"
	"kovra/internal/config"
	"kovra/internal/handler"
	"kovra/internal/ledger"
	"kovra/internal/repository"
)

// Demo tenant IDs (from seed migration)
var (
	EuroFintechTenantID = uuid.MustParse("019471a0-0000-7000-8000-000000000001")
	BritPayTenantID     = uuid.MustParse("019471a0-0000-7000-8000-000000000002")
	IndoRemitTenantID   = uuid.MustParse("019471a0-0000-7000-8000-000000000003")
	SwedeMartTenantID   = uuid.MustParse("019471a0-0000-7000-8000-000000000004")
)

// testContext holds test dependencies
type testContext struct {
	pool         *pgxpool.Pool
	ledgerClient *ledger.Client
	cacheClient  *cache.Client
	router       chi.Router
	cfg          *config.Config
}

func setupTestContext(t *testing.T) *testContext {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "failed to load config")

	// Override with test database URL if provided
	if dbURL := os.Getenv("TEST_DATABASE_URL"); dbURL != "" {
		cfg.Database.URL = dbURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	require.NoError(t, err, "failed to connect to database")

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "failed to ping database")

	tc := &testContext{
		pool: pool,
		cfg:  cfg,
	}

	// Try to connect to TigerBeetle (optional for Phase 1 Week 1)
	ledgerClient, err := ledger.NewClient(cfg.TigerBeetle)
	if err != nil {
		t.Logf("TigerBeetle not available: %v (some tests will be skipped)", err)
	} else {
		tc.ledgerClient = ledgerClient
	}

	// Try to connect to Redis (optional)
	cacheClient, err := cache.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		t.Logf("Redis not available: %v (some tests will be skipped)", err)
	} else {
		tc.cacheClient = cacheClient
	}

	// Setup router with handlers
	tc.router = setupRouter(tc)

	return tc
}

func (tc *testContext) cleanup() {
	if tc.ledgerClient != nil {
		tc.ledgerClient.Close()
	}
	if tc.cacheClient != nil {
		tc.cacheClient.Close()
	}
	if tc.pool != nil {
		tc.pool.Close()
	}
}

func setupRouter(tc *testContext) chi.Router {
	logger, _ := zap.NewDevelopment()

	legalEntityRepo := repository.NewLegalEntityRepository(tc.pool)
	tenantRepo := repository.NewTenantRepository(tc.pool)
	walletRepo := repository.NewWalletRepository(tc.pool)
	transferRepo := repository.NewTransferRepository(tc.pool)

	legalEntityHandler := handler.NewLegalEntityHandler(legalEntityRepo)
	tenantHandler := handler.NewTenantHandler(tenantRepo)
	walletHandler := handler.NewWalletHandler(walletRepo, tc.ledgerClient)
	transferHandler := handler.NewTransferHandler(transferRepo, walletRepo)

	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/legal-entities", legalEntityHandler.List)
		r.Get("/legal-entities/{id}", legalEntityHandler.Get)
		r.Get("/legal-entities/code/{code}", legalEntityHandler.GetByCode)
		r.Get("/legal-entities/{id}/tenants", tenantHandler.ListByLegalEntity)

		r.Post("/tenants", tenantHandler.Create)
		r.Get("/tenants/{id}", tenantHandler.Get)
		r.Patch("/tenants/{id}", tenantHandler.Update)
		r.Get("/tenants/{id}/wallets", walletHandler.ListByTenant)
		r.Get("/tenants/{id}/transfers", transferHandler.ListByTenant)

		r.Post("/wallets", walletHandler.Create)
		r.Get("/wallets/{id}", walletHandler.Get)
		r.Get("/wallets/{id}/balance", walletHandler.GetBalance)

		r.Post("/transfers", transferHandler.Create)
		r.Get("/transfers/{id}", transferHandler.Get)
	})

	_ = logger // suppress unused warning
	return r
}

// TestWeek1Demo runs the Phase 1 Week 1 demo scenarios
func TestWeek1Demo(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup()

	t.Run("1_LegalEntitiesExist", func(t *testing.T) {
		testLegalEntitiesExist(t, tc)
	})

	t.Run("2_TenantsSeeded", func(t *testing.T) {
		testTenantsSeeded(t, tc)
	})

	t.Run("3_TenantHierarchy", func(t *testing.T) {
		testTenantHierarchy(t, tc)
	})

	t.Run("4_PricingPoliciesExist", func(t *testing.T) {
		testPricingPoliciesExist(t, tc)
	})

	t.Run("5_LimitPoliciesExist", func(t *testing.T) {
		testLimitPoliciesExist(t, tc)
	})

	t.Run("6_PricingExcludeConstraint", func(t *testing.T) {
		testPricingExcludeConstraint(t, tc)
	})

	t.Run("7_CreateTransferInCorrectPartition", func(t *testing.T) {
		testTransferPartitioning(t, tc)
	})

	if tc.ledgerClient != nil {
		t.Run("8_CreateWalletWithTigerBeetle", func(t *testing.T) {
			testCreateWalletWithTigerBeetle(t, tc)
		})
	}

	if tc.cacheClient != nil {
		t.Run("9_RedisFXRateLock", func(t *testing.T) {
			testRedisFXRateLock(t, tc)
		})
	}
}

// Test 1: Verify all legal entities exist
func testLegalEntitiesExist(t *testing.T, tc *testContext) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/legal-entities", nil)
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var entities []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &entities)
	require.NoError(t, err)

	assert.Len(t, entities, 3, "should have 3 legal entities")

	// Verify each entity
	codes := make(map[string]bool)
	for _, e := range entities {
		codes[e["code"].(string)] = true
	}

	assert.True(t, codes["KOVRA_EU"], "KOVRA_EU should exist")
	assert.True(t, codes["KOVRA_UK"], "KOVRA_UK should exist")
	assert.True(t, codes["KOVRA_ID"], "KOVRA_ID should exist")

	t.Log("✓ All 3 legal entities exist (KOVRA_EU, KOVRA_UK, KOVRA_ID)")
}

// Test 2: Verify demo tenants are seeded
func testTenantsSeeded(t *testing.T, tc *testContext) {
	tenantIDs := []uuid.UUID{
		EuroFintechTenantID,
		BritPayTenantID,
		IndoRemitTenantID,
		SwedeMartTenantID,
	}

	for _, id := range tenantIDs {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+id.String(), nil)
		w := httptest.NewRecorder()
		tc.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "tenant %s should exist", id)

		var tenant map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &tenant)
		require.NoError(t, err)

		t.Logf("✓ Tenant exists: %s (%s)", tenant["display_name"], tenant["country"])
	}
}

// Test 3: Verify tenant hierarchy (SwedeMart is sub-tenant of EuroFintech)
func testTenantHierarchy(t *testing.T, tc *testContext) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+SwedeMartTenantID.String(), nil)
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var tenant map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &tenant)
	require.NoError(t, err)

	assert.Equal(t, "sub_merchant", tenant["tenant_kind"], "SwedeMart should be a sub_merchant")
	assert.Equal(t, EuroFintechTenantID.String(), tenant["parent_tenant_id"], "SwedeMart's parent should be EuroFintech")

	t.Log("✓ Tenant hierarchy verified: SwedeMart → EuroFintech")
}

// Test 4: Verify pricing policies exist for all tenants
func testPricingPoliciesExist(t *testing.T, tc *testContext) {
	ctx := context.Background()

	var count int
	err := tc.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pricing_policies`).Scan(&count)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, count, 4, "should have at least 4 pricing policies")

	// Verify EuroFintech has premium pricing (100 bps)
	var margin int
	err = tc.pool.QueryRow(ctx, `
		SELECT fx_margin_bps FROM pricing_policies
		WHERE tenant_id = $1 AND valid_until IS NULL
	`, EuroFintechTenantID).Scan(&margin)
	require.NoError(t, err)

	assert.Equal(t, 100, margin, "EuroFintech should have 100 bps margin")

	t.Logf("✓ %d pricing policies exist, EuroFintech margin: %d bps", count, margin)
}

// Test 5: Verify limit policies exist for all tenants
func testLimitPoliciesExist(t *testing.T, tc *testContext) {
	ctx := context.Background()

	var count int
	err := tc.pool.QueryRow(ctx, `SELECT COUNT(*) FROM limit_policies`).Scan(&count)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, count, 4, "should have at least 4 limit policies")

	// Verify EuroFintech has high limits
	var dailyLimit float64
	err = tc.pool.QueryRow(ctx, `
		SELECT daily_limit_usd FROM limit_policies
		WHERE tenant_id = $1
	`, EuroFintechTenantID).Scan(&dailyLimit)
	require.NoError(t, err)

	assert.Equal(t, 1000000.0, dailyLimit, "EuroFintech should have $1M daily limit")

	t.Logf("✓ %d limit policies exist, EuroFintech daily limit: $%.0f", count, dailyLimit)
}

// Test 6: Verify EXCLUDE constraint prevents overlapping pricing
func testPricingExcludeConstraint(t *testing.T, tc *testContext) {
	ctx := context.Background()

	// Try to insert overlapping pricing policy for EuroFintech
	_, err := tc.pool.Exec(ctx, `
		INSERT INTO pricing_policies (tenant_id, fx_margin_bps, valid_from)
		VALUES ($1, 200, NOW())
	`, EuroFintechTenantID)

	// Should fail due to EXCLUDE constraint
	assert.Error(t, err, "overlapping pricing should be rejected")
	assert.Contains(t, err.Error(), "pricing_no_overlap", "error should mention constraint")

	t.Log("✓ EXCLUDE constraint prevents overlapping pricing periods")
}

// Test 7: Verify transfers land in correct partitions based on currency
func testTransferPartitioning(t *testing.T, tc *testContext) {
	ctx := context.Background()

	// Create EUR transfer (should go to transfers_eu)
	eurTransfer := map[string]any{
		"tenant_id":       EuroFintechTenantID.String(),
		"from_currency":   "EUR",
		"to_currency":     "EUR",
		"from_amount":     "1000.00",
		"to_amount":       "1000.00",
		"fx_rate":         "1.0",
		"idempotency_key": "test-eur-" + uuid.New().String(),
	}

	body, _ := json.Marshal(eurTransfer)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "EUR transfer should be created")

	var transfer map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &transfer)
	require.NoError(t, err)

	// Verify it's in transfers_eu partition
	var partitionName string
	err = tc.pool.QueryRow(ctx, `
		SELECT tableoid::regclass::text
		FROM transfers
		WHERE id = $1
	`, transfer["id"]).Scan(&partitionName)
	require.NoError(t, err)

	assert.Equal(t, "transfers_eu", partitionName, "EUR transfer should be in transfers_eu partition")
	t.Logf("✓ EUR transfer landed in %s partition", partitionName)

	// Create IDR transfer (should go to transfers_id)
	idrTransfer := map[string]any{
		"tenant_id":       EuroFintechTenantID.String(),
		"from_currency":   "EUR",
		"to_currency":     "IDR",
		"from_amount":     "100.00",
		"to_amount":       "1750000.00",
		"fx_rate":         "17500.00",
		"idempotency_key": "test-idr-" + uuid.New().String(),
	}

	body, _ = json.Marshal(idrTransfer)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "IDR transfer should be created")

	err = json.Unmarshal(w.Body.Bytes(), &transfer)
	require.NoError(t, err)

	err = tc.pool.QueryRow(ctx, `
		SELECT tableoid::regclass::text
		FROM transfers
		WHERE id = $1
	`, transfer["id"]).Scan(&partitionName)
	require.NoError(t, err)

	assert.Equal(t, "transfers_id", partitionName, "IDR transfer should be in transfers_id partition")
	t.Logf("✓ IDR transfer landed in %s partition", partitionName)

	// Create GBP transfer (should go to transfers_uk)
	gbpTransfer := map[string]any{
		"tenant_id":       BritPayTenantID.String(),
		"from_currency":   "GBP",
		"to_currency":     "GBP",
		"from_amount":     "500.00",
		"to_amount":       "500.00",
		"fx_rate":         "1.0",
		"idempotency_key": "test-gbp-" + uuid.New().String(),
	}

	body, _ = json.Marshal(gbpTransfer)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "GBP transfer should be created")

	err = json.Unmarshal(w.Body.Bytes(), &transfer)
	require.NoError(t, err)

	err = tc.pool.QueryRow(ctx, `
		SELECT tableoid::regclass::text
		FROM transfers
		WHERE id = $1
	`, transfer["id"]).Scan(&partitionName)
	require.NoError(t, err)

	assert.Equal(t, "transfers_uk", partitionName, "GBP transfer should be in transfers_uk partition")
	t.Logf("✓ GBP transfer landed in %s partition", partitionName)
}

// Test 8: Create wallet with TigerBeetle integration
func testCreateWalletWithTigerBeetle(t *testing.T, tc *testContext) {
	if tc.ledgerClient == nil {
		t.Skip("TigerBeetle not available")
	}

	// Create EUR wallet for EuroFintech
	walletReq := map[string]any{
		"tenant_id": EuroFintechTenantID.String(),
		"currency":  "EUR",
	}

	body, _ := json.Marshal(walletReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	// May fail if wallet already exists (from previous test run)
	if w.Code == http.StatusConflict {
		t.Log("✓ EUR wallet already exists for EuroFintech")
		return
	}

	require.Equal(t, http.StatusCreated, w.Code, "wallet should be created")

	var wallet map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &wallet)
	require.NoError(t, err)

	assert.NotEmpty(t, wallet["id"])
	assert.NotEmpty(t, wallet["tb_account_id"], "should have TigerBeetle account ID")

	t.Logf("✓ Wallet created with TB account: %v", wallet["tb_account_id"])

	// Verify balance endpoint works
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wallets/%s/balance", wallet["id"]), nil)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "balance endpoint should work")

	var balance map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &balance)
	require.NoError(t, err)

	t.Logf("✓ Wallet balance: available=%s, pending=%s, total=%s",
		balance["available"], balance["pending"], balance["total"])
}

// Test 9: Redis FX Rate Lock
func testRedisFXRateLock(t *testing.T, tc *testContext) {
	if tc.cacheClient == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	quoteID := uuid.New().String()

	// Lock an FX rate
	// FXRateLock uses value (not pointer) - small struct, passed by value to LockFXRate
	lock := cache.FXRateLock{
		QuoteID:      quoteID,
		FromCurrency: "EUR",
		ToCurrency:   "IDR",
		Rate:         decimal.RequireFromString("17500.50"), // decimal.Decimal is value type, not pointer
	}

	err := tc.cacheClient.LockFXRate(ctx, lock, 30*time.Second)
	require.NoError(t, err)

	t.Log("✓ FX rate locked in Redis")

	// Retrieve the locked rate
	retrieved, err := tc.cacheClient.GetFXRate(ctx, quoteID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, "EUR", retrieved.FromCurrency)
	assert.Equal(t, "IDR", retrieved.ToCurrency)
	assert.Equal(t, "17500.5", retrieved.Rate.String())

	t.Logf("✓ Retrieved locked rate: %s %s/%s", retrieved.Rate.String(), retrieved.FromCurrency, retrieved.ToCurrency)

	// Try to lock same quote again (should fail)
	err = tc.cacheClient.LockFXRate(ctx, lock, 30*time.Second)
	assert.Error(t, err, "duplicate lock should fail")

	t.Log("✓ Duplicate FX rate lock prevented")

	// Clean up
	err = tc.cacheClient.DeleteFXRate(ctx, quoteID)
	require.NoError(t, err)
}

// TestWeek2Demo runs the Phase 1 Week 2 demo scenarios
func TestWeek2Demo(t *testing.T) {
	tc := setupTestContext(t)
	defer tc.cleanup()

	if tc.ledgerClient == nil {
		t.Skip("TigerBeetle not available - Week 2 demo requires TigerBeetle cluster")
	}

	t.Run("1_MultiCurrencyAccounts", func(t *testing.T) {
		testMultiCurrencyAccounts(t, tc)
	})

	// Note: Full Week 2 tests (atomic FX chain, hold/capture) require
	// TigerBeetle cluster and wallet service implementation
}

func testMultiCurrencyAccounts(t *testing.T, tc *testContext) {
	currencies := []string{"EUR", "GBP", "IDR"}

	for _, currency := range currencies {
		walletReq := map[string]any{
			"tenant_id": EuroFintechTenantID.String(),
			"currency":  currency,
		}

		body, _ := json.Marshal(walletReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/wallets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		tc.router.ServeHTTP(w, req)

		// Accept both created and conflict (already exists)
		if w.Code == http.StatusCreated {
			t.Logf("✓ Created %s wallet for EuroFintech", currency)
		} else if w.Code == http.StatusConflict {
			t.Logf("✓ %s wallet already exists for EuroFintech", currency)
		} else {
			t.Errorf("Unexpected status %d for %s wallet", w.Code, currency)
		}
	}

	// List wallets for tenant
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+EuroFintechTenantID.String()+"/wallets", nil)
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var wallets []map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &wallets)
	require.NoError(t, err)

	t.Logf("✓ EuroFintech has %d wallets", len(wallets))
	for _, w := range wallets {
		t.Logf("  - %s wallet: %s", w["currency"], w["id"])
	}
}
