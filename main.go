package gojs

import (
	"context"

	"github.com/dop251/goja"
	"github.com/runner-mei/gojs/js/compiler"
	jslib "github.com/runner-mei/gojs/js/lib"
)

type runtimeCtxKey struct{}

func (key *runtimeCtxKey) String() string {
	return "js-runtime"
}

var (
	ctxKeyRuntime = &runtimeCtxKey{}
)

// WithRuntime attaches the given goja runtime to the context.
func WithRuntime(ctx context.Context, rt *Runtime) context.Context {
	return context.WithValue(ctx, ctxKeyRuntime, rt)
}

// GetRuntime retrieves the attached goja runtime from the given context.
func GetRuntime(ctx context.Context) *Runtime {
	v := ctx.Value(ctxKeyRuntime)
	if v == nil {
		return nil
	}
	return v.(*Runtime)
}

func New() *Runtime {
	r, err := NewWith(nil)
	if err != nil {
		panic(err)
	}
	return r
}

func NewWith(opts *RuntimeOptions) (*Runtime, error) {
	if opts == nil {
		opts = &RuntimeOptions{}
	}

	compatMode, err := ValidateCompatibilityMode(opts.CompatibilityMode)
	if err != nil {
		return nil, err
	}

	rt := &Runtime{
		CompatibilityMode: compatMode,
		Compiler:          compiler.New(),
		Runtime:           goja.New(),
	}
	rt.Runtime.SetFieldNameMapper(FieldNameMapper{})
	rt.Runtime.SetRandSource(NewRandSource())
	if compatMode == compiler.CompatibilityModeExtended {
		if _, err := rt.Runtime.RunProgram(jslib.GetCoreJS()); err != nil {
			return nil, err
		}
	}

	rt.Set("__ENV", opts.Env)
	return rt, nil
}

type Runtime struct {
	CompatibilityMode compiler.CompatibilityMode

	*compiler.Compiler
	*goja.Runtime
	ctx context.Context
}

// Compile the program in the given CompatibilityMode, wrapping it between pre and post code
func (r *Runtime) Compile(filename, src, pre, post string,
	strict bool, compatMode CompatibilityMode) (*goja.Program, string, error) {
	return r.Compiler.Compile(src, filename, pre, post, strict, r.CompatibilityMode)
}

func (r *Runtime) RunString(ctx context.Context, str string) (goja.Value, error) {
	r.ctx = WithRuntime(ctx, r)
	return r.Runtime.RunString(str)
}

func (r *Runtime) RunScript(ctx context.Context, name, src string) (goja.Value, error) {
	r.ctx = WithRuntime(ctx, r)
	return r.Runtime.RunScript(name, src)
}

func (r *Runtime) RunProgram(ctx context.Context, p *goja.Program) (goja.Value, error) {
	r.ctx = WithRuntime(ctx, r)
	return r.Runtime.RunProgram(p)
}

func (r *Runtime) convertValue(value interface{}) interface{} {
	switch i := value.(type) {
	case func(context.Context, goja.FunctionCall) goja.Value:
		return func(call goja.FunctionCall) goja.Value {
			return i(r.ctx, call)
		}
	case func(context.Context, goja.ConstructorCall) *goja.Object:
		return func(call goja.ConstructorCall) *goja.Object {
			return i(r.ctx, call)
		}
	case map[string]interface{}:
		newValues := make(map[string]interface{}, len(i))
		for k, v := range i {
			newValues[k] = r.convertValue(v)
		}
		return newValues
	default:
		return value
	}
}

func (r *Runtime) Set(name string, value interface{}) {
	r.Runtime.Set(name, r.convertValue(value))
}

func (r *Runtime) ToValue(i interface{}) goja.Value {
	return r.Runtime.ToValue(r.convertValue(i))
}

// Instantiates the bundle into an existing runtime. Not public because it also messes with a bunch
// of other things, will potentially thrash data and makes a mess in it if the operation fails.
func InstantiateEnv(rt *Runtime) *goja.Object {
	exports := rt.Runtime.NewObject()
	rt.Runtime.Set("exports", exports)
	module := rt.Runtime.NewObject()
	_ = module.Set("exports", exports)
	rt.Runtime.Set("module", module)
	return exports
}
