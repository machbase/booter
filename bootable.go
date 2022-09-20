package booter

import (
	"fmt"
	"sync"
)

type Bootable interface {
	Start() error
	Stop()
}

type BootFactory struct {
	Id          string
	NewConfig   func() any
	NewInstance func(config any) (Bootable, error)
}

var factoryRegistry = make(map[string]*BootFactory)
var factoryRegistryLock sync.Mutex

func RegisterBootFactory(def *BootFactory) {
	factoryRegistryLock.Lock()
	if _, exists := factoryRegistry[def.Id]; !exists {
		factoryRegistry[def.Id] = def
	}
	factoryRegistryLock.Unlock()
}

func UnregisterBootFactory(id string) {
	delete(factoryRegistry, id)
}

func getFactory(id string) *BootFactory {
	if obj, ok := factoryRegistry[id]; ok {
		return obj
	}
	return nil
}

func Register[T any](moduleId string, configFactory func() T, factory func(conf T) (Bootable, error)) {
	RegisterBootFactory(&BootFactory{
		Id: moduleId,
		NewConfig: func() any {
			return configFactory()
		},
		NewInstance: func(conf any) (Bootable, error) {
			if c, ok := conf.(T); ok {
				return factory(c)
			} else {
				return nil, fmt.Errorf("invalid config type: %T", conf)
			}
		},
	})
}
