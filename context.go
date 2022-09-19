package booter

import (
	"os"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type EnvContext struct {
	Variables map[string]cty.Value
	Functions map[string]function.Function
}

func LoadEnvContext(path string) (*EnvContext, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type anyhcl struct {
		Remains any `hcl:",remain"`
	}

	env := &anyhcl{}
	if err := hclsimple.Decode(path, content, nil, env); err != nil {
		return nil, err
	}

	body, ok := env.Remains.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("invalid hcl syntax")
	}

	vars := make(map[string]cty.Value)
	for _, block := range body.Blocks {
		obj := make(map[string]cty.Value)
		for _, attr := range block.Body.Attributes {
			name, value := _eval_attribute(attr, nil)
			obj[name] = value
		}
		vars[block.Type] = cty.ObjectVal(obj)
	}

	rt := &EnvContext{
		Variables: vars,
		Functions: map[string]function.Function{
			"env":    GetEnvFunc,
			"env2":   GetEnv2Func,
			"upper":  stdlib.UpperFunc,
			"lower":  stdlib.LowerFunc,
			"min":    stdlib.MinFunc,
			"max":    stdlib.MaxFunc,
			"strlen": stdlib.StrlenFunc,
			"substr": stdlib.SubstrFunc,
		},
	}
	return rt, nil
}
