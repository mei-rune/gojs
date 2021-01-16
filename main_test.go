package gojs

import (
	"context"
	"testing"

	"github.com/dop251/goja"
)

func TestNativeCallWithContextParameter(t *testing.T) {
	vm := New()

	valueTrue := vm.ToValue(true)
	valueFalse := vm.ToValue(false)

	vm.Set("f", func(ctx context.Context, _ goja.FunctionCall) goja.Value {
		if ctx.Value("a") == "b" {
			return valueTrue
		}
		return valueFalse
	})

	ctx := context.WithValue(context.Background(), "a", "b")
	ret, err := vm.RunString(ctx, `f()`)
	if err != nil {
		t.Fatal(err)
	}
	if ret != valueTrue {
		t.Fatal(ret)
	}

	ctx = context.WithValue(context.Background(), "a", "c")
	ret, err = vm.RunString(ctx, `f()`)
	if err != nil {
		t.Fatal(err)
	}
	if ret != valueFalse {
		t.Fatal(ret)
	}
}
