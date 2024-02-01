// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2022 The Happy Authors

// Package address provides functions for working with "happy" addresses,
// which are URL-like strings that define the location of a resource in the "happy" system.
// It includes functions for parsing and resolving addresses, as well as methods for converting
// addresses to strings and checking the format of addresses. The package also provides a
// convenient way to get the current application's address in the "happy" system.
package address

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"
)

const (
	// MustCompile against following expression.
	Regexp = "^[a-z][a-z0-9-./]*[a-z0-9]$"
	dot    = '.'
)

var (
	ErrAddr = errors.New("address")

	alnum = &unicode.RangeTable{ //nolint:gochecknoglobals
		R16: []unicode.Range16{
			{'0', '9', 1},
			{'A', 'Z', 1},
			{'a', 'z', 1},
		},
	}
)

// Address represents a happy address in the form of happy://host/instance/path.
// The happy address is a scheme-based URL that defines the location of a module
// or service within a service mesh.
type Address struct {
	// url is the underlying URL representation of the happy address.
	url *url.URL

	// Host is the hostname component of the happy address.
	host string

	// Instance is the instance component of the happy address, which defines the service mesh the address belongs to.
	instance string

	module     string
	reversedns string
}

// String reassembles the Address into a valid URL string.
// The general form of the result is one of:
//
//	happy://userinfo@host/path?query#fragment
//
// Any non-ASCII characters in host are escaped.
// To obtain the path, String uses net.URL.EscapedPath().
//
// In the second form, the following rules apply:
//   - if u.User is nil, userinfo@ is omitted.
//   - if u.Host is non-empty and u.Path begins with a /,
//     the form host/path does not add its own /.
//   - if u.RawQuery is empty, ?query is omitted.
//   - if u.Fragment is empty, #fragment is omitted.
func (a *Address) String() string {
	return a.url.String()
}

func (a *Address) Host() string {
	return a.host
}

func (a *Address) Instance() string {
	return a.instance
}

func (a *Address) ReverseDNS() string {
	return a.reversedns
}

func (a *Address) Module() string {
	return a.module
}

// Parse takes a string representation of an address and returns a pointer to a new Address struct.
// If the input string is not a valid representation of an address, an error is returned.
func (a *Address) Parse(ref string) (*Address, error) {
	refurl, err := a.url.Parse(ref)
	if err != nil {
		return nil, err
	}
	return &Address{
		url:      refurl,
		instance: a.instance,
		host:     refurl.Host,
	}, nil
}

// ResolveService returns the resolved service Adddress
// for the given network and instance address.
// If the address cannot be resolved, it returns an error.
func (a *Address) ResolveService(svc string) (*Address, error) {
	if !strings.HasPrefix(svc, "happy://") {
		svc = path.Join(a.instance, "service", svc)
	}
	svcaddr, err := a.Parse(svc)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(svcaddr.url.Path, "/"+svcaddr.instance+"/service") {
		return nil, fmt.Errorf("%w: not a service %s", ErrAddr, svcaddr.String())
	}
	return svcaddr, nil
}

// FromModule returns a new Address created from the given go module path.
// If the module is empty, the resulting address will be an empty address.
// If the module is not an empty string, it must be a valid module path that
// conforms to Go's package name syntax and will be used to create a new address.
func FromModule(host, modulepath string) (*Address, error) {
	// fully qualified ?
	sl := strings.Split(modulepath, "/")
	if len(sl) == 1 {
		return Parse("happy://" + host + "/" + ensure(modulepath))
	}

	var rev []string
	var rmdomain bool
	if strings.Contains(sl[0], ".") {
		rmdomain = true
		domainparts := sort.StringSlice(strings.Split(sl[0], "."))
		sort.Sort(domainparts)
		rev = append(rev, ensure(strings.Join(domainparts, ".")))
	}
	p := len(sl)
	for i := 0; i < p; i++ {
		if rmdomain && i == 0 {
			continue
		}
		rev = append(rev, (sl[i]))
	}
	reversedns := strings.Join(rev, ".")
	addr, err := Parse("happy://" + host + "/" + reversedns)
	if err != nil {
		return nil, err
	}
	addr.module = modulepath
	addr.reversedns = reversedns
	return addr, nil
}

// Current returns format of current application happy proto address.
func Current() (*Address, error) {
	var name string
	if info, available := debug.ReadBuildInfo(); available {

		if info.Path == "command-line-arguments" {
			return nil, errors.Join(
				fmt.Errorf("%w: unable to read module info", ErrAddr),
				fmt.Errorf("%w: possible reason go run main.go vs go run ./", ErrAddr),
			)
		} else {
			name = info.Path
		}

	} else {
		pc, _, _, _ := runtime.Caller(0)
		ps := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl := len(ps)
		if ps[pl-2][0] == '(' {
			name = strings.Join(ps[0:pl-2], ".")
		} else {
			name = strings.Join(ps[0:pl-1], ".")
		}
	}
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return FromModule(host, name)
}

// Current returns format of current application happy proto address.
func CurrentForDepth(d int) (*Address, error) {
	var name string
	if info, available := debug.ReadBuildInfo(); available {

		if info.Path == "command-line-arguments" {
			return nil, errors.Join(
				fmt.Errorf("%w: unable to read module info", ErrAddr),
				fmt.Errorf("%w: possible reason go run main.go vs go run ./", ErrAddr),
			)
		} else if info.Path != "" {
			name = info.Path
		}
	}
	if name == "" {
		pc, _, _, _ := runtime.Caller(d)
		ps := strings.Split(runtime.FuncForPC(pc).Name(), ".")
		pl := len(ps)
		if ps[pl-2][0] == '(' {
			name = strings.Join(ps[0:pl-2], ".")
		} else {
			name = strings.Join(ps[0:pl-1], ".")
		}
	}
	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return FromModule(host, name)
}

// Valid returns true if s is string which is valid app name.
func Valid(s string) bool {
	re := regexp.MustCompile(Regexp)
	return re.MatchString(s)
}

// Parse takes a string address and returns a new Address instance.
// If the address is not valid, an error is returned.
func Parse(rawAddr string) (*Address, error) {
	if rawAddr == "" {
		return nil, fmt.Errorf("%w: empty address", ErrAddr)
	}
	if !strings.HasPrefix(rawAddr, "happy://") {
		host, err := Current()
		if err != nil {
			return nil, err
		}
		full, err := url.JoinPath(host.String(), rawAddr)
		if err != nil {
			return nil, err
		}
		return Parse(full)
	}
	url, err := url.Parse(rawAddr)
	if err != nil {
		return nil, err
	}
	urlparts := strings.Split(url.Path, "/")
	var instance string
	if len(urlparts) > 1 {
		instance = urlparts[1]
	}
	return &Address{
		url:      url,
		host:     url.Host,
		instance: instance,
	}, nil
}

func ensure(in string) string {
	if in == "-" {
		return in
	}
	var b bytes.Buffer
	for _, c := range in {
		isAlnum := unicode.Is(alnum, c)
		isSpace := unicode.IsSpace(c)
		isLower := unicode.IsLower(c)
		if isSpace || (!isAlnum && c != dot) {
			continue
		}
		if !isLower {
			c = unicode.ToLower(c)
		}
		b.WriteRune(c)
	}
	return b.String()
}
