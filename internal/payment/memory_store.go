package payment

import "sync"

// MemoryRuleStore is an in-memory implementation of RuleRepository.
type MemoryRuleStore struct {
	mu    sync.RWMutex
	rules map[string]PricingRule
	order []string // Maintain insertion order
}

func NewMemoryRuleStore() *MemoryRuleStore {
	return &MemoryRuleStore{
		rules: make(map[string]PricingRule),
		order: make([]string, 0),
	}
}

func (s *MemoryRuleStore) Get(id string) (PricingRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[id]
	return r, ok
}

func (s *MemoryRuleStore) List() []PricingRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PricingRule, 0, len(s.rules))
	for _, id := range s.order {
		if r, exists := s.rules[id]; exists && r.Enabled {
			out = append(out, r)
		}
	}
	return out
}

func (s *MemoryRuleStore) Upsert(rule PricingRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to order if it's a new rule
	if _, exists := s.rules[rule.ID]; !exists {
		s.order = append(s.order, rule.ID)
	}

	s.rules[rule.ID] = rule
	return nil
}

func (s *MemoryRuleStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from order
	for i, orderID := range s.order {
		if orderID == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}

	delete(s.rules, id)
	return nil
}
