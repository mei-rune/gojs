/*
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

package metrics

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/runner-mei/gojs"
	"github.com/runner-mei/gojs/lib"
	"github.com/runner-mei/gojs/stats"
)

func TestMetrics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	types := map[string]stats.MetricType{
		"Counter": stats.Counter,
		"Gauge":   stats.Gauge,
		"Trend":   stats.Trend,
		"Rate":    stats.Rate,
	}
	values := map[string]struct {
		JS    string
		Float float64
	}{
		"Float": {`2.5`, 2.5},
		"Int":   {`5`, 5.0},
		"True":  {`true`, 1.0},
		"False": {`false`, 0.0},
	}
	for fn, mtyp := range types {
		fn, mtyp := fn, mtyp
		t.Run(fn, func(t *testing.T) {
			t.Parallel()
			for isTime, valueType := range map[bool]stats.ValueType{false: stats.Default, true: stats.Time} {
				isTime, valueType := isTime, valueType
				t.Run(fmt.Sprintf("isTime=%v", isTime), func(t *testing.T) {
					t.Parallel()
					rt := gojs.New()
					rt.SetFieldNameMapper(gojs.FieldNameMapper{})
					rt.Bind("metrics", New())

					// root, _ := lib.NewGroup("", nil)
					// child, _ := root.Group("child")
					samples := make(chan stats.SampleContainer, 1000)
					state := &lib.State{
						Options: lib.Options{SystemTags: stats.NewSystemTagSet(stats.TagGroup)},
						// Group:   root,
						Samples: samples,
						Tags:    map[string]string{
							// "group": root.Path,
						},
					}

					isTimeString := ""
					if isTime {
						isTimeString = `, true`
					}
					_, err := rt.RunString(ctx,
						fmt.Sprintf(`var m = new metrics.%s("my_metric"%s)`, fn, isTimeString),
					)
					if !assert.NoError(t, err) {
						return
					}

					t.Run("ExitInit", func(t *testing.T) {
						newctx := lib.WithState(ctx, state)
						_, err := rt.RunString(newctx, fmt.Sprintf(`new metrics.%s("my_metric")`, fn))
						assert.EqualError(t, err, "GoError: metrics must be declared in the init context at apply (native)")
					})

					// groups := map[string]*lib.Group{
					// 	"Root":  root,
					// 	"Child": child,
					// }
					// for name, g := range groups {
					// 	name, g := name, g
					t.Run("Root", func(t *testing.T) {
						// state.Group = g
						//state.Tags["group"] = g.Path
						for name, val := range values {
							t.Run(name, func(t *testing.T) {
								t.Run("Simple", func(t *testing.T) {
									_, err := rt.RunString(ctx, fmt.Sprintf(`m.add(%v)`, val.JS))
									assert.NoError(t, err)
									bufSamples := stats.GetBufferedSamples(samples)
									if assert.Len(t, bufSamples, 1) {
										sample, ok := bufSamples[0].(stats.Sample)
										require.True(t, ok)

										assert.NotZero(t, sample.Time)
										assert.Equal(t, sample.Value, val.Float)
										assert.Equal(t, map[string]string{
											//"group": g.Path,
										}, sample.Tags.CloneTags())
										assert.Equal(t, "my_metric", sample.Metric.Name)
										assert.Equal(t, mtyp, sample.Metric.Type)
										assert.Equal(t, valueType, sample.Metric.Contains)
									}
								})
								t.Run("Tags", func(t *testing.T) {
									_, err := rt.RunString(ctx, fmt.Sprintf(`m.add(%v, {a:1})`, val.JS))
									assert.NoError(t, err)
									bufSamples := stats.GetBufferedSamples(samples)
									if assert.Len(t, bufSamples, 1) {
										sample, ok := bufSamples[0].(stats.Sample)
										require.True(t, ok)

										assert.NotZero(t, sample.Time)
										assert.Equal(t, sample.Value, val.Float)
										assert.Equal(t, map[string]string{
											//"group": g.Path,
											"a": "1",
										}, sample.Tags.CloneTags())
										assert.Equal(t, "my_metric", sample.Metric.Name)
										assert.Equal(t, mtyp, sample.Metric.Type)
										assert.Equal(t, valueType, sample.Metric.Contains)
									}
								})
							})
						}
					})
					//}
				})
			}
		})
	}
}

func TestMetricNames(t *testing.T) {
	t.Parallel()
	var testMap = map[string]bool{
		"simple":       true,
		"still_simple": true,
		"":             false,
		"@":            false,
		"a":            true,
		"special\n\t":  false,
		// this has both hangul and japanese numerals .
		"hello.World_in_한글一안녕一세상": true,
		// too long
		"tooolooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooog": false,
	}

	for key, value := range testMap {
		t.Run(key, func(t *testing.T) {
			assert.Equal(t, value, checkName(key), key)
		})
	}
}
