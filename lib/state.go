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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/oxtoacart/bpool"
	"golang.org/x/net/http2"
	"golang.org/x/time/rate"

	"github.com/runner-mei/gojs/lib/netext"
	"github.com/runner-mei/gojs/lib/types"
	"github.com/runner-mei/gojs/stats"
	"github.com/runner-mei/log"
)

// DialContexter is an interface that can dial with a context
type DialContexter interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// State provides the volatile state for a VU.
type State struct {
	// Global options.
	Options Options

	// Logger. Avoid using the global logger.
	Logger log.Logger

	// Networking equipment.
	Transport http.RoundTripper
	Dialer    DialContexter
	CookieJar *cookiejar.Jar
	TLSConfig *tls.Config

	// Rate limits.
	RPSLimit *rate.Limiter

	// Sample channel, possibly buffered
	Samples chan<- stats.SampleContainer

	// Buffer pool; use instead of allocating fresh buffers when possible.
	// TODO: maybe use https://golang.org/pkg/sync/#Pool ?
	BPool *bpool.BufferPool

	Tags map[string]string
}

// CloneTags makes a copy of the tags map and returns it.
func (s *State) CloneTags() map[string]string {
	tags := make(map[string]string, len(s.Tags))
	for k, v := range s.Tags {
		tags[k] = v
	}
	return tags
}

func parseTTL(ttlS string) (time.Duration, error) {
	ttl := time.Duration(0)
	switch ttlS {
	case "inf":
		// cache "infinitely"
		ttl = time.Hour * 24 * 365
	case "0":
		// disable cache
	case "":
		ttlS = types.DefaultDNSConfig().TTL.String
		fallthrough
	default:
		var err error
		ttl, err = types.ParseExtendedDuration(ttlS)
		if ttl < 0 || err != nil {
			return ttl, fmt.Errorf("invalid DNS TTL: %s", ttlS)
		}
	}
	return ttl, nil
}

func NewState(logger log.Logger, opts Options) (*State, error) {
	var rpsLimit *rate.Limiter
	if rps := opts.RPS; rps.Valid {
		rpsLimit = rate.NewLimiter(rate.Limit(rps.Int64), 1)
	}

	if opts.SystemTags == nil {
		opts.SystemTags = &stats.DefaultSystemTagSet
	}
	// if opts.SummaryTrendStats == nil {
	// 	opts.SummaryTrendStats = lib.DefaultSummaryTrendStats
	// }

	defDNS := types.DefaultDNSConfig()
	if !opts.DNS.TTL.Valid {
		opts.DNS.TTL = defDNS.TTL
	}
	if !opts.DNS.Select.Valid {
		opts.DNS.Select = defDNS.Select
	}
	if !opts.DNS.Policy.Valid {
		opts.DNS.Policy = defDNS.Policy
	}

	ttl, err := parseTTL(opts.DNS.TTL.String)
	if err != nil {
		return nil, err
	}
	resolver := netext.NewResolver(net.LookupIP,
		ttl,
		opts.DNS.Select.DNSSelect,
		opts.DNS.Policy.DNSPolicy)

	var cipherSuites []uint16
	if opts.TLSCipherSuites != nil {
		cipherSuites = *opts.TLSCipherSuites
	}

	var tlsVersions netext.TLSVersions
	if opts.TLSVersion != nil {
		tlsVersions = *opts.TLSVersion
	}

	tlsAuth := opts.TLSAuth
	certs := make([]tls.Certificate, len(tlsAuth))
	nameToCert := make(map[string]*tls.Certificate)
	for i, auth := range tlsAuth {
		for _, name := range auth.Domains {
			cert, err := auth.Certificate()
			if err != nil {
				return nil, err
			}
			certs[i] = *cert
			nameToCert[name] = &certs[i]
		}
	}

	dialer := &netext.Dialer{
		Dialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		},
		Resolver:         resolver,
		Blacklist:        opts.BlacklistIPs,
		BlockedHostnames: opts.BlockedHostnames.Trie,
		Hosts:            opts.Hosts,
	}
	if opts.LocalIPs.Valid {
		var ipIndex uint64 = 0
		dialer.Dialer.LocalAddr = &net.TCPAddr{IP: opts.LocalIPs.Pool.GetIP(ipIndex)}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.InsecureSkipTLSVerify.Bool,
		CipherSuites:       cipherSuites,
		MinVersion:         uint16(tlsVersions.Min),
		MaxVersion:         uint16(tlsVersions.Max),
		Certificates:       certs,
		NameToCertificate:  nameToCert,
		Renegotiation:      tls.RenegotiateFreelyAsClient,
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSClientConfig:     tlsConfig,
		DialContext:         dialer.DialContext,
		DisableCompression:  true,
		DisableKeepAlives:   opts.NoConnectionReuse.Bool,
		MaxIdleConns:        int(opts.Batch.Int64),
		MaxIdleConnsPerHost: int(opts.BatchPerHost.Int64),
	}
	_ = http2.ConfigureTransport(transport)

	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return &State{
		Logger:    logger,
		Options:   opts,
		Transport: transport,
		Dialer:    dialer,
		TLSConfig: tlsConfig,
		CookieJar: cookieJar,
		RPSLimit:  rpsLimit,
		BPool:     bpool.NewBufferPool(100),
		Tags:      opts.RunTags.CloneTags(),
	}, nil
}
