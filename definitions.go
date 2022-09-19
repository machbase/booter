package booter

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
)

type Definition struct {
	Id       string            `hcl:"id,label"`
	Priority int               `hcl:"priority,optional"`
	Disabled bool              `hcl:"disabled,optional"`
	Prefix   string            `hcl:"prefix,optional"`
	Config   *ConfigDefinition `hcl:",block"`
}

type ConfigDefinition struct {
	Name    string `hcl:"name,label"`
	Remains any    `hcl:",remain"`
}

func loadModuleConfig(envCtx *EnvContext, path string, args []string) ([]*Definition, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "loadModuleConfig read file")
	}

	////////////////////////////////////////////////////////
	// module을 제외한 block들을 모두 변수 정의로 간주하고 읽어들인다.
	pass1Ctx := &hcl.EvalContext{}
	if envCtx != nil {
		pass1Ctx.Variables = envCtx.Variables
		pass1Ctx.Functions = envCtx.Functions
	}
	type anyhcl struct {
		Remains any `hcl:",remain"`
	}
	pass1 := &anyhcl{}
	if err := hclsimple.Decode(path, content, pass1Ctx, pass1); err != nil {
		return nil, errors.Wrap(err, "loadModuleConfig pass1")
	}

	body, ok := pass1.Remains.(*hclsyntax.Body)
	if !ok {
		return nil, errors.Wrap(err, "loadModuleConfig invalid syntax")
	}

	vars := make(map[string]cty.Value)
	for _, block := range body.Blocks {
		if block.Type == "module" {
			continue
		}
		obj := make(map[string]cty.Value)
		for _, attr := range block.Body.Attributes {
			name, value := _eval_attribute(attr, nil)
			obj[name] = value
		}
		vars[block.Type] = cty.ObjectVal(obj)
	}
	////////////////////////////////////////////////////////
	// module block을 읽을 때 적용할 context를 정의한다.
	ctx := pass1Ctx.NewChild()
	ctx.Variables = vars

	////////////////////////////////////////////////////////
	// module block들을 읽어들인다.
	type moduleConf2Pass struct {
		Modules []*Definition `hcl:"module,block"`
		Remains hcl.Body      `hcl:",remain"`
	}

	// https://hcl.readthedocs.io/en/latest/go_decoding_hcldec.html
	// spec := hcldec.ObjectSpec{
	// 	"module": &hcldec.BlockMapSpec{},
	// }

	pass2 := &moduleConf2Pass{}
	if err := hclsimple.Decode(path, content, ctx, pass2); err != nil {
		return nil, errors.Wrap(err, "loadModuleConfig pass2")
	}

	enabledDefs := make([]*Definition, 0)
	for _, mod := range pass2.Modules {
		if mod.Disabled {
			continue
		}
		md := getBootFactory(mod.Id)
		if md == nil {
			return nil, fmt.Errorf("module factory not found: %s", mod.Id)
		}
		mdCfg := md.NewConfig()
		if mdCfg == nil {
			// config가 정의되지 않은 module
			enabledDefs = append(enabledDefs, mod)
			continue
		}

		mdCfgRef := reflect.ValueOf(mdCfg)
		if mdCfgRef.Kind() == reflect.Pointer {
			mdCfgRef = reflect.Indirect(mdCfgRef)
		}

		fmt.Printf("==> %#v\n", mod.Config)
		/*
			attr := mod.Config.(*hcl.Attribute)
			val, diagnostic := attr.Expr.Value(ctx)
			if diagnostic != nil {
				return nil, errors.New(diagnostic.Error())
			}
			iter := val.ElementIterator()
			for iter.Next() {
				k, v := iter.Element()
				snakeFieldName := k.AsString()
				currentLocation = fmt.Sprintf("%s '%s'", mod.Id, snakeFieldName)

				fieldName := toCamelCase(snakeFieldName, true)
				fieldRef := mdCfgRef.FieldByName(fieldName)

				if fieldRef.IsValid() && fieldRef.CanSet() {
					_set_reflect_cty(fieldName, &currentLocation, fieldRef, v)
				} else {
					panic(errors.New("unknown field"))
				}
			}

			// command line arguments
			if len(mod.Prefix) > 0 {
				prefix := fmt.Sprintf("--%s", mod.Prefix)
				for i, arg := range args {
					if strings.HasPrefix(arg, prefix) {
						snakeFieldName := arg[len(prefix):]
						fieldName := toCamelCase(snakeFieldName, true)
						fieldRef := mdCfgRef.FieldByName(fieldName)

						if fieldRef.IsValid() && fieldRef.CanSet() {
							_set_reflect_flag(fieldName, fieldRef, args, i)
						} else {
							panic(fmt.Errorf("unknown flag: %s", arg))
						}
					}
				}

				// set materialized config
				mod.Config = mdCfg
			}
		*/
		enabledDefs = append(enabledDefs, mod)
	}

	return enabledDefs, nil
}

func _eval_attribute(attr *hclsyntax.Attribute, ctx *hcl.EvalContext) (string, cty.Value) {
	var name = fmt.Sprintf("%s", attr.Name)
	var value cty.Value
	switch expr := attr.Expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		value = expr.Val
	case *hclsyntax.TemplateExpr:
		value, _ = expr.Value(ctx)
	case *hclsyntax.FunctionCallExpr:
		value, _ = expr.Value(ctx)
		// fmt.Printf("==> name: %s \n==> expr: %#v\n==>valu: %#v\n", name, expr, value)
	default:
		panic(fmt.Errorf("Unknown attribute type %-20s %T\n", attr.Name, attr.Expr))
	}
	return name, value
}

