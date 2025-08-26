// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors
package cron

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Configuration options for creating a parser. Most options specify which
// fields should be included, while others enable features. If a field is not
// included the parser will assume a default value. These options do not change
// the order fields are parse in.
type ParseOption int

const (
	Second         ParseOption = 1 << iota // Seconds field, default 0
	SecondOptional                         // Optional seconds field, default 0
	Minute                                 // Minutes field, default 0
	Hour                                   // Hours field, default 0
	Dom                                    // Day of month field, default *
	Month                                  // Month field, default *
	Dow                                    // Day of week field, default *
	DowOptional                            // Optional day of week field, default *
	Descriptor                             // Allow descriptors such as @monthly, @weekly, etc.
)

func (o ParseOption) String() string {
	switch o {
	case Second:
		return "Second"
	case SecondOptional:
		return "SecondOptional"
	case Minute:
		return "Minute"
	case Hour:
		return "Hour"
	case Dom:
		return "Dom"
	case Month:
		return "Month"
	case Dow:
		return "Dow"
	case DowOptional:
		return "DowOptional"
	case Descriptor:
		return "Descriptor"
	}
	return "Unknown"
}

var places = []ParseOption{
	Second,
	Minute,
	Hour,
	Dom,
	Month,
	Dow,
}

var defaults = []string{
	"0",
	"0",
	"0",
	"*",
	"*",
	"*",
}

// A custom Parser that can be configured.
type Parser struct {
	options ParseOption
}

// NewParser creates a Parser with custom options.
//
// It panics if more than one Optional is given, since it would be impossible to
// correctly infer which optional is provided or missing in general.
//
// Examples
//
//	// Standard parser without descriptors
//	specParser := NewParser(Minute | Hour | Dom | Month | Dow)
//	sched, err := specParser.Parse("0 0 15 */3 *")
//
//	// Same as above, just excludes time fields
//	specParser := NewParser(Dom | Month | Dow)
//	sched, err := specParser.Parse("15 */3 *")
//
//	// Same as above, just makes Dow optional
//	specParser := NewParser(Dom | Month | DowOptional)
//	sched, err := specParser.Parse("15 */3")
func NewParser(options ParseOption) Parser {
	optionals := 0
	if options&DowOptional > 0 {
		optionals++
	}
	if options&SecondOptional > 0 {
		optionals++
	}
	if optionals > 1 {
		panic("multiple optionals may not be configured")
	}
	return Parser{options}
}

// Parse returns a new crontab schedule representing the given spec.
// It returns a descriptive error if the spec is not valid.
// It accepts crontab specs and features configured by NewParser.
func (p Parser) Parse(spec string) (Schedule, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

	if spec == "-" {
		return &ScheduleSpec{Disabled: true}, nil
	}

	var loc = time.Local
	if strings.HasPrefix(spec, "TZ=") || strings.HasPrefix(spec, "CRON_TZ=") {
		var err error
		i := strings.Index(spec, " ")
		if i == -1 {
			return nil, fmt.Errorf("invalid timezone specification: %s", spec)
		}
		eq := strings.Index(spec, "=")
		if loc, err = time.LoadLocation(spec[eq+1 : i]); err != nil {
			return nil, fmt.Errorf("provided bad location %s: %v", spec[eq+1:i], err)
		}
		spec = strings.TrimSpace(spec[i:])
	}

	if strings.HasPrefix(spec, "@") {
		if p.options&Descriptor == 0 {
			return nil, fmt.Errorf("parser does not accept descriptors: %v", spec)
		}
		return parseDescriptor(spec, loc)
	}

	fields := strings.Fields(spec)
	fields, err := normalizeFields(fields, p.options)
	if err != nil {
		return nil, err
	}

	field := func(field string, r bounds, fieldOpt ParseOption) uint64 {
		if err != nil {
			return 0
		}
		var bits uint64
		bits, err = getField(field, r, fieldOpt)
		return bits
	}

	var (
		second     = field(fields[0], seconds, Second)
		minute     = field(fields[1], minutes, Minute)
		hour       = field(fields[2], hours, Hour)
		dayofmonth = field(fields[3], dom, Dom)
		month      = field(fields[4], months, Month)
		dayofweek  = field(fields[5], dow, Dow)
	)
	if err != nil {
		return nil, err
	}

	return &ScheduleSpec{
		Second:   second,
		Minute:   minute,
		Hour:     hour,
		Dom:      dayofmonth,
		Month:    month,
		Dow:      dayofweek,
		Location: loc,
	}, nil
}

