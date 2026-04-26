package service

import (
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestApplyPromotionUsesQuantityThreshold(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:promotion_quantity_threshold?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Promotion{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	now := time.Now()
	promotion := models.Promotion{
		Name:        "quantity-special-price",
		ScopeType:   constants.ScopeTypeProduct,
		ScopeRefID:  1,
		Type:        constants.PromotionTypeSpecialPrice,
		Value:       models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		MinQuantity: 5,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&promotion).Error; err != nil {
		t.Fatalf("create promotion failed: %v", err)
	}

	svc := NewPromotionService(repository.NewPromotionRepository(db))
	product := &models.Product{
		ID:          1,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
	}

	unmatched, unmatchedPrice, err := svc.ApplyPromotion(product, 4)
	if err != nil {
		t.Fatalf("apply unmatched promotion failed: %v", err)
	}
	if unmatched != nil {
		t.Fatalf("expected no promotion below quantity threshold, got: %+v", unmatched)
	}
	if !unmatchedPrice.Decimal.Equal(decimal.NewFromInt(80)) {
		t.Fatalf("expected base price below threshold, got: %s", unmatchedPrice.String())
	}

	matched, matchedPrice, err := svc.ApplyPromotion(product, 5)
	if err != nil {
		t.Fatalf("apply matched promotion failed: %v", err)
	}
	if matched == nil || matched.ID != promotion.ID {
		t.Fatalf("expected quantity promotion, got: %+v", matched)
	}
	if !matchedPrice.Decimal.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("expected special unit price 50, got: %s", matchedPrice.String())
	}
}

func TestApplyPromotionForSKUPreferSKURuleThenFallbackProductRule(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:promotion_sku_scope?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Promotion{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	productRule := models.Promotion{
		Name:       "product-fixed-10",
		ScopeType:  constants.ScopeTypeProduct,
		ScopeRefID: 1,
		Type:       constants.PromotionTypeFixed,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:   true,
	}
	skuRule := models.Promotion{
		Name:       "sku-special-60",
		ScopeType:  constants.ScopeTypeSKU,
		ScopeRefID: 100,
		Type:       constants.PromotionTypeSpecialPrice,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		IsActive:   true,
	}
	if err := db.Create(&productRule).Error; err != nil {
		t.Fatalf("create product promotion failed: %v", err)
	}
	if err := db.Create(&skuRule).Error; err != nil {
		t.Fatalf("create sku promotion failed: %v", err)
	}

	svc := NewPromotionService(repository.NewPromotionRepository(db))
	product := &models.Product{
		ID:          1,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
	}

	matched, matchedPrice, err := svc.ApplyPromotionForSKU(product, 100, 1)
	if err != nil {
		t.Fatalf("apply sku promotion failed: %v", err)
	}
	if matched == nil || matched.ID != skuRule.ID {
		t.Fatalf("expected sku promotion, got: %+v", matched)
	}
	if !matchedPrice.Decimal.Equal(decimal.NewFromInt(60)) {
		t.Fatalf("expected sku special price 60, got: %s", matchedPrice.String())
	}

	fallback, fallbackPrice, err := svc.ApplyPromotionForSKU(product, 101, 1)
	if err != nil {
		t.Fatalf("apply fallback promotion failed: %v", err)
	}
	if fallback == nil || fallback.ID != productRule.ID {
		t.Fatalf("expected product promotion fallback, got: %+v", fallback)
	}
	if !fallbackPrice.Decimal.Equal(decimal.NewFromInt(90)) {
		t.Fatalf("expected product fallback price 90, got: %s", fallbackPrice.String())
	}
}

func TestPromotionAdminCreateSupportsSKUScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:promotion_admin_sku_scope?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Promotion{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	svc := NewPromotionAdminService(repository.NewPromotionRepository(db))
	promotion, err := svc.Create(CreatePromotionInput{
		Name:       "sku-special-price",
		ScopeType:  constants.ScopeTypeSKU,
		ScopeRefID: 100,
		Type:       constants.PromotionTypeSpecialPrice,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
	})
	if err != nil {
		t.Fatalf("create sku promotion failed: %v", err)
	}
	if promotion.ScopeType != constants.ScopeTypeSKU {
		t.Fatalf("expected sku scope, got: %s", promotion.ScopeType)
	}
	if promotion.ScopeRefID != 100 {
		t.Fatalf("expected sku ref id 100, got: %d", promotion.ScopeRefID)
	}
}
