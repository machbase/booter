package booter

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
)

func ParseFile(path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parse0(nil, filepath.Base(path), content, target)
}

func Parse(content []byte, target any) error {
	return parse0(nil, "__nofile.hcl", content, target)
}

func ParseFileWithContext(envCtx *EnvContext, path string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return parse0(envCtx, filepath.Base(path), content, target)
}

func ParseWithContext(envCtx *EnvContext, content []byte, target any) error {
	return parse0(envCtx, "__nofile.hcl", content, target)
}

func parse0(envCtx *EnvContext, filename string, content []byte, target any) error {
	pass1Ctx := &hcl.EvalContext{}
	if envCtx != nil {
		pass1Ctx.Variables = envCtx.Variables
		pass1Ctx.Functions = envCtx.Functions
	}
	pass1 := &anyhcl{}
	if err := hclsimple.Decode(filename, content, pass1Ctx, pass1); err != nil {
		return errors.Wrap(err, "Parser-pass1")
	}

	body, ok := pass1.Remains.(*hclsyntax.Body)
	if !ok {
		return errors.New("invalid hcl syntax")
	}

	currentLocation := ""
	defer func() {
		ex := recover()
		if ex != nil {
			if err, ok := ex.(error); ok {
				panic(errors.Wrapf(err, "at %s", currentLocation))
			} else {
				panic(fmt.Errorf("at %s, %v", currentLocation, ex))
			}
		}
	}()

	targetRef := reflect.ValueOf(target)
	if targetRef.Kind() == reflect.Pointer {
		targetRef = reflect.Indirect(targetRef)
	}
	for _, attr := range body.Attributes {
		name, value := _eval_attribute(attr, pass1Ctx)
		currentLocation = fmt.Sprintf("%s", name)

		fieldName := toCamelCase(name, true)
		fieldRef := targetRef.FieldByName(fieldName)

		if fieldRef.IsValid() && fieldRef.CanSet() {
			_set_reflect_cty(fieldName, &currentLocation, fieldRef, value)
		} else {
			return errors.New("unknown field")
		}
	}

	return nil
}
