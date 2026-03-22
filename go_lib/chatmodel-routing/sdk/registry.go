package sdk

import (
	"fmt"
	"go_lib/chatmodel-routing/adapter"
)

var providers = make(map[string]Factory)

type Factory func(apiKey string) adapter.Provider

func Register(name string, factory Factory) {
	providers[name] = factory
}

func Get(name string, apiKey string) (adapter.Provider, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return factory(apiKey), nil
}