// normalizeFields takes a subset set of the time fields and returns the full set
// with defaults (zeroes) populated for unset fields.
//
// As part of performing this function, it also validates that the provided
// fields are compatible with the configured options.
func normalizeFields(fields []string, options ParseOption) ([]string, error) {
	if len(fields) == 1 && strings.HasPrefix(fields[0], "@") && options&Descriptor > 0 {
		return fields, nil // Defer to parseDescriptor
	}
	// Validate optionals & add their field to options
	optionals := 0
	if options&SecondOptional > 0 {
		options |= Second
		optionals++
	}
	if options&DowOptional > 0 {
		options |= Dow
		optionals++
	}
	if optionals > 1 {
		return nil, fmt.Errorf("multiple optionals may not be configured")
	}

	// Figure out how many fields we need
	max := 0
	for _, place := range places {
		if options&place > 0 {
			max++
		}
	}
	min := max - optionals

	// Validate number of fields
	if count := len(fields); count < min || count > max {
		if min == max {
			return nil, fmt.Errorf("expected exactly %d fields, found %d: %s", min, count, fields)
		}
		return nil, fmt.Errorf("expected %d to %d fields, found %d: %s", min, max, count, fields)
	}

	// Populate the optional field if not provided
	if min < max && len(fields) == min {
		switch {
		case options&DowOptional > 0:
			fields = append(fields, defaults[5]) // TODO: improve access to default
		case options&SecondOptional > 0:
			fields = append([]string{defaults[0]}, fields...)
		default:
			return nil, fmt.Errorf("unknown optional field")
		}
	}

	// Populate all fields not part of options with their defaults
	n := 0
	expandedFields := make([]string, len(places))
	copy(expandedFields, defaults)
	for i, place := range places {
		if options&place > 0 {
			expandedFields[i] = fields[n]
			n++
		}
	}
	return expandedFields, nil
}

var standardParser = NewParser(
	Minute | Hour | Dom | Month | Dow | Descriptor,
)

// ParseStandard returns a new crontab schedule representing the given
// standardSpec (https://en.wikipedia.org/wiki/Cron). It requires 5 entries
// representing: minute, hour, day of month, month and day of week, in that
// order. It returns a descriptive error if the spec is not valid.
//
// It accepts
//   - Standard crontab specs, e.g. "* * * * ?"
//   - Descriptors, e.g. "@midnight", "@every 1h30m"
func ParseStandard(standardSpec string) (Schedule, error) {
	return standardParser.Parse(standardSpec)
}

var withOptionalSecondParser = NewParser(
	SecondOptional | Minute | Hour | Dom | Month | Dow | Descriptor,
)

func OptionalSecondParser() Parser {
	return withOptionalSecondParser
}

func ParseWithOptionalSecond(spec string) (Schedule, error) {
	return withOptionalSecondParser.Parse(spec)
}

var withOptionalDowParser = NewParser(Minute | Hour | Dom | Month | DowOptional | Descriptor)
var minimalParser = NewParser(Dom | Month | Dow | Descriptor)
var minimalWithOptionalDowParser = NewParser(Dom | Month | DowOptional | Descriptor)

// OptionalDowParser returns a parser
// with Minute, Hour, Dom, Month, optional Dow, and Descriptor support.
func OptionalDowParser() Parser {
	return withOptionalDowParser
}

// MinimalParser returns a parser
// with Dom, Month, Dow, and Descriptor support (no time fields).
func MinimalParser() Parser {
	return minimalParser
}

// MinimalWithOptionalDowParser returns a parser
// with Dom, Month, optional Dow, and Descriptor support.
func MinimalWithOptionalDowParser() Parser {
	return minimalWithOptionalDowParser
}

// ParseWithOptionalDow parses a cron spec
// with optional day of week.
func ParseWithOptionalDow(spec string) (Schedule, error) {
	return withOptionalDowParser.Parse(spec)
}

// ParseMinimal parses a minimal cron spec
// with Dom, Month, Dow, and Descriptor.
func ParseMinimal(spec string) (Schedule, error) {
	return minimalParser.Parse(spec)
}

// ParseMinimalWithOptionalDow parses a minimal cron spec
// with Dom, Month, optional Dow, and Descriptor.
func ParseMinimalWithOptionalDow(spec string) (Schedule, error) {
	return minimalWithOptionalDowParser.Parse(spec)
}

// getField returns an Int with the bits set representing all of the times that
// the field represents or error parsing field value.  A "field" is a comma-separated
// list of "ranges".
func getField(field string, r bounds, fieldOpt ParseOption) (uint64, error) {
	var bits uint64
	ranges := strings.FieldsFunc(field, func(r rune) bool { return r == ',' })
	for _, expr := range ranges {
		bit, err := getRange(expr, r, fieldOpt)
		if err != nil {
			return bits, err
		}
		bits |= bit
	}
	return bits, nil
}

