/*
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
	"crypto/tls"
	"encoding/json"

	"github.com/pkg/errors"
)

// Describes a TLS version. Serialised to/from JSON as a string, eg. "tls1.2".
type TLSVersion int

func (v TLSVersion) MarshalJSON() ([]byte, error) {
	return []byte(`"` + SupportedTLSVersionsToString[v] + `"`), nil
}

func (v *TLSVersion) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		*v = 0
		return nil
	}
	ver, ok := SupportedTLSVersions[str]
	if !ok {
		return errors.Errorf("unknown TLS version: %s", str)
	}
	*v = ver
	return nil
}

// Fields for TLSVersions. Unmarshalling hack.
type TLSVersionsFields struct {
	Min TLSVersion `json:"min" ignored:"true"` // Minimum allowed version, 0 = any.
	Max TLSVersion `json:"max" ignored:"true"` // Maximum allowed version, 0 = any.
}

// Describes a set (min/max) of TLS versions.
type TLSVersions TLSVersionsFields

func (v *TLSVersions) UnmarshalJSON(data []byte) error {
	var fields TLSVersionsFields
	if err := json.Unmarshal(data, &fields); err != nil {
		var ver TLSVersion
		if err2 := json.Unmarshal(data, &ver); err2 != nil {
			return err
		}
		fields.Min = ver
		fields.Max = ver
	}
	*v = TLSVersions(fields)
	return nil
}

func (v *TLSVersions) IsTLS13() bool {
	return v.Min == TLSVersion13 || v.Max == TLSVersion13
}

// A list of TLS cipher suites.
// Marshals and unmarshals from a list of names, eg. "TLS_ECDHE_RSA_WITH_RC4_128_SHA".
type TLSCipherSuites []uint16

// MarshalJSON will return the JSON representation according to supported TLS cipher suites
func (s *TLSCipherSuites) MarshalJSON() ([]byte, error) {
	var suiteNames []string
	for _, id := range *s {
		if suiteName, ok := SupportedTLSCipherSuitesToString[id]; ok {
			suiteNames = append(suiteNames, suiteName)
		} else {
			return nil, errors.Errorf("Unknown cipher suite id: %d", id)
		}
	}

	return json.Marshal(suiteNames)
}

func (s *TLSCipherSuites) UnmarshalJSON(data []byte) error {
	var suiteNames []string
	if err := json.Unmarshal(data, &suiteNames); err != nil {
		return err
	}

	var suiteIDs []uint16
	for _, name := range suiteNames {
		if suiteID, ok := SupportedTLSCipherSuites[name]; ok {
			suiteIDs = append(suiteIDs, suiteID)
		} else {
			return errors.New("Unknown cipher suite: " + name)
		}
	}

	*s = suiteIDs

	return nil
}

// Fields for TLSAuth. Unmarshalling hack.
type TLSAuthFields struct {
	// Certificate and key as a PEM-encoded string, including "-----BEGIN CERTIFICATE-----".
	Cert string `json:"cert"`
	Key  string `json:"key"`

	// Domains to present the certificate to. May contain wildcards, eg. "*.example.com".
	Domains []string `json:"domains"`
}

// Defines a TLS client certificate to present to certain hosts.
type TLSAuth struct {
	TLSAuthFields
	certificate *tls.Certificate
}

func (c *TLSAuth) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &c.TLSAuthFields); err != nil {
		return err
	}
	if _, err := c.Certificate(); err != nil {
		return err
	}
	return nil
}

func (c *TLSAuth) Certificate() (*tls.Certificate, error) {
	if c.certificate == nil {
		cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
		if err != nil {
			return nil, err
		}
		c.certificate = &cert
	}
	return c.certificate, nil
}

// From https://golang.org/pkg/crypto/tls/#pkg-constants

// SupportedTLSVersions is string-to-constant map of available TLS versions.
//nolint: gochecknoglobals
var SupportedTLSVersions = map[string]TLSVersion{
	"ssl3.0": tls.VersionSSL30,
	"tls1.0": tls.VersionTLS10,
	"tls1.1": tls.VersionTLS11,
	"tls1.2": tls.VersionTLS12,
	"tls1.3": TLSVersion13,
}

// SupportedTLSVersionsToString is constant-to-string map of available TLS versions.
//nolint: gochecknoglobals
var SupportedTLSVersionsToString = map[TLSVersion]string{
	tls.VersionSSL30: "ssl3.0",
	tls.VersionTLS10: "tls1.0",
	tls.VersionTLS11: "tls1.1",
	tls.VersionTLS12: "tls1.2",
	TLSVersion13:     "tls1.3",
}

// SupportedTLSCipherSuites is string-to-constant map of available TLS cipher suites.
//nolint: gochecknoglobals
var SupportedTLSCipherSuites = map[string]uint16{
	// TLS 1.0 - 1.2 cipher suites.
	"TLS_RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,

	// TLS 1.3 cipher suites.
	"TLS_AES_128_GCM_SHA256":       TLS13_CIPHER_SUITE_TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384":       TLS13_CIPHER_SUITE_TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256": TLS13_CIPHER_SUITE_TLS_CHACHA20_POLY1305_SHA256,
}

// SupportedTLSCipherSuitesToString is constant-to-string map of available TLS cipher suites.
//nolint: gochecknoglobals
var SupportedTLSCipherSuitesToString = map[uint16]string{
	tls.TLS_RSA_WITH_RC4_128_SHA:                    "TLS_RSA_WITH_RC4_128_SHA",
	tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:               "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA:                "TLS_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_RSA_WITH_AES_256_CBC_SHA:                "TLS_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_RSA_WITH_AES_128_CBC_SHA256:             "TLS_RSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_RSA_WITH_AES_128_GCM_SHA256:             "TLS_RSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384:             "TLS_RSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:            "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:        "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:        "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:              "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:         "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:          "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:          "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:       "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:     "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:       "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:     "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256:     "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:        "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:      "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:       "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	TLS13_CIPHER_SUITE_TLS_AES_128_GCM_SHA256:       "TLS_AES_128_GCM_SHA256",
	TLS13_CIPHER_SUITE_TLS_AES_256_GCM_SHA384:       "TLS_AES_256_GCM_SHA384",
	TLS13_CIPHER_SUITE_TLS_CHACHA20_POLY1305_SHA256: "TLS_CHACHA20_POLY1305_SHA256",
}
