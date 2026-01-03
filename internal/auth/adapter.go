package auth

import (
	"context"
	"errors"

	"github.com/bher20/eratemanager/internal/storage"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// Adapter implements the Casbin persist.Adapter interface using storage.Storage.
type Adapter struct {
	storage storage.Storage
}

// NewAdapter returns a new Casbin adapter.
func NewAdapter(s storage.Storage) *Adapter {
	return &Adapter{storage: s}
}

// LoadPolicy loads all policy rules from the storage.
func (a *Adapter) LoadPolicy(model model.Model) error {
	rules, err := a.storage.LoadCasbinRules(context.Background())
	if err != nil {
		return err
	}

	for _, rule := range rules {
		line := rule.PType
		if rule.V0 != "" {
			line += ", " + rule.V0
		}
		if rule.V1 != "" {
			line += ", " + rule.V1
		}
		if rule.V2 != "" {
			line += ", " + rule.V2
		}
		if rule.V3 != "" {
			line += ", " + rule.V3
		}
		if rule.V4 != "" {
			line += ", " + rule.V4
		}
		if rule.V5 != "" {
			line += ", " + rule.V5
		}
		persist.LoadPolicyLine(line, model)
	}
	return nil
}

// SavePolicy saves all policy rules to the storage.
func (a *Adapter) SavePolicy(model model.Model) error {
	// We don't implement SavePolicy because we use incremental Add/RemovePolicy.
	// If we needed to support SavePolicy, we would need to clear the table and re-insert everything.
	return errors.New("not implemented")
}

// AddPolicy adds a policy rule to the storage.
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	r := storage.CasbinRule{PType: ptype}
	if len(rule) > 0 {
		r.V0 = rule[0]
	}
	if len(rule) > 1 {
		r.V1 = rule[1]
	}
	if len(rule) > 2 {
		r.V2 = rule[2]
	}
	if len(rule) > 3 {
		r.V3 = rule[3]
	}
	if len(rule) > 4 {
		r.V4 = rule[4]
	}
	if len(rule) > 5 {
		r.V5 = rule[5]
	}
	return a.storage.AddCasbinRule(context.Background(), r)
}

// RemovePolicy removes a policy rule from the storage.
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	r := storage.CasbinRule{PType: ptype}
	if len(rule) > 0 {
		r.V0 = rule[0]
	}
	if len(rule) > 1 {
		r.V1 = rule[1]
	}
	if len(rule) > 2 {
		r.V2 = rule[2]
	}
	if len(rule) > 3 {
		r.V3 = rule[3]
	}
	if len(rule) > 4 {
		r.V4 = rule[4]
	}
	if len(rule) > 5 {
		r.V5 = rule[5]
	}
	return a.storage.RemoveCasbinRule(context.Background(), r)
}

// RemoveFilteredPolicy removes policy rules that match the filter from the storage.
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	// This is required for RemovePolicy to work correctly in some cases, or for API calls.
	// Since our storage interface doesn't support complex filtering yet, we can implement a basic version
	// or just return error.
	// However, Casbin often calls RemoveFilteredPolicy when RemovePolicy is called? No, RemovePolicy calls RemovePolicy.
	// But UpdatePolicy might call RemoveFilteredPolicy.

	// For now, let's implement a simple loop if we can't do it in DB efficiently without changing interface.
	// But wait, we can't loop over DB easily.
	// Let's just return error for now and see if it breaks anything.
	// Actually, `RemovePolicy` is what we use most.
	return errors.New("not implemented")
}