// getRange returns the bits indicated by the given expression:
//
//	number | number "-" number [ "/" number ]
//
// or error parsing range.
func getRange(expr string, r bounds, field ParseOption) (uint64, error) {
	var (
		start, end, step uint
		rangeAndStep     = strings.Split(expr, "/")
		lowAndHigh       = strings.Split(rangeAndStep[0], "-")
		singleDigit      = len(lowAndHigh) == 1
		err              error
	)

	var extra uint64
	if lowAndHigh[0] == "*" || lowAndHigh[0] == "?" {
		start = r.min
		end = r.max
		extra = starBit
	} else if lowAndHigh[0] == "L" && (field == Dom || field == Dow) {
		if field == Dom {
			start = 28 // Possible last days of month
			end = 31
			extra = lastDomBit
		} else { // Dow
			start = r.min // Use the specified weekday (resolved in Next)
			end = start
			extra = lastDowBit
		}
	} else {
		start, err = parseIntOrName(lowAndHigh[0], r.names)
		if err != nil {
			return 0, err
		}
		switch len(lowAndHigh) {
		case 1:
			end = start
		case 2:
			end, err = parseIntOrName(lowAndHigh[1], r.names)
			if err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("too many hyphens: %s", expr)
		}
	}

	switch len(rangeAndStep) {
	case 1:
		step = 1
	case 2:
		step, err = mustParseInt(rangeAndStep[1])
		if err != nil {
			return 0, err
		}
		if singleDigit {
			end = r.max
		}
		if step > 1 {
			extra = 0
		}
	default:
		return 0, fmt.Errorf("too many slashes: %s", expr)
	}

	if start < r.min {
		return 0, fmt.Errorf("beginning of range (%d) below minimum (%d): %s", start, r.min, expr)
	}
	if end > r.max {
		return 0, fmt.Errorf("end of range (%d) above maximum (%d): %s", end, r.max, expr)
	}
	if start > end {
		return 0, fmt.Errorf("beginning of range (%d) beyond end of range (%d): %s", start, end, expr)
	}
	if step == 0 {
		return 0, fmt.Errorf("step of range should be a positive number: %s", expr)
	}

	return getBits(start, end, step) | extra, nil
}

// parseIntOrName returns the (possibly-named) integer contained in expr.
func parseIntOrName(expr string, names map[string]uint) (uint, error) {
	if names != nil {
		if namedInt, ok := names[strings.ToLower(expr)]; ok {
			return namedInt, nil
		}
	}
	return mustParseInt(expr)
}

// mustParseInt parses the given expression as an int or returns an error.
func mustParseInt(expr string) (uint, error) {
	num, err := strconv.Atoi(expr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse int from %s: %s", expr, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("negative number (%d) not allowed: %s", num, expr)
	}

	return uint(num), nil
}

// getBits sets all bits in the range [min, max], modulo the given step size.
func getBits(min, max, step uint) uint64 {
	var bits uint64

	// If step is 1, use shifts.
	if step == 1 {
		return ^(math.MaxUint64 << (max + 1)) & (math.MaxUint64 << min)
	}

	// Else, use a simple loop.
	for i := min; i <= max; i += step {
		bits |= 1 << i
	}
	return bits
}

// all returns all bits within the given bounds.  (plus the star bit)
func all(r bounds) uint64 {
	return getBits(r.min, r.max, 1) | starBit
}

// parseDescriptor returns a predefined schedule for the expression, or error if none matches.
func parseDescriptor(descriptor string, loc *time.Location) (Schedule, error) {
	switch descriptor {
	case "@yearly", "@annually":
		return annual(loc), nil
	case "@quarterly":
		return quarterly(loc), nil
	case "@monthly":
		return monthly(loc), nil
	case "@lastday":
		return lastday(loc), nil
	case "@lastweekday":
		return lastweekday(loc), nil
	case "@firstweekday":
		return firstweekday(loc), nil
	case "@weekly":
		return weekly(loc), nil
	case "@weekdays":
		return weekdays(loc), nil
	case "@weekends":
		return weekends(loc), nil
	case "@daily", "@midnight", "@dayly", "@everyday":
		return midnight(loc), nil
	case "@noon":
		return noon(loc), nil
	case "@hourly":
		return hourly(loc), nil
	}

	const every = "@every "
	if strings.HasPrefix(descriptor, every) {
		duration, err := time.ParseDuration(descriptor[len(every):])
		if err != nil {
			return nil, fmt.Errorf("failed to parse duration %s: %s", descriptor, err)
		}
		return Every(duration), nil
	}

	return nil, fmt.Errorf("unrecognized descriptor: %s", descriptor)
}
