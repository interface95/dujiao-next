package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// PromotionService 活动价服务
type PromotionService struct {
	promotionRepo repository.PromotionRepository
}

// NewPromotionService 创建活动价服务
func NewPromotionService(promotionRepo repository.PromotionRepository) *PromotionService {
	return &PromotionService{
		promotionRepo: promotionRepo,
	}
}

// GetProductPromotions 获取商品所有有效活动规则（用于前端展示）
func (s *PromotionService) GetProductPromotions(productID uint) ([]models.Promotion, error) {
	return s.promotionRepo.GetAllActiveByProduct(productID, time.Now())
}

// ApplyPromotion 应用活动价规则（支持阶梯匹配）
func (s *PromotionService) ApplyPromotion(product *models.Product, quantity int) (*models.Promotion, models.Money, error) {
	if product == nil || quantity <= 0 {
		return nil, models.Money{}, ErrPromotionInvalid
	}

	now := time.Now()
	promotions, err := s.promotionRepo.GetAllActiveByProduct(product.ID, now)
	if err != nil {
		return nil, models.Money{}, err
	}
	if len(promotions) == 0 {
		return nil, product.PriceAmount, nil
	}

	matched := matchPromotionRule(promotions, product.PriceAmount.Decimal, quantity)

	if matched == nil {
		return nil, product.PriceAmount, nil
	}

	unitPrice, err := s.calculateUnitPrice(product.PriceAmount, matched)
	if err != nil {
		return nil, models.Money{}, err
	}

	return matched, unitPrice, nil
}

func matchPromotionRule(promotions []models.Promotion, basePrice decimal.Decimal, quantity int) *models.Promotion {
	subtotal := basePrice.Mul(decimal.NewFromInt(int64(quantity)))

	var quantityMatched *models.Promotion
	for i := range promotions {
		p := &promotions[i]
		if strings.ToLower(strings.TrimSpace(p.ScopeType)) != constants.ScopeTypeProduct {
			continue
		}
		if p.MinQuantity <= 0 || quantity < p.MinQuantity {
			continue
		}
		if quantityMatched == nil || p.MinQuantity > quantityMatched.MinQuantity {
			quantityMatched = p
		}
	}
	if quantityMatched != nil {
		return quantityMatched
	}

	var amountMatched *models.Promotion
	for i := range promotions {
		p := &promotions[i]
		if strings.ToLower(strings.TrimSpace(p.ScopeType)) != constants.ScopeTypeProduct {
			continue
		}
		if p.MinQuantity > 0 {
			continue
		}
		if p.MinAmount.Decimal.GreaterThan(decimal.Zero) && subtotal.LessThan(p.MinAmount.Decimal) {
			continue
		}
		if amountMatched == nil || p.MinAmount.Decimal.GreaterThan(amountMatched.MinAmount.Decimal) {
			amountMatched = p
		}
	}
	return amountMatched
}

func (s *PromotionService) calculateUnitPrice(base models.Money, promotion *models.Promotion) (models.Money, error) {
	value := promotion.Value.Decimal
	if value.LessThanOrEqual(decimal.Zero) {
		return models.Money{}, ErrPromotionInvalid
	}

	switch strings.ToLower(strings.TrimSpace(promotion.Type)) {
	case constants.PromotionTypeFixed:
		discounted := base.Decimal.Sub(value)
		if discounted.LessThan(decimal.Zero) {
			discounted = decimal.Zero
		}
		return models.NewMoneyFromDecimal(discounted), nil
	case constants.PromotionTypePercent:
		percent := decimal.NewFromInt(100).Sub(value)
		if percent.LessThan(decimal.Zero) {
			percent = decimal.Zero
		}
		discounted := base.Decimal.Mul(percent).Div(decimal.NewFromInt(100))
		return models.NewMoneyFromDecimal(discounted), nil
	case constants.PromotionTypeSpecialPrice:
		return models.NewMoneyFromDecimal(value), nil
	default:
		return models.Money{}, ErrPromotionInvalid
	}
}
