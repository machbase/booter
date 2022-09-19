package booter

import "sync"

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

func getBootFactory(id string) *BootFactory {
	return factoryRegistry[id]
}

func Register(moduleId string, configFactory func() any, factory func(conf any) (Bootable, error)) {
	RegisterBootFactory(&BootFactory{
		Id:          moduleId,
		NewConfig:   configFactory,
		NewInstance: factory,
	})
}
