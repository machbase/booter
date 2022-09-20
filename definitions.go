package booter

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type Definition struct {
	Id       string
	Priority int
	Disabled bool
	Config   cty.Value
}

func LoadDefinitions(files []string) ([]*Definition, error) {
	body, err := readFile(files)
	if err != nil {
		return nil, err
	}

	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "module", LabelNames: []string{"id"}},
			{Type: "define", LabelNames: []string{"id"}},
		},
	}
	content, diag := body.Content(schema)
	if diag.HasErrors() {
		return nil, errors.New(diag.Error())
	}

	defines := make([]*hcl.Block, 0)
	modules := make([]*hcl.Block, 0)

	for _, block := range content.Blocks {
		if block.Type == "define" {
			defines = append(defines, block)
		} else if block.Type == "module" {
			modules = append(modules, block)
		}
	}

	functions := map[string]function.Function{
		"env":    GetEnvFunc,
		"env2":   GetEnv2Func,
		"flag":   GetFlagFunc,
		"upper":  stdlib.UpperFunc,
		"lower":  stdlib.LowerFunc,
		"min":    stdlib.MinFunc,
		"max":    stdlib.MaxFunc,
		"strlen": stdlib.StrlenFunc,
		"substr": stdlib.SubstrFunc,
	}

	variables := make(map[string]cty.Value)
	for _, d := range defines {
		id := d.Labels[0]
		sb := d.Body.(*hclsyntax.Body)
		for _, attr := range sb.Attributes {
			name := fmt.Sprintf("%s_%s", id, attr.Name)
			value, diag := attr.Expr.Value(nil)
			if diag.HasErrors() {
				return nil, errors.New(diag.Error())
			}
			variables[name] = value
		}
	}

	evalCtx := &hcl.EvalContext{
		Variables: variables,
		Functions: functions,
	}

	schema = &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "priority", Required: false},
			{Name: "disabled", Required: false},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "config", LabelNames: []string{}},
		},
	}

	result := make([]*Definition, 0)
	for _, m := range modules {
		moduleId := m.Labels[0]
		moduleDef := &Definition{Id: moduleId}

		content, diag := m.Body.Content(schema)
		if diag.HasErrors() {
			return nil, errors.New(diag.Error())
		}
		// module attributes
		for _, attr := range content.Attributes {
			name := attr.Name
			value, diag := attr.Expr.Value(evalCtx)
			if diag.HasErrors() {
				return nil, errors.New(diag.Error())
			}
			switch name {
			case "priority":
				moduleDef.Priority = PriorityFromCty(value)
			case "disabled":
				moduleDef.Disabled, _ = BoolFromCty(value)
			}
		}
		for _, c := range content.Blocks {
			if c.Type == "config" {
				obj, err := ObjectValFromBody(c.Body.(*hclsyntax.Body), evalCtx)
				if err != nil {
					return nil, err
				}
				moduleDef.Config = obj
			} else {
				return nil, fmt.Errorf("unknown block %s", c.Type)
			}
		}
		result = append(result, moduleDef)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result, nil
}

func readFile(files []string) (hcl.Body, error) {
	hclFiles := make([]*hcl.File, 0)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		hclFile, hclDiag := hclsyntax.ParseConfig(content, file, hcl.Pos{})
		if hclDiag.HasErrors() {
			return nil, errors.New(hclDiag.Error())
		}
		hclFiles = append(hclFiles, hclFile)
	}
	return hcl.MergeFiles(hclFiles), nil
}

func ObjectValFromBody(body *hclsyntax.Body, evalCtx *hcl.EvalContext) (cty.Value, error) {
	rt := make(map[string]cty.Value)
	for _, attr := range body.Attributes {
		value, diag := attr.Expr.Value(evalCtx)
		if diag.HasErrors() {
			return cty.NilVal, errors.New(diag.Error())
		}
		rt[attr.Name] = value
	}
	for _, block := range body.Blocks {
		bval, err := ObjectValFromBody(block.Body, evalCtx)
		if err != nil {
			return cty.NilVal, err
		}
		rt[block.Type] = bval
	}
	return cty.ObjectVal(rt), nil
}

