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

package netext

import (
	"cn/com/hengwei36/pkg/goutils/errors"
	"context"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/runner-mei/gojs/lib/metrics"
	"github.com/runner-mei/gojs/lib/types"
	"github.com/runner-mei/gojs/stats"
)

// IPNet is a wrapper around net.IPNet for JSON unmarshalling
type IPNet struct {
	net.IPNet
}

// UnmarshalText populates the IPNet from the given CIDR
func (ipnet *IPNet) UnmarshalText(b []byte) error {
	newIPNet, err := ParseCIDR(string(b))
	if err != nil {
		return errors.Wrap(err, "Failed to parse CIDR")
	}

	*ipnet = *newIPNet

	return nil
}

// ParseCIDR creates an IPNet out of a CIDR string
func ParseCIDR(s string) (*IPNet, error) {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	parsedIPNet := IPNet{IPNet: *ipnet}

	return &parsedIPNet, nil
}

// HostAddress stores information about IP and port
// for a host.
type HostAddress net.TCPAddr

// NewHostAddress creates a pointer to a new address with an IP object.
func NewHostAddress(ip net.IP, portString string) (*HostAddress, error) {
	var port int
	if portString != "" {
		var err error
		if port, err = strconv.Atoi(portString); err != nil {
			return nil, err
		}
	}

	return &HostAddress{
		IP:   ip,
		Port: port,
	}, nil
}

// String converts a HostAddress into a string.
func (h *HostAddress) String() string {
	return (*net.TCPAddr)(h).String()
}

// MarshalText implements the encoding.TextMarshaler interface.
// The encoding is the same as returned by String, with one exception:
// When len(ip) is zero, it returns an empty slice.
func (h *HostAddress) MarshalText() ([]byte, error) {
	if h == nil || len(h.IP) == 0 {
		return []byte(""), nil
	}

	if len(h.IP) != net.IPv4len && len(h.IP) != net.IPv6len {
		return nil, &net.AddrError{Err: "invalid IP address", Addr: h.IP.String()}
	}

	return []byte(h.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The IP address is expected in a form accepted by ParseIP.
func (h *HostAddress) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return &net.ParseError{Type: "IP address", Text: "<nil>"}
	}

	ip, port, err := splitHostPort(text)
	if err != nil {
		return err
	}

	nh, err := NewHostAddress(ip, port)
	if err != nil {
		return err
	}

	*h = *nh

	return nil
}

func splitHostPort(text []byte) (net.IP, string, error) {
	host, port, err := net.SplitHostPort(string(text))
	if err != nil {
		// This error means that there is no port.
		// Make host the full text.
		host = string(text)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, "", &net.ParseError{Type: "IP address", Text: host}
	}

	return ip, port, nil
}

// Dialer wraps net.Dialer and provides k6 specific functionality -
// tracing, blacklists and DNS cache and aliases.
type Dialer struct {
	net.Dialer

	Resolver         Resolver
	Blacklist        []*IPNet
	BlockedHostnames *types.HostnameTrie
	Hosts            map[string]*HostAddress

	BytesRead    int64
	BytesWritten int64
}

// NewDialer constructs a new Dialer with the given DNS resolver.
func NewDialer(dialer net.Dialer, resolver Resolver) *Dialer {
	return &Dialer{
		Dialer:   dialer,
		Resolver: resolver,
	}
}

// BlackListedIPError is an error that is returned when a given IP is blacklisted
type BlackListedIPError struct {
	ip  net.IP
	net *IPNet
}

func (b BlackListedIPError) Error() string {
	return fmt.Sprintf("IP (%s) is in a blacklisted range (%s)", b.ip, b.net)
}

// BlockedHostError is returned when a given hostname is blocked
type BlockedHostError struct {
	hostname string
	match    string
}

func (b BlockedHostError) Error() string {
	return fmt.Sprintf("hostname (%s) is in a blocked pattern (%s)", b.hostname, b.match)
}

// DialContext wraps the net.Dialer.DialContext and handles the k6 specifics
func (d *Dialer) DialContext(ctx context.Context, proto, addr string) (net.Conn, error) {
	dialAddr, err := d.getDialAddr(addr)
	if err != nil {
		return nil, err
	}
	conn, err := d.Dialer.DialContext(ctx, proto, dialAddr)
	if err != nil {
		return nil, err
	}
	conn = &Conn{conn, &d.BytesRead, &d.BytesWritten}
	return conn, err
}

