package booter

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

type Booter interface {
	Startup() error
	Shutdown()
	ShutdownAndExit(exitCode int)

	WaitSignal()
	NotifySignal()

	GetDefinition(id string) *Definition
	GetInstance(id string) Boot
	GetConfig(id string) any

	AddShutdownHook(...func())
}

type boot struct {
	moduleDefs []*Definition
	wrappers   []wrapper
	quitChan   chan os.Signal

	startupHooks  []func()
	shutdownHooks []func()
}

func NewWithDefinitions(definitions []*Definition) (Booter, error) {
	b := &boot{
		moduleDefs: definitions,
	}
	return b, nil
}

func (this *boot) Startup() error {
	bootlog.Println(len(this.moduleDefs), "modules defined")
	for _, def := range this.moduleDefs {
		state := "enabled"
		if def.Disabled {
			state = "disabled"
		}
		bootlog.Println(def.Id, def.Name, state)

		if def.Disabled {
			continue
		}
		// find factory
		fact := getFactory(def.Id)
		if fact == nil {
			return fmt.Errorf("module %s is not found", def.Id)
		}
		// create config
		config := fact.NewConfig()
		objName := fmt.Sprintf("%T", config)
		if strings.HasPrefix(objName, "*") {
			objName = objName[1:]
		}
		// evalute config values
		err := EvalObject(objName, config, def.Config)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("config %s", objName))
		}
		// create instance
		mod, err := fact.NewInstance(config)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("instance %s", def.Id))
		}
		wrap := wrapper{
			id:         def.Id,
			definition: def,
			real:       mod,
			conf:       config,
			state:      None,
		}
		this.wrappers = append(this.wrappers, wrap)
	}

	// dependency injection
	for _, wrap := range this.wrappers {
		for _, inj := range wrap.definition.Injects {
			if err := wrap.inject(&inj, this.wrappers); err != nil {
				return err
			}
		}
	}
	bootlog.Println(len(this.wrappers), "modules enabled")

	// startup-hook & Start()
	for _, wrap := range this.wrappers {
		wrap.state = Starting
	}
	for _, hook := range this.startupHooks {
		hook()
	}
	for _, wrap := range this.wrappers {
		wrap.state = Starting
		bootlog.Println("start", wrap.id, wrap.definition.Name)
		err := wrap.real.Start()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("mod start %s", wrap.id))
		}
		wrap.state = Run
	}
	return nil
}

func (this *boot) Shutdown() {
	// shutdown-hook & Stop()
	for _, wrap := range this.wrappers {
		wrap.state = Stopping
	}
	for _, hook := range this.shutdownHooks {
		hook()
	}
	for i := len(this.wrappers) - 1; i >= 0; i-- {
		wrap := this.wrappers[i]
		bootlog.Println("stop", wrap.id, wrap.definition.Name)
		instance := wrap.real
		instance.Stop()
		wrap.state = Stop
	}
}

func (this *boot) ShutdownAndExit(exitCode int) {
	this.Shutdown()
	os.Exit(exitCode)
}

func (this *boot) WaitSignal() {
	// signal handler
	this.quitChan = make(chan os.Signal)
	signal.Notify(this.quitChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// wait signal
	<-this.quitChan
}

func (this *boot) NotifySignal() {
	if this.quitChan != nil {
		this.quitChan <- syscall.SIGINT
	}
}

func (this *boot) AddShutdownHook(f ...func()) {
	this.shutdownHooks = append(this.shutdownHooks, f...)
}

func (this *boot) GetDefinition(id string) *Definition {
	for _, def := range this.moduleDefs {
		if def.Id == id {
			return def
		}
	}
	return nil
}

func (this *boot) GetInstance(id string) Boot {
	for _, mod := range this.wrappers {
		if mod.id == id {
			return mod.real
		}
	}
	return nil
}

func (this *boot) GetConfig(id string) any {
	for _, mod := range this.wrappers {
		if mod.id == id {
			return mod.conf
		}
	}
	return nil
}

type wrapper struct {
	id         string
	definition *Definition
	real       Boot
	conf       any
	state      State
}

type State int

const (
	None State = iota
	Starting
	Run
	Stopping
	Stop
)

func (wrap *wrapper) inject(inj *InjectionDef, wrappers []wrapper) error {
	var targetMod Boot
	for _, w := range wrappers {
		if w.definition.Name == inj.Target || w.id == inj.Target {
			targetMod = w.real
			break
		}
	}
	if targetMod == nil {
		return fmt.Errorf("%s inject into %s, not found", wrap.id, inj.Target)
	}
	mod := reflect.ValueOf(targetMod)
	var modPtr reflect.Value
	if mod.Kind() == reflect.Pointer {
		modPtr = mod
		mod = reflect.Indirect(mod)
	}
	field := mod.FieldByName(inj.FieldName)
	if field.IsValid() {
		bootlog.Println(wrap.definition.Name, "inject into", inj.Target, "by field", inj.FieldName)
		field.Set(reflect.ValueOf(wrap.real))
	} else {
		if modPtr.IsValid() {
			setter := modPtr.MethodByName(inj.FieldName)
			if setter.IsValid() {
				bootlog.Println(wrap.definition.Name, "inject into", inj.Target, "by method", inj.FieldName)
				setter.Call([]reflect.Value{reflect.ValueOf(wrap.real)})
			} else {
				return fmt.Errorf("%s %s is not accessible", inj.Target, inj.FieldName)
			}
		} else {
			return fmt.Errorf("%s %s is not accessible", inj.Target, inj.FieldName)
		}
	}
	return nil
}
