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
	"fmt"
	"reflect"
	"testing"

	"github.com/runner-mei/log"
	"github.com/runner-mei/log/logtest"
)

func TestConsoleContext(t *testing.T) {
	rt := New()
	rt.SetFieldNameMapper(FieldNameMapper{})

	logger, logEntries := logtest.NewObservedLogger()
	rt.Bind("console", &console{logger})

	ctx := context.Background()
	_, err := rt.RunString(ctx, `console.log("a")`)
	if err != nil {
		t.Error(err)
	}
	if exists, entry := logtest.LastEntry(logEntries); exists {
		if "a" != entry.Message {
			t.Error("excepted a got", entry.Message)
		}
	} else {
		t.Error("nothing logged")
	}
}

func TestConsole(t *testing.T) {
	levels := map[string]log.Level{
		"log":   log.InfoLevel,
		"debug": log.DebugLevel,
		"info":  log.InfoLevel,
		"warn":  log.WarnLevel,
		"error": log.ErrorLevel,
	}
	argsets := map[string]struct {
		Message string
		Context []log.Field
	}{
		`"string"`:         {Message: "string"},
		`"string","a","b"`: {Message: "string", Context: []log.Field{log.String("0", "a"), log.String("1", "b")}},
		`"string",1,2`:     {Message: "string", Context: []log.Field{log.String("0", "1"), log.String("1", "2")}},
		`{}`:               {Message: "[object Object]"},
	}
	for name, level := range levels {
		name, level := name, level
		t.Run(name, func(t *testing.T) {
			for args, result := range argsets {
				args, result := args, result
				t.Run(args, func(t *testing.T) {
					rt := New()
					rt.SetFieldNameMapper(FieldNameMapper{})

					logger, logEntries := logtest.NewObservedLogger()
					rt.Bind("console", &console{logger})

					ctx := context.Background()

					_, err := rt.RunString(ctx, fmt.Sprintf(
						`console.%s(%s);`,
						name, args,
					))
					if err != nil {
						t.Error(err)
					}

					exists, entry := logtest.LastEntry(logEntries)
					if !exists {
						t.Error("nothing logged")
						return
					}

					if level != entry.Level {
						t.Error("excepted", level, "got", entry.Level)
					}

					if result.Message != entry.Message {
						t.Error("excepted", result.Message, "got", entry.Message)
					}

					if len(result.Context) == 0 {
						if len(entry.Context) != 0 {
							t.Error("excepted", result.Context)
							t.Error("got     ", entry.Context)
						}
					} else if excepted, actual := logtest.FieldsMap(result.Context), logtest.FieldsMap(entry.Context); !reflect.DeepEqual(excepted, actual) {
						t.Error("excepted", excepted)
						t.Error("got     ", actual)
					}
				})
			}
		})
	}
}