func _set_reflect_flag(refName string, ref reflect.Value, args []string, idx int) {
	switch ref.Kind() {
	case reflect.Bool:
		ref.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(args[idx+1], 10, 64)
		if err != nil {
			panic(err)
		}
		ref.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(args[idx+1], 10, 64)
		if err != nil {
			panic(err)
		}
		ref.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(args[idx+1], 64)
		if err != nil {
			panic(err)
		}
		ref.SetFloat(v)
	case reflect.String:
		ref.SetString(args[idx+1])
	case reflect.Array:
		slice := strings.Split(args[idx+1], ",")
		ref.Set(reflect.ValueOf(slice))
	case reflect.Slice:
		slice := strings.Split(args[idx+1], ",")
		ref.Set(reflect.ValueOf(slice))
	default:
		panic(fmt.Errorf("unsupported reflection type: %s for %s", ref.Kind(), refName))
	}
}

func EvalObject(objName string, obj any, value cty.Value) error {
	ref := reflect.ValueOf(obj)
	return EvalReflectValue(objName, ref, value)
}

func EvalReflectValue(refName string, ref reflect.Value, value cty.Value) error {
	if ref.Kind() == reflect.Pointer {
		ref = reflect.Indirect(ref)
	}
	switch ref.Kind() {
	case reflect.Struct:
		if value.Type().IsObjectType() {
			valmap := value.AsValueMap()
			for k, v := range valmap {
				field := ref.FieldByName(k)
				if !field.IsValid() {
					return fmt.Errorf("%s field not found in %s", k, refName)
				}
				err := EvalReflectValue(fmt.Sprintf("%s.%s", refName, k), field, v)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("%s should be object", refName)
		}
	case reflect.String:
		if value.Type() == cty.String {
			ref.SetString(value.AsString())
		} else {
			return fmt.Errorf("%s should be string", refName)
		}
	case reflect.Bool:
		if value.Type() == cty.Bool || value.Type() == cty.String {
			if v, err := BoolFromCty(value); err != nil {
				return err
			} else {
				ref.SetBool(v)
			}
		} else {
			return fmt.Errorf("%s should be bool", refName)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Type() == cty.Number || value.Type() == cty.String {
			if v, err := Int64FromCty(value); err != nil {
				return err
			} else {
				ref.SetInt(v)
			}
		} else {
			return fmt.Errorf("%s should be int", refName)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Type() == cty.Number || value.Type() == cty.String {
			if v, err := Uint64FromCty(value); err != nil {
				return err
			} else {
				ref.SetUint(v)
			}
		} else {
			return fmt.Errorf("%s should be uint", refName)
		}
	case reflect.Slice:
		vs := value.AsValueSlice()
		slice := reflect.MakeSlice(ref.Type(), len(vs), len(vs))
		for i, elm := range vs {
			elmName := fmt.Sprintf("%s[%d]", refName, i)
			err := EvalReflectValue(elmName, slice.Index(i), elm)
			if err != nil {
				return err
			}
		}
		ref.Set(slice)
	case reflect.Map:
		vm := value.AsValueMap()
		maps := reflect.MakeMap(ref.Type())
		keyType := ref.Type().Key()
		if keyType.Kind() != reflect.String {
			panic(fmt.Errorf("unsupported map key type: %v", keyType))
		}
		valType := ref.Type().Elem()
		if valType.Kind() == reflect.Pointer {
			fmt.Printf("pointer map val type: %v", valType)
		}
		for k, v := range vm {
			val := reflect.Indirect(reflect.New(valType))
			elmName := fmt.Sprintf("%s[\"%s\"]", refName, k)
			EvalReflectValue(elmName, val, v)
			maps.SetMapIndex(reflect.ValueOf(k), val)
		}
		ref.Set(maps)
	default:
		return fmt.Errorf("unsupported reflection %s type: %s", refName, ref.Kind())
	}
	return nil
}
