package gojs

import (
	"context"

	"github.com/dop251/goja"
)

func New() *Runtime {
	return &Runtime{
		Runtime: goja.New(),
	}
}

type Runtime struct {
	*goja.Runtime
	ctx context.Context
}

func (r *Runtime) RunString(ctx context.Context, str string) (goja.Value, error) {
	r.ctx = ctx
	return r.Runtime.RunString(str)
}

func (r *Runtime) RunScript(ctx context.Context, name, src string) (goja.Value, error) {
	r.ctx = ctx
	return r.Runtime.RunScript(name, src)
}

func (r *Runtime) RunProgram(ctx context.Context, p *goja.Program) (goja.Value, error) {
	r.ctx = ctx
	return r.Runtime.RunProgram(p)
}

func (r *Runtime) Set(name string, value interface{}) {
	switch i := value.(type) {
	case func(context.Context, goja.FunctionCall) goja.Value:
		r.Runtime.Set(name, func(call goja.FunctionCall) goja.Value {
			return i(r.ctx, call)
		})
	case func(context.Context, goja.ConstructorCall) *goja.Object:
		r.Runtime.Set(name, func(call goja.ConstructorCall) *goja.Object {
			return i(r.ctx, call)
		})
	default:
		r.Runtime.Set(name, value)
	}
}

func (r *Runtime) ToValue(i interface{}) goja.Value {
	switch i := i.(type) {
	case func(context.Context, goja.FunctionCall) goja.Value:
		return r.Runtime.ToValue(func(call goja.FunctionCall) goja.Value {
			return i(r.ctx, call)
		})
	case func(context.Context, goja.ConstructorCall) *goja.Object:
		return r.Runtime.ToValue(func(call goja.ConstructorCall) *goja.Object {
			return i(r.ctx, call)
		})
	default:
		return r.Runtime.ToValue(i)
	}
}
