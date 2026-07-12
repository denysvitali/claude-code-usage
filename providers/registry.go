// Package providers is the compiled provider registry.
package providers

import (
	"fmt"
	"sort"
)

// Capability describes what a provider can expose.
type Capability struct {
	ID          string
	Name        string
	Auth        string
	Implemented bool
}

// All returns a deterministic copy of the provider registry.
func All() []Capability {
	result := make([]Capability, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, definition.Capability)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// Lookup returns the capability for a provider ID, or an error if unknown.
func Lookup(id string) (Capability, error) {
	if definition, ok := definitionFor(id); ok {
		return definition.Capability, nil
	}
	return Capability{}, fmt.Errorf("unknown provider %q", id)
}
