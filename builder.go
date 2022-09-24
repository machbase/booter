package booter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

type Builder interface {
	Build(definitions []*Definition) (Booter, error)
	BuildWithContent(content []byte) (Booter, error)
	BuildWithFiles(files []string) (Booter, error)
	BuildWithDir(configDir string) (Booter, error)

	AddStartupHook(hooks ...func())
	AddShutdownHook(hooks ...func())
	SetFunction(name string, f function.Function)
	SetVariable(name string, value any) error
}

type builder struct {
	startupHooks  []func()
	shutdownHooks []func()
	functions     map[string]function.Function
	variables     map[string]cty.Value
}

func NewBuilder() Builder {
	b := &builder{
		functions: make(map[string]function.Function),
		variables: make(map[string]cty.Value),
	}
	for k, v := range DefaultFunctions {
		b.functions[k] = v
	}
	return b
}

func (this *builder) Build(definitions []*Definition) (Booter, error) {
	b, err := NewWithDefinitions(definitions)
	if err != nil {
		return nil, err
	}
	rt := b.(*boot)
	rt.startupHooks = this.startupHooks
	rt.shutdownHooks = this.shutdownHooks
	return rt, nil
}

func (this *builder) BuildWithContent(content []byte) (Booter, error) {
	definitions, err := LoadDefinitions(content, this.makeContext())
	if err != nil {
		return nil, err
	}
	return this.Build(definitions)
}

func (this *builder) BuildWithFiles(files []string) (Booter, error) {
	definitions, err := LoadDefinitionFiles(files, this.makeContext())
	if err != nil {
		return nil, err
	}
	return this.Build(definitions)
}

func (this *builder) BuildWithDir(configDir string) (Booter, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config directory")
	}

	files := make([]string, 0)
	for _, file := range entries {
		if !strings.HasSuffix(file.Name(), ".hcl") {
			continue
		}
		files = append(files, file.Name())
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i] < files[j]
	})
	result := make([]string, 0)
	for _, file := range files {
		result = append(result, filepath.Join(configDir, file))
	}
	return this.BuildWithFiles(result)
}

func (this *builder) AddStartupHook(hooks ...func()) {
	this.startupHooks = append(this.startupHooks, hooks...)
}

func (this *builder) AddShutdownHook(hooks ...func()) {
	this.shutdownHooks = append(this.shutdownHooks, hooks...)
}

func (this *builder) makeContext() *hcl.EvalContext {
	evalCtx := &hcl.EvalContext{
		Functions: this.functions,
		Variables: this.variables,
	}
	if evalCtx.Functions == nil {
		evalCtx.Functions = make(map[string]function.Function)
	}
	return evalCtx
}

func (this *builder) SetFunction(name string, f function.Function) {
	this.functions[name] = f
}

func (this *builder) SetVariable(name string, value any) (err error) {
	if len(name) == 0 {
		return errors.New("can not define with empty name")
	}
	var v cty.Value
	switch raw := value.(type) {
	case string:
		v, err = gocty.ToCtyValue(raw, cty.String)
	case bool:
		v, err = gocty.ToCtyValue(raw, cty.Bool)
	case int, int32, int64, float32, float64:
		v, err = gocty.ToCtyValue(raw, cty.Number)
	default:
		return fmt.Errorf("can not define %s with value type %T", name, value)
	}

	if err == nil {
		this.variables[name] = v
	}
	return
}
