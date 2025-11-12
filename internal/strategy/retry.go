package strategy

import (
	"math"
	"time"

	"github.com/Rizwank123/notification_system/internal/config"
)

type RetryStrategy struct {
	cfg *config.Config
}

func NewRetryStrategy(cfg *config.Config) *RetryStrategy {
	return &RetryStrategy{cfg: cfg}
}

func (rs *RetryStrategy) CalculateDelay(retryCount int) time.Duration {
	delay := time.Duration(rs.cfg.RetryBaseDelay) * time.Duration(math.Pow(2, float64(retryCount)))

	if delay > time.Duration(rs.cfg.RetryMaxDelay) {
		delay = time.Duration(rs.cfg.RetryMaxDelay)
	}

	return delay
}

func (rs *RetryStrategy) ShouldRetry(retryCount int) bool {
	return retryCount < rs.cfg.MaxRetries
}
