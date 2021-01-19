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

package lib

import (
	"encoding/json"
	"fmt"
	"reflect"

	null "gopkg.in/guregu/null.v3"

	"github.com/runner-mei/gojs/lib/netext"
	"github.com/runner-mei/gojs/lib/types"
	"github.com/runner-mei/gojs/stats"
)

// DefaultScenarioName is used as the default key/ID of the scenario config entries
// that were created due to the use of the shortcut execution control options (i.e. duration+vus,
// iterations+vus, or stages)
const DefaultScenarioName = "default"

// DefaultSummaryTrendStats are the default trend columns shown in the test summary output
// nolint: gochecknoglobals
var DefaultSummaryTrendStats = []string{"avg", "min", "med", "max", "p(90)", "p(95)"}

type Options struct {
	// Limit HTTP requests per second.
	RPS null.Int `json:"rps" envconfig:"K6_RPS"`

	// DNS handling configuration.
	DNS types.DNSConfig `json:"dns" envconfig:"K6_DNS"`

	// How many HTTP redirects do we follow?
	MaxRedirects null.Int `json:"maxRedirects" envconfig:"K6_MAX_REDIRECTS"`

	// Default User Agent string for HTTP requests.
	UserAgent null.String `json:"userAgent" envconfig:"K6_USER_AGENT"`

	// How many batch requests are allowed in parallel, in total and per host?
	Batch        null.Int `json:"batch" envconfig:"K6_BATCH"`
	BatchPerHost null.Int `json:"batchPerHost" envconfig:"K6_BATCH_PER_HOST"`

	// Should all HTTP requests and responses be logged (excluding body)?
	HTTPDebug null.String `json:"httpDebug" envconfig:"K6_HTTP_DEBUG"`

	// Accept invalid or untrusted TLS certificates.
	InsecureSkipTLSVerify null.Bool `json:"insecureSkipTLSVerify" envconfig:"K6_INSECURE_SKIP_TLS_VERIFY"`

	// Specify TLS versions and cipher suites, and present client certificates.
	TLSCipherSuites *netext.TLSCipherSuites `json:"tlsCipherSuites" envconfig:"K6_TLS_CIPHER_SUITES"`
	TLSVersion      *netext.TLSVersions     `json:"tlsVersion" ignored:"true"`
	TLSAuth         []*netext.TLSAuth       `json:"tlsAuth" envconfig:"K6_TLSAUTH"`

	// Throw warnings (eg. failed HTTP requests) as errors instead of simply logging them.
	Throw null.Bool `json:"throw" envconfig:"K6_THROW"`

	// Define thresholds; these take the form of 'metric=["snippet1", "snippet2"]'.
	// To create a threshold on a derived metric based on tag queries ("submetrics"), create a
	// metric on a nonexistent metric named 'real_metric{tagA:valueA,tagB:valueB}'.
	Thresholds map[string]stats.Thresholds `json:"thresholds" envconfig:"K6_THRESHOLDS"`

	// Blacklist IP ranges that tests may not contact. Mainly useful in hosted setups.
	BlacklistIPs []*netext.IPNet `json:"blacklistIPs" envconfig:"K6_BLACKLIST_IPS"`

	// Block hostname patterns that tests may not contact.
	BlockedHostnames types.NullHostnameTrie `json:"blockHostnames" envconfig:"K6_BLOCK_HOSTNAMES"`

	// Hosts overrides dns entries for given hosts
	Hosts map[string]*netext.HostAddress `json:"hosts" envconfig:"K6_HOSTS"`

	// Disable keep-alive connections
	NoConnectionReuse null.Bool `json:"noConnectionReuse" envconfig:"K6_NO_CONNECTION_REUSE"`

	// These values are for third party collectors' benefit.
	// Can't be set through env vars.
	External map[string]json.RawMessage `json:"ext" ignored:"true"`

	// Summary trend stats for trend metrics (response times) in CLI output
	SummaryTrendStats []string `json:"summaryTrendStats" envconfig:"K6_SUMMARY_TREND_STATS"`

	// Summary time unit for summary metrics (response times) in CLI output
	SummaryTimeUnit null.String `json:"summaryTimeUnit" envconfig:"K6_SUMMARY_TIME_UNIT"`

	// Which system tags to include with metrics ("method", "vu" etc.)
	// Use pointer for identifying whether user provide any tag or not.
	SystemTags *stats.SystemTagSet `json:"systemTags" envconfig:"K6_SYSTEM_TAGS"`

	// Tags to be applied to all samples for this running
	RunTags *stats.SampleTags `json:"tags" envconfig:"K6_TAGS"`

	// Buffer size of the channel for metric samples; 0 means unbuffered
	MetricSamplesBufferSize null.Int `json:"metricSamplesBufferSize" envconfig:"K6_METRIC_SAMPLES_BUFFER_SIZE"`

	// Do not reset cookies after a VU iteration
	NoCookiesReset null.Bool `json:"noCookiesReset" envconfig:"K6_NO_COOKIES_RESET"`

	// Discard Http Responses Body
	DiscardResponseBodies null.Bool `json:"discardResponseBodies" envconfig:"K6_DISCARD_RESPONSE_BODIES"`

	// Redirect console logging to a file
	ConsoleOutput null.String `json:"-" envconfig:"K6_CONSOLE_OUTPUT"`

	// Specify client IP ranges and/or CIDR from which VUs will make requests
	LocalIPs types.NullIPPool `json:"-" envconfig:"K6_LOCAL_IPS"`
}

