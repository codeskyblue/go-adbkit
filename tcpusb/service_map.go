package tcpusb

import "sync"

// ServiceMap manages active services by their local ID
type ServiceMap struct {
	mu       sync.RWMutex
	services map[uint32]*Service
	count    int
}

// NewServiceMap creates a new service map
func NewServiceMap() *ServiceMap {
	return &ServiceMap{
		services: make(map[uint32]*Service),
		count:    0,
	}
}

// Insert adds a service to the map
func (sm *ServiceMap) Insert(localID uint32, service *Service) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.services[localID]; exists {
		return ErrServiceExists
	}

	sm.services[localID] = service
	sm.count++
	return nil
}

// Get retrieves a service by its local ID
func (sm *ServiceMap) Get(localID uint32) *Service {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.services[localID]
}

// Remove removes a service from the map
func (sm *ServiceMap) Remove(localID uint32) *Service {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	service := sm.services[localID]
	if service != nil {
		delete(sm.services, localID)
		sm.count--
	}
	return service
}

// Count returns the number of active services
func (sm *ServiceMap) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.count
}

// End closes all services
func (sm *ServiceMap) End() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, service := range sm.services {
		service.End()
	}
	sm.services = make(map[uint32]*Service)
	sm.count = 0
}