func _set_reflect_flag(fieldName string, fieldRef reflect.Value, args []string, idx int) {
	switch fieldRef.Kind() {
	case reflect.Bool:
		fieldRef.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(args[idx+1], 10, 64)
		if err != nil {
			panic(err)
		}
		fieldRef.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(args[idx+1], 10, 64)
		if err != nil {
			panic(err)
		}
		fieldRef.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(args[idx+1], 64)
		if err != nil {
			panic(err)
		}
		fieldRef.SetFloat(v)
	case reflect.String:
		fieldRef.SetString(args[idx+1])
	case reflect.Array:
		slice := strings.Split(args[idx+1], ",")
		fieldRef.Set(reflect.ValueOf(slice))
	case reflect.Slice:
		slice := strings.Split(args[idx+1], ",")
		fieldRef.Set(reflect.ValueOf(slice))
	default:
		panic(fmt.Errorf("unsupported reflection type: %s for %s", fieldRef.Kind(), fieldName))
	}
}

func _parse_int(str string) int64 {
	var v int64
	var err error
	if strings.HasSuffix(str, "ms") {
		if v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64); err == nil {
			return v * int64(time.Millisecond)
		}
	} else if strings.HasSuffix(str, "s") {
		if v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64); err == nil {
			return v * int64(time.Second)
		}
	} else if strings.HasSuffix(str, "m") {
		if v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64); err == nil {
			return v * int64(time.Minute)
		}
	} else if strings.HasSuffix(str, "h") {
		if v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64); err == nil {
			return v * int64(time.Hour)
		}
	} else if strings.HasSuffix(str, "d") {
		if v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64); err == nil {
			return v * int64(time.Hour) * 24
		}
	}

	v, err = strconv.ParseInt(str[0:len(str)-1], 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func _set_reflect_cty(fieldName string, currentLocation *string, fieldRef reflect.Value, value cty.Value) {
	switch fieldRef.Kind() {
	case reflect.Bool:
		fieldRef.SetBool(value.True())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value.Type() == cty.Number {
			bf := value.AsBigFloat()
			v, _ := bf.Int64()
			fieldRef.SetInt(v)
		} else if value.Type() == cty.String {
			fieldRef.SetInt(_parse_int(value.AsString()))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bf := value.AsBigFloat()
		v, _ := bf.Uint64()
		fieldRef.SetUint(v)
	case reflect.Float32, reflect.Float64:
		bf := value.AsBigFloat()
		v, _ := bf.Float64()
		fieldRef.SetFloat(v)
	case reflect.String:
		fieldRef.SetString(value.AsString())
	case reflect.Array:
	case reflect.Slice:
		vs := value.AsValueSlice()
		slice := reflect.MakeSlice(fieldRef.Type(), len(vs), len(vs))
		for i, elm := range vs {
			elmName := fmt.Sprintf("%s[%d]'s value", fieldName, i)
			*currentLocation = fmt.Sprintf("%s - %s", *currentLocation, elmName)
			_set_reflect_cty(elmName, currentLocation, slice.Index(i), elm)
		}
		fieldRef.Set(slice)
	case reflect.Map:
		vm := value.AsValueMap()
		maps := reflect.MakeMap(fieldRef.Type())
		keyType := fieldRef.Type().Key()
		if keyType.Kind() != reflect.String {
			panic(fmt.Errorf("unsupported map key type: %v", keyType))
		}
		valType := fieldRef.Type().Elem()
		if valType.Kind() == reflect.Pointer {
			fmt.Printf("pointer map val type: %v", valType)
		}
		for k, v := range vm {
			val := reflect.Indirect(reflect.New(valType))
			*currentLocation = fmt.Sprintf("%s, '%s'", *currentLocation, k)
			_set_reflect_cty(fmt.Sprintf("%s[\"%s\"]", fieldName, k), currentLocation, val, v)
			maps.SetMapIndex(reflect.ValueOf(k), val)
		}
		fieldRef.Set(maps)
	case reflect.Struct:
		vm := value.AsValueMap()
		// fmt.Printf("======> %#v\n------> %#v\n", fieldRef, vm)
		for k, v := range vm {
			elmName := toCamelCase(k, true)
			elmRef := fieldRef.FieldByName(elmName)
			*currentLocation = fmt.Sprintf("%s, '%s'", *currentLocation, elmName)
			_set_reflect_cty(fmt.Sprintf("%s.%s", fieldName, elmName), currentLocation, elmRef, v)
		}
	default:
		panic(fmt.Errorf("unsupported reflection type: %s for %s", fieldRef.Kind(), fieldName))
	}
}

// Converts a string to CamelCase
var uppercaseAcronym = map[string]string{
	"ID": "id",
}

func toCamelCase(s string, initCase bool) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if a, ok := uppercaseAcronym[s]; ok {
		s = a
	}

	n := strings.Builder{}
	n.Grow(len(s))
	capNext := initCase
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if capNext {
			if vIsLow {
				v += 'A'
				v -= 'a'
			}
		} else if i == 0 {
			if vIsCap {
				v += 'a'
				v -= 'A'
			}
		}
		if vIsCap || vIsLow {
			n.WriteByte(v)
			capNext = false
		} else if vIsNum := v >= '0' && v <= '9'; vIsNum {
			n.WriteByte(v)
			capNext = true
		} else {
			capNext = v == '_' || v == ' ' || v == '-' || v == '.'
		}
	}
	return n.String()
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if a, ok := uppercaseAcronym[s]; ok {
		s = a
	}

	snake := matchFirstCap.ReplaceAllString(s, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