// Returns the result of overwriting any fields with any that are set on the argument.
//
// Example:
//   a := Options{VUs: null.IntFrom(10), VUsMax: null.IntFrom(10)}
//   b := Options{VUs: null.IntFrom(5)}
//   a.Apply(b) // Options{VUs: null.IntFrom(5), VUsMax: null.IntFrom(10)}
func (o Options) Apply(opts Options) Options {
	if opts.RPS.Valid {
		o.RPS = opts.RPS
	}
	if opts.MaxRedirects.Valid {
		o.MaxRedirects = opts.MaxRedirects
	}
	if opts.UserAgent.Valid {
		o.UserAgent = opts.UserAgent
	}
	if opts.Batch.Valid {
		o.Batch = opts.Batch
	}
	if opts.BatchPerHost.Valid {
		o.BatchPerHost = opts.BatchPerHost
	}
	if opts.HTTPDebug.Valid {
		o.HTTPDebug = opts.HTTPDebug
	}
	if opts.InsecureSkipTLSVerify.Valid {
		o.InsecureSkipTLSVerify = opts.InsecureSkipTLSVerify
	}
	if opts.TLSCipherSuites != nil {
		o.TLSCipherSuites = opts.TLSCipherSuites
	}
	if opts.TLSVersion != nil {
		o.TLSVersion = opts.TLSVersion
		if o.TLSVersion.IsTLS13() {
			enableTLS13()
		}
	}
	if opts.TLSAuth != nil {
		o.TLSAuth = opts.TLSAuth
	}
	if opts.Throw.Valid {
		o.Throw = opts.Throw
	}
	if opts.Thresholds != nil {
		o.Thresholds = opts.Thresholds
	}
	if opts.BlacklistIPs != nil {
		o.BlacklistIPs = opts.BlacklistIPs
	}
	if opts.BlockedHostnames.Valid {
		o.BlockedHostnames = opts.BlockedHostnames
	}
	if opts.Hosts != nil {
		o.Hosts = opts.Hosts
	}
	if opts.NoConnectionReuse.Valid {
		o.NoConnectionReuse = opts.NoConnectionReuse
	}
	// if opts.NoVUConnectionReuse.Valid {
	// 	o.NoVUConnectionReuse = opts.NoVUConnectionReuse
	// }
	// if opts.MinIterationDuration.Valid {
	// 	o.MinIterationDuration = opts.MinIterationDuration
	// }
	if opts.NoCookiesReset.Valid {
		o.NoCookiesReset = opts.NoCookiesReset
	}
	if opts.External != nil {
		o.External = opts.External
	}
	if opts.SummaryTrendStats != nil {
		o.SummaryTrendStats = opts.SummaryTrendStats
	}
	if opts.SummaryTimeUnit.Valid {
		o.SummaryTimeUnit = opts.SummaryTimeUnit
	}
	if opts.SystemTags != nil {
		o.SystemTags = opts.SystemTags
	}
	if !opts.RunTags.IsEmpty() {
		o.RunTags = opts.RunTags
	}
	if opts.MetricSamplesBufferSize.Valid {
		o.MetricSamplesBufferSize = opts.MetricSamplesBufferSize
	}
	if opts.DiscardResponseBodies.Valid {
		o.DiscardResponseBodies = opts.DiscardResponseBodies
	}
	if opts.ConsoleOutput.Valid {
		o.ConsoleOutput = opts.ConsoleOutput
	}
	if opts.LocalIPs.Valid {
		o.LocalIPs = opts.LocalIPs
	}
	if opts.DNS.TTL.Valid {
		o.DNS.TTL = opts.DNS.TTL
	}
	if opts.DNS.Select.Valid {
		o.DNS.Select = opts.DNS.Select
	}
	if opts.DNS.Policy.Valid {
		o.DNS.Policy = opts.DNS.Policy
	}

	return o
}