// GetTrail creates a new NetTrail instance with the Dialer
// sent and received data metrics and the supplied times and tags.
// TODO: Refactor this according to
// https://github.com/loadimpact/k6/pull/1203#discussion_r337938370
func (d *Dialer) GetTrail(
	startTime, endTime time.Time, tags *stats.SampleTags,
) *NetTrail {
	bytesWritten := atomic.SwapInt64(&d.BytesWritten, 0)
	bytesRead := atomic.SwapInt64(&d.BytesRead, 0)
	samples := []stats.Sample{
		{
			Time:   endTime,
			Metric: metrics.DataSent,
			Value:  float64(bytesWritten),
			Tags:   tags,
		},
		{
			Time:   endTime,
			Metric: metrics.DataReceived,
			Value:  float64(bytesRead),
			Tags:   tags,
		},
	}
	// if fullIteration {
	// 	samples = append(samples, stats.Sample{
	// 		Time:   endTime,
	// 		//Metric: metrics.IterationDuration,
	// 		Value:  stats.D(endTime.Sub(startTime)),
	// 		Tags:   tags,
	// 	})
	// 	if emitIterations {
	// 		samples = append(samples, stats.Sample{
	// 			Time:   endTime,
	// 			//Metric: metrics.Iterations,
	// 			Value:  1,
	// 			Tags:   tags,
	// 		})
	// 	}
	// }

	return &NetTrail{
		BytesRead:    bytesRead,
		BytesWritten: bytesWritten,
		// FullIteration: fullIteration,
		StartTime: startTime,
		EndTime:   endTime,
		Tags:      tags,
		Samples:   samples,
	}
}

func (d *Dialer) getDialAddr(addr string) (string, error) {
	remote, err := d.findRemote(addr)
	if err != nil {
		return "", err
	}

	for _, ipnet := range d.Blacklist {
		if ipnet.Contains(remote.IP) {
			return "", BlackListedIPError{ip: remote.IP, net: ipnet}
		}
	}

	return remote.String(), nil
}

func (d *Dialer) findRemote(addr string) (*HostAddress, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(host)
	if d.BlockedHostnames != nil && ip == nil {
		if match, blocked := d.BlockedHostnames.Contains(host); blocked {
			return nil, BlockedHostError{hostname: host, match: match}
		}
	}

	remote, err := d.getConfiguredHost(addr, host, port)
	if err != nil || remote != nil {
		return remote, err
	}

	if ip != nil {
		return NewHostAddress(ip, port)
	}

	ip, err = d.Resolver.LookupIP(host)
	if err != nil {
		return nil, err
	}

	if ip == nil {
		return nil, fmt.Errorf("lookup %s: no such host", host)
	}

	return NewHostAddress(ip, port)
}

func (d *Dialer) getConfiguredHost(addr, host, port string) (*HostAddress, error) {
	if remote, ok := d.Hosts[addr]; ok {
		return remote, nil
	}

	if remote, ok := d.Hosts[host]; ok {
		if remote.Port != 0 || port == "" {
			return remote, nil
		}

		newPort, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}

		newRemote := *remote
		newRemote.Port = newPort

		return &newRemote, nil
	}

	return nil, nil
}

// NetTrail contains information about the exchanged data size and length of a
// series of connections from a particular netext.Dialer
type NetTrail struct {
	BytesRead    int64
	BytesWritten int64
	// FullIteration bool
	StartTime time.Time
	EndTime   time.Time
	Tags      *stats.SampleTags
	Samples   []stats.Sample
}

// Ensure that interfaces are implemented correctly
var _ stats.ConnectedSampleContainer = &NetTrail{}

// GetSamples implements the stats.SampleContainer interface.
func (ntr *NetTrail) GetSamples() []stats.Sample {
	return ntr.Samples
}

// GetTags implements the stats.ConnectedSampleContainer interface.
func (ntr *NetTrail) GetTags() *stats.SampleTags {
	return ntr.Tags
}

// GetTime implements the stats.ConnectedSampleContainer interface.
func (ntr *NetTrail) GetTime() time.Time {
	return ntr.EndTime
}

// Conn wraps net.Conn and keeps track of sent and received data size
type Conn struct {
	net.Conn

	BytesRead, BytesWritten *int64
}

func (c *Conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		atomic.AddInt64(c.BytesRead, int64(n))
	}
	return n, err
}

func (c *Conn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		atomic.AddInt64(c.BytesWritten, int64(n))
	}
	return n, err
}
