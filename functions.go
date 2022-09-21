package booter

import (
	"fmt"
	"os"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

func SetFunction(name string, f function.Function) {
	predefFunctions[name] = f
}

var predefFunctions = map[string]function.Function{
	"env":         GetEnvFunc,
	"envOrError":  GetEnv2Func,
	"flag":        GetFlagFunc,
	"flagOrError": GetFlag2Func,
	"pname":       GetPnameFunc,
	"version":     GetVersionFunc,
	"upper":       stdlib.UpperFunc,
	"lower":       stdlib.LowerFunc,
	"min":         stdlib.MinFunc,
	"max":         stdlib.MaxFunc,
	"strlen":      stdlib.StrlenFunc,
	"substr":      stdlib.SubstrFunc,
}

var GetPnameFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(Pname()), nil
	},
})

var GetVersionFunc = function.New(&function.Spec{
	Params: []function.Parameter{},
	Type:   function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(VersionString()), nil
	},
})

var GetEnv2Func = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "env",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out, ok := os.LookupEnv(in)
		if !ok {
			return cty.NilVal, fmt.Errorf("required env variable %s missing", in)
		}
		return cty.StringVal(out), nil
	},
})

var GetEnvFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "env",
			Type:             cty.String,
			AllowDynamicType: true,
		},
		{
			Name:      "default",
			Type:      cty.String,
			AllowNull: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		def := ""
		if !args[1].IsNull() {
			def = args[1].AsString()
		}
		out, ok := os.LookupEnv(in)
		if !ok {
			out = def
		}
		return cty.StringVal(out), nil
	},
})

var GetFlag2Func = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "flag",
			Type:             cty.String,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := ""
		for i, arg := range os.Args {
			if arg == in {
				if i < len(os.Args)-1 {
					out = os.Args[i+1]
				}
				return cty.StringVal(out), nil
			}
		}
		return cty.NilVal, fmt.Errorf("required flag %s missing", in)
	},
})

var GetFlagFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "flag",
			Type:             cty.String,
			AllowDynamicType: true,
		},
		{
			Name:      "default",
			Type:      cty.String,
			AllowNull: true,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		in := args[0].AsString()
		out := ""
		if !args[1].IsNull() {
			out = args[1].AsString()
		}
		for i, arg := range os.Args {
			if arg == in {
				if i < len(os.Args)-1 {
					out = os.Args[i+1]
				}
				break
			}
		}
		return cty.StringVal(out), nil
	},
})
