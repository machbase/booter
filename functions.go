package booter

import (
	"os"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var GetEnvFunc = function.New(&function.Spec{
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
		out := os.Getenv(in)
		return cty.StringVal(out), nil
	},
})

var GetEnv2Func = function.New(&function.Spec{
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
