package health

import (
	"sync"
	"time"
)

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	StatusHealthy   ServiceStatus = "healthy"
	StatusUnhealthy ServiceStatus = "unhealthy"
	StatusDegraded  ServiceStatus = "degraded"
	StatusUnknown   ServiceStatus = "unknown"
)

// ServiceHealth contains health information for a service
type ServiceHealth struct {
	Status      ServiceStatus          `json:"status"`
	LastCheck   time.Time             `json:"last_check"`
	LastHealthy time.Time             `json:"last_healthy,omitempty"`
	Message     string                `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthManager manages the health status of various services
type HealthManager struct {
	services map[string]*ServiceHealth
	mutex    sync.RWMutex
}

// NewHealthManager creates a new health manager
func NewHealthManager() *HealthManager {
	return &HealthManager{
		services: make(map[string]*ServiceHealth),
	}
}

// UpdateServiceHealth updates the health status of a service
func (hm *HealthManager) UpdateServiceHealth(serviceName string, status ServiceStatus, message string, details map[string]interface{}) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	now := time.Now()
	
	health, exists := hm.services[serviceName]
	if !exists {
		health = &ServiceHealth{}
		hm.services[serviceName] = health
	}

	health.Status = status
	health.LastCheck = now
	health.Message = message
	health.Details = details

	// Update last healthy time if status is healthy
	if status == StatusHealthy {
		health.LastHealthy = now
	}
}

// SetServiceHealthy marks a service as healthy
func (hm *HealthManager) SetServiceHealthy(serviceName string, message string) {
	hm.UpdateServiceHealth(serviceName, StatusHealthy, message, nil)
}

// SetServiceUnhealthy marks a service as unhealthy
func (hm *HealthManager) SetServiceUnhealthy(serviceName string, message string, details map[string]interface{}) {
	hm.UpdateServiceHealth(serviceName, StatusUnhealthy, message, details)
}

// SetServiceDegraded marks a service as degraded
func (hm *HealthManager) SetServiceDegraded(serviceName string, message string, details map[string]interface{}) {
	hm.UpdateServiceHealth(serviceName, StatusDegraded, message, details)
}

// GetServiceHealth returns the health status of a specific service
func (hm *HealthManager) GetServiceHealth(serviceName string) (*ServiceHealth, bool) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	health, exists := hm.services[serviceName]
	if !exists {
		return nil, false
	}

	// Make a copy to avoid race conditions
	healthCopy := *health
	if health.Details != nil {
		healthCopy.Details = make(map[string]interface{})
		for k, v := range health.Details {
			healthCopy.Details[k] = v
		}
	}

	return &healthCopy, true
}

// GetAllServicesHealth returns the health status of all services
func (hm *HealthManager) GetAllServicesHealth() map[string]*ServiceHealth {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]*ServiceHealth)
	for name, health := range hm.services {
		// Make a copy to avoid race conditions
		healthCopy := *health
		if health.Details != nil {
			healthCopy.Details = make(map[string]interface{})
			for k, v := range health.Details {
				healthCopy.Details[k] = v
			}
		}
		result[name] = &healthCopy
	}

	return result
}

// GetOverallHealth returns the overall health status of the system
func (hm *HealthManager) GetOverallHealth() ServiceStatus {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	if len(hm.services) == 0 {
		return StatusUnknown
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, health := range hm.services {
		switch health.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		case StatusUnknown:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusDegraded // System is degraded if any service is unhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}

	return StatusHealthy
}

// IsServiceHealthy checks if a specific service is healthy
func (hm *HealthManager) IsServiceHealthy(serviceName string) bool {
	health, exists := hm.GetServiceHealth(serviceName)
	return exists && health.Status == StatusHealthy
}

// GetHealthSummary returns a summary of system health
func (hm *HealthManager) GetHealthSummary() map[string]interface{} {
	allHealth := hm.GetAllServicesHealth()
	overallStatus := hm.GetOverallHealth()

	summary := map[string]interface{}{
		"overall_status": overallStatus,
		"services":       allHealth,
		"timestamp":      time.Now(),
	}

	// Count services by status
	statusCounts := map[ServiceStatus]int{
		StatusHealthy:   0,
		StatusUnhealthy: 0,
		StatusDegraded:  0,
		StatusUnknown:   0,
	}

	for _, health := range allHealth {
		statusCounts[health.Status]++
	}

	summary["status_counts"] = statusCounts
	summary["total_services"] = len(allHealth)

	return summary
}