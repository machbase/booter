package booter

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

type Boot interface {
	Startup() error
	Shutdown()

	WaitSignal()
	NotifySignal()

	GetDefinition(id string) *Definition
	GetInstance(id string) Bootable
	GetConfig(id string) any
}

type boot struct {
	moduleDefs []*Definition
	wrappers   []wrapper
	quitChan   chan os.Signal
}

type wrapper struct {
	id         string
	definition *Definition
	real       Bootable
	conf       any
	state      State
}

type State int

const (
	None State = iota
	PreStart
	Starting
	Run
	Stopping
	Stop
)

func New(configDir string) (Boot, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config directory")
	}

	files := make([]string, 0)
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".hcl") {
			continue
		}
		files = append(files, filepath.Join(configDir, file.Name()))
	}
	return NewWithFiles(files)
}

func NewWithFiles(files []string) (Boot, error) {
	b := &boot{}

	definitions, err := LoadDefinitions(files)
	if err != nil {
		return nil, err
	}
	b.moduleDefs = definitions
	return b, nil
}

type Hook map[string]func()

var PreStartHook map[string]func()
var PreStopHook map[string]func()

func (this *boot) Startup() error {
	for _, def := range this.moduleDefs {
		if def.Disabled {
			continue
		}
		// find factory
		fact := getBootFactory(def.Id)
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

	// pre-start
	for _, wrap := range this.wrappers {
		wrap.state = PreStart
		if hook, ok := PreStartHook[wrap.id]; ok {
			hook()
		}
	}

	// start & post-start
	for _, wrap := range this.wrappers {
		wrap.state = Starting
		err := wrap.real.Start()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("mod start %s", wrap.id))
		}
		wrap.state = Run
	}
	return nil
}

func (this *boot) Shutdown() {
	for i := len(this.wrappers); i >= 0; i-- {
		mod := this.wrappers[i]
		mod.state = Stopping
		if hook, ok := PreStopHook[mod.id]; ok {
			hook()
		}
		instance := mod.real
		instance.Stop()
		mod.state = Stop
	}
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

func (this *boot) GetDefinition(id string) *Definition {
	for _, def := range this.moduleDefs {
		if def.Id == id {
			return def
		}
	}
	return nil
}

func (this *boot) GetInstance(id string) Bootable {
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
