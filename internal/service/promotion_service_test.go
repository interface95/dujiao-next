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
