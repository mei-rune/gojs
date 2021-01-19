/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2020 Load Impact
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

package data

import (
	"context"
	"testing"

	"github.com/dop251/goja"
	"github.com/runner-mei/gojs"
	"github.com/stretchr/testify/require"
)

const makeArrayScript = `
var array = new data.SharedArray("shared",function() {
    var n = 50;
    var arr = new Array(n);
    for (var i = 0 ; i <n; i++) {
        arr[i] = {value: "something" +i};
    }
	return arr;
});
`

func newConfiguredRuntime(ctx context.Context, initEnv *gojs.InitEnvironment) (*gojs.Runtime, error) {
	rt := gojs.New()
	rt.SetFieldNameMapper(gojs.FieldNameMapper{})

	rt.Bind("data", new(data))
	_, err := rt.RunString(ctx, "var SharedArray = data.SharedArray;")

	return rt, err
}

func TestSharedArrayConstructorExceptions(t *testing.T) {
	t.Parallel()
	initEnv := &gojs.InitEnvironment{
		SharedObjects: gojs.NewSharedObjects(),
	}
	ctx := gojs.WithInitEnv(context.Background(), initEnv)
	rt, err := newConfiguredRuntime(ctx, initEnv)
	require.NoError(t, err)
	cases := map[string]struct {
		code, err string
	}{
		"returning string": {
			code: `new SharedArray("wat", function() {return "whatever"});`,
			err:  "only arrays can be made into SharedArray",
		},
		"empty name": {
			code: `new SharedArray("", function() {return []});`,
			err:  "empty name provided to SharedArray's constructor",
		},
		"function in the data": {
			code: `
			var s = new SharedArray("wat2", function() {return [{s: function() {}}]});
			if (s[0].s !== undefined) {
				throw "s[0].s should be undefined"
			}
		`,
			err: "",
		},
	}

	for name, testCase := range cases {
		name, testCase := name, testCase
		t.Run(name, func(t *testing.T) {
			_, err := rt.RunString(ctx, testCase.code)
			if testCase.err == "" {
				require.NoError(t, err)
				return // the t.Run
			}

			require.Error(t, err)
			exc := err.(*goja.Exception)
			require.Contains(t, exc.Error(), testCase.err)
		})
	}
}

func TestSharedArrayAnotherRuntimeExceptions(t *testing.T) {
	t.Parallel()

	initEnv := &gojs.InitEnvironment{
		SharedObjects: gojs.NewSharedObjects(),
	}
	ctx := gojs.WithInitEnv(context.Background(), initEnv)
	rt, err := newConfiguredRuntime(ctx, initEnv)
	require.NoError(t, err)
	_, err = rt.RunString(ctx, makeArrayScript)
	require.NoError(t, err)

	// create another Runtime with new ctx but keep the initEnv
	rt, err = newConfiguredRuntime(ctx, initEnv)
	require.NoError(t, err)
	_, err = rt.RunString(ctx, makeArrayScript)
	require.NoError(t, err)

	// use strict is required as otherwise just nothing happens
	cases := map[string]struct {
		code, err string
	}{
		"setting in for-of": {
			code: `'use strict'; for (var v of array) { v.data = "bad"; }`,
			err:  "Cannot add property data, object is not extensible",
		},
		"setting from index": {
			code: `'use strict'; array[2].data2 = "bad2"`,
			err:  "Cannot add property data2, object is not extensible",
		},
		"setting property on the proxy": {
			code: `'use strict'; array.something = "something"`,
			err:  "Host object field something cannot be made configurable",
		},
		"setting index on the proxy": {
			code: `'use strict'; array[2] = "something"`,
			err:  "Host object field 2 cannot be made configurable",
		},
	}

	for name, testCase := range cases {
		name, testCase := name, testCase
		t.Run(name, func(t *testing.T) {
			_, err := rt.RunString(ctx, testCase.code)
			if testCase.err == "" {
				require.NoError(t, err)
				return // the t.Run
			}

			require.Error(t, err)
			exc := err.(*goja.Exception)
			require.Contains(t, exc.Error(), testCase.err)
		})
	}
}

func TestSharedArrayAnotherRuntimeWorking(t *testing.T) {
	t.Parallel()

	initEnv := &gojs.InitEnvironment{
		SharedObjects: gojs.NewSharedObjects(),
	}
	ctx := gojs.WithInitEnv(context.Background(), initEnv)
	rt, err := newConfiguredRuntime(ctx, initEnv)
	require.NoError(t, err)
	_, err = rt.RunString(ctx, makeArrayScript)
	require.NoError(t, err)

	// create another Runtime with new ctx but keep the initEnv
	rt, err = newConfiguredRuntime(ctx, initEnv)
	require.NoError(t, err)
	_, err = rt.RunString(ctx, makeArrayScript)
	require.NoError(t, err)

	_, err = rt.RunString(ctx, `
	if (array[2].value !== "something2") {
		throw new Error("bad array[2]="+array[2].value);
	}
	if (array.length != 50) {
		throw new Error("bad length " +array.length);
	}

	var i = 0;
	for (var v of array) {
		if (v.value !== "something"+i) {
			throw new Error("bad v.value="+v.value+" for i="+i);
		}
		i++;
	}
	`)
	require.NoError(t, err)
}
