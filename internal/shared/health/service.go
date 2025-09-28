package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/jia-app/paymentservice/internal/shared/metrics"
)

// Service provides health check endpoints
type Service struct {
	healthChecker    *metrics.HealthChecker
	metricsCollector *metrics.MetricsCollector
	logger           *zap.Logger
}

// NewService creates a new health check service
func NewService(metricsCollector *metrics.MetricsCollector) *Service {
	return &Service{
		healthChecker:    metrics.NewHealthChecker(metricsCollector),
		metricsCollector: metricsCollector,
		logger:           log.L(context.Background()),
	}
}

// RegisterRoutes registers health check routes
func (s *Service) RegisterRoutes(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", s.healthCheck)
	router.GET("/health/ready", s.readinessCheck)
	router.GET("/health/live", s.livenessCheck)

	// Metrics endpoint
	router.GET("/metrics", s.metrics)

	// Status endpoint with detailed information
	router.GET("/status", s.status)
}

// healthCheck provides a comprehensive health check
func (s *Service) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	start := time.Now()
	statuses := s.healthChecker.CheckHealth(ctx)
	duration := time.Since(start)

	// Determine overall health
	overallHealthy := true
	for _, status := range statuses {
		if status.Status != "healthy" {
			overallHealthy = false
			break
		}
	}

	response := gin.H{
		"status":     getOverallStatus(overallHealthy),
		"timestamp":  time.Now().Format(time.RFC3339),
		"duration":   duration.String(),
		"components": statuses,
	}

	if overallHealthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// readinessCheck checks if the service is ready to accept traffic
func (s *Service) readinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check critical components
	statuses := s.healthChecker.CheckHealth(ctx)

	// For readiness, we only check critical components
	criticalComponents := []string{"database", "redis"}
	ready := true

	for _, component := range criticalComponents {
		if status, exists := statuses[component]; !exists || status.Status != "healthy" {
			ready = false
			break
		}
	}

	response := gin.H{
		"status":    getOverallStatus(ready),
		"timestamp": time.Now().Format(time.RFC3339),
		"ready":     ready,
	}

	if ready {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// livenessCheck checks if the service is alive
func (s *Service) livenessCheck(c *gin.Context) {
	// Liveness check is simple - if we can respond, we're alive
	response := gin.H{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(time.Now()).String(), // This would be actual uptime in production
	}

	c.JSON(http.StatusOK, response)
}

// metrics provides Prometheus metrics
func (s *Service) metrics(c *gin.Context) {
	// In a real implementation, this would return Prometheus metrics
	// For now, return a simple response
	response := gin.H{
		"message":   "Metrics endpoint - Prometheus metrics would be served here",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// status provides detailed service status
func (s *Service) status(c *gin.Context) {
	ctx := c.Request.Context()

	start := time.Now()
	statuses := s.healthChecker.CheckHealth(ctx)
	duration := time.Since(start)

	// Determine overall health
	overallHealthy := true
	for _, status := range statuses {
		if status.Status != "healthy" {
			overallHealthy = false
			break
		}
	}

	response := gin.H{
		"service":        "payment-service",
		"version":        "1.0.0", // This would come from build info
		"status":         getOverallStatus(overallHealthy),
		"timestamp":      time.Now().Format(time.RFC3339),
		"check_duration": duration.String(),
		"components":     statuses,
		"endpoints": gin.H{
			"health":  "/health",
			"ready":   "/health/ready",
			"live":    "/health/live",
			"metrics": "/metrics",
			"status":  "/status",
		},
	}

	if overallHealthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// getOverallStatus returns the overall status string
func getOverallStatus(healthy bool) string {
	if healthy {
		return "healthy"
	}
	return "unhealthy"
}