// // Validate checks if all of the specified options make sense
// func (o Options) Validate() []error {
// 	// TODO: validate all of the other options... that we should have already been validating...
// 	// TODO: maybe integrate an external validation lib: https://github.com/avelino/awesome-go#validation
// 	var errors []error
// 	if o.ExecutionSegmentSequence != nil {
// 		var segmentFound bool
// 		for _, segment := range *o.ExecutionSegmentSequence {
// 			if o.ExecutionSegment.Equal(segment) {
// 				segmentFound = true
// 				break
// 			}
// 		}
// 		if !segmentFound {
// 			errors = append(errors,
// 				fmt.Errorf("provided segment %s can't be found in sequence %s",
// 					o.ExecutionSegment, o.ExecutionSegmentSequence))
// 		}
// 	}
// 	return append(errors, o.Scenarios.Validate()...)
// }

// ForEachSpecified enumerates all struct fields and calls the supplied function with each
// element that is valid. It panics for any unfamiliar or unexpected fields, so make sure
// new fields in Options are accounted for.
func (o Options) ForEachSpecified(structTag string, callback func(key string, value interface{})) {
	structType := reflect.TypeOf(o)
	structVal := reflect.ValueOf(o)
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		fieldVal := structVal.Field(i)
		value := fieldVal.Interface()

		shouldCall := false
		switch fieldType.Type.Kind() {
		case reflect.Struct:
			// Unpack any guregu/null values
			shouldCall = fieldVal.FieldByName("Valid").Bool()
			valOrZero := fieldVal.MethodByName("ValueOrZero")
			if shouldCall && valOrZero.IsValid() {
				value = valOrZero.Call([]reflect.Value{})[0].Interface()
				if v, ok := value.(types.Duration); ok {
					value = v.String()
				}
			}
		case reflect.Slice:
			shouldCall = fieldVal.Len() > 0
		case reflect.Map:
			shouldCall = fieldVal.Len() > 0
		case reflect.Ptr:
			shouldCall = !fieldVal.IsNil()
		default:
			panic(fmt.Sprintf("Unknown Options field %#v", fieldType))
		}

		if shouldCall {
			key, ok := fieldType.Tag.Lookup(structTag)
			if !ok {
				key = fieldType.Name
			}

			callback(key, value)
		}
	}
}
