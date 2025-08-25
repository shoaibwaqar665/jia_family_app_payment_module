package pricing

import "sync"

// MemoryRuleStore is an in-memory implementation of RuleRepository.
type MemoryRuleStore struct {
	mu    sync.RWMutex
	rules map[string]PricingRule
}

func NewMemoryRuleStore() *MemoryRuleStore {
	return &MemoryRuleStore{rules: make(map[string]PricingRule)}
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
	for _, r := range s.rules {
		if r.Enabled {
			out = append(out, r)
		}
	}
	return out
}

func (s *MemoryRuleStore) Upsert(rule PricingRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.ID] = rule
	return nil
}

func (s *MemoryRuleStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rules, id)
	return nil
}
