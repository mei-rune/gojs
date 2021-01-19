/*
 * this is copy from k6
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package gojs

import (
	"context"
	"reflect"
	"strings"

	"github.com/dop251/goja"
	"github.com/serenize/snaker"
)

var (
	ctxT    = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorT  = reflect.TypeOf((*error)(nil)).Elem()
	jsValT  = reflect.TypeOf((*goja.Value)(nil)).Elem()
	fnCallT = reflect.TypeOf((*goja.FunctionCall)(nil)).Elem()

	constructWrap = goja.MustCompile(
		"__constructor__",
		`(function(impl) { return function() { return impl.apply(this, arguments); } })`,
		true,
	)
)

// if a fieldName is the key of this map exactly than the value for the given key should be used as
// the name of the field in js
//nolint: gochecknoglobals
var fieldNameExceptions = map[string]string{
	"OCSP": "ocsp",
}

// FieldName Returns the JS name for an exported struct field. The name is snake_cased, with respect for
// certain common initialisms (URL, ID, HTTP, etc).
func FieldName(t reflect.Type, f reflect.StructField) string {
	// PkgPath is non-empty for unexported fields.
	if f.PkgPath != "" {
		return ""
	}

	// Allow a `js:"name"` tag to override the default name.
	if tag := f.Tag.Get("js"); tag != "" {
		// Matching encoding/json, `js:"-"` hides a field.
		if tag == "-" {
			return ""
		}
		return tag
	}

	if exception, ok := fieldNameExceptions[f.Name]; ok {
		return exception
	}

	// Default to lowercasing the first character of the field name.
	return snaker.CamelToSnake(f.Name)
}

// if a methodName is the key of this map exactly than the value for the given key should be used as
// the name of the method in js
//nolint: gochecknoglobals
var methodNameExceptions = map[string]string{
	"JSON": "json",
	"HTML": "html",
	"URL":  "url",
	"OCSP": "ocsp",
}

// MethodName Returns the JS name for an exported method. The first letter of the method's name is
// lowercased, otherwise it is unaltered.
func MethodName(t reflect.Type, m reflect.Method) string {
	// A field with a name beginning with an X is a constructor, and just gets the prefix stripped.
	// Note: They also get some special treatment from Bridge(), see further down.
	if m.Name[0] == 'X' {
		return m.Name[1:]
	}

	if exception, ok := methodNameExceptions[m.Name]; ok {
		return exception
	}
	// Lowercase the first character of the method name.
	return strings.ToLower(m.Name[0:1]) + m.Name[1:]
}

// FieldNameMapper for goja.Runtime.SetFieldNameMapper()
type FieldNameMapper struct{}

// FieldName is part of the goja.FieldNameMapper interface
// https://godoc.org/github.com/dop251/goja#FieldNameMapper
func (FieldNameMapper) FieldName(t reflect.Type, f reflect.StructField) string {
	return FieldName(t, f)
}

// MethodName is part of the goja.FieldNameMapper interface
// https://godoc.org/github.com/dop251/goja#FieldNameMapper
func (FieldNameMapper) MethodName(t reflect.Type, m reflect.Method) string { return MethodName(t, m) }

// Bind the provided value v to the provided runtime
func (r *Runtime) Bind(name string, v interface{}) {
	exports := r.ToBindObject(v)
	r.Runtime.Set(name, exports)
}

func (r *Runtime) ToBindObject(v interface{}) map[string]interface{} {
	exports := make(map[string]interface{})

	val := reflect.ValueOf(v)
	typ := val.Type()
	for i := 0; i < typ.NumMethod(); i++ {
		meth := typ.Method(i)
		name := MethodName(typ, meth)
		fn := val.Method(i)

		// Figure out if we want to do any wrapping of it.
		fnT := fn.Type()
		numIn := fnT.NumIn()
		numOut := fnT.NumOut()
		hasError := (numOut > 1 && fnT.Out(1) == errorT)
		wantsContext := false

		if numIn > 0 {
			in0 := fnT.In(0)
			if in0 == ctxT {
				wantsContext = true
			}
		}
		if hasError || wantsContext {
			isVariadic := fnT.IsVariadic()
			realFn := fn
			fn = reflect.ValueOf(func(call goja.FunctionCall) goja.Value {
				// Number of arguments: the higher number between the function's required arguments
				// and the number of arguments actually given.
				args := make([]reflect.Value, numIn)

				// Inject any requested parameters, and reserve them to offset user args.
				reservedArgs := 0
				if wantsContext {
					args[0] = reflect.ValueOf(r.ctx)
					reservedArgs++
				}

				// Copy over arguments.
				for i := 0; i < numIn; i++ {
					if i < reservedArgs {
						continue
					}

					T := fnT.In(i)

					// A function that takes a goja.FunctionCall takes only that arg (+ injected).
					if T == fnCallT {
						args[i] = reflect.ValueOf(call)
						break
					}

					// The last arg to a varadic function is a slice of the remainder.
					if isVariadic && i == numIn-1 {
						varArgsLen := len(call.Arguments) - (i - reservedArgs)
						if varArgsLen <= 0 {
							args[i] = reflect.Zero(T)
							break
						}
						varArgs := reflect.MakeSlice(T, varArgsLen, varArgsLen)
						emT := T.Elem()
						for j := 0; j < varArgsLen; j++ {
							arg := call.Arguments[i+j-reservedArgs]
							v := reflect.New(emT)
							if err := r.Runtime.ExportTo(arg, v.Interface()); err != nil {
								Throw(r, err)
							}
							varArgs.Index(j).Set(v.Elem())
						}
						args[i] = varArgs
						break
					}

					arg := call.Argument(i - reservedArgs)

					// Optimization: no need to allocate a pointer and export for a zero value.
					if goja.IsUndefined(arg) {
						if T == jsValT {
							args[i] = reflect.ValueOf(goja.Undefined())
							continue
						}
						args[i] = reflect.Zero(T)
						continue
					}

					// Allocate a T* and export the JS value to it.
					v := reflect.New(T)
					if err := r.Runtime.ExportTo(arg, v.Interface()); err != nil {
						Throw(r, err)
					}
					args[i] = v.Elem()
				}

				var ret []reflect.Value
				if isVariadic {
					ret = realFn.CallSlice(args)
				} else {
					ret = realFn.Call(args)
				}

				if len(ret) > 0 {
					if hasError && !ret[1].IsNil() {
						Throw(r, ret[1].Interface().(error))
					}
					return r.Runtime.ToValue(ret[0].Interface())
				}
				return goja.Undefined()
			})
		}

		// X-Prefixed methods are assumed to be constructors; use a closure to wrap them in a
		// pure-JS function to allow them to be `new`d. (This is an awful hack...)
		if meth.Name[0] == 'X' {
			wrapperV, _ := r.Runtime.RunProgram(constructWrap)
			wrapper, _ := goja.AssertFunction(wrapperV)
			v, _ := wrapper(goja.Undefined(), r.Runtime.ToValue(fn.Interface()))
			exports[name] = v
		} else {
			exports[name] = fn.Interface()
		}
	}

	// If v is a pointer, we need to indirect it to access fields.
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = val.Type()
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := FieldName(typ, field)
		if name != "" {
			exports[name] = val.Field(i).Interface()
		}
	}

	return exports
}
