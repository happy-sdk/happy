// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2023 The Happy Authors
package cron

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var secondParser = NewParser(Second | Minute | Hour | Dom | Month | DowOptional | Descriptor)

func TestRange(t *testing.T) {
	// zero := uint64(0)
	ranges := []struct {
		expr     string
		bounds   bounds
		field    ParseOption
		expected uint64
		err      string
	}{
		// Single values (Dom)
		{"5", bounds{0, 31, nil}, Dom, 1 << 5, ""},
		{"0", bounds{0, 31, nil}, Dom, 1 << 0, ""},
		{"31", bounds{0, 31, nil}, Dom, 1 << 31, ""},

		// Single values (Dow)
		{"5", bounds{0, 6, nil}, Dow, 1 << 5, ""},
		{"0", bounds{0, 6, nil}, Dow, 1 << 0, ""},
		{"6", bounds{0, 6, nil}, Dow, 1 << 6, ""},

		// // Ranges (Dom)
		// {"5-5", bounds{0, 31, nil}, Dom, 1 << 5, ""},
		// {"5-6", bounds{0, 31, nil}, Dom, 1<<5 | 1<<6, ""},
		// {"28-31", bounds{0, 31, nil}, Dom, 1<<28 | 1<<29 | 1<<30 | 1<<31, ""},

		// // Ranges (Dow)
		// {"5-6", bounds{0, 6, nil}, Dow, 1<<5 | 1<<6, ""},
		// {"1-5", bounds{0, 6, nil}, Dow, getBits(1, 5, 1), ""}, // Matches @lastweekday Dow

		// // Steps (Dom)
		// {"5-6/2", bounds{0, 31, nil}, Dom, 1 << 5, ""},
		// {"5-7/2", bounds{0, 31, nil}, Dom, 1<<5 | 1<<7, ""},
		// {"5-7/1", bounds{0, 31, nil}, Dom, 1<<5 | 1<<6 | 1<<7, ""},

		// // Steps (Dow)
		// {"1-5/2", bounds{0, 6, nil}, Dow, 1<<1 | 1<<3 | 1<<5, ""},

		// // Wildcard (Dom)
		// {"*", bounds{1, 31, nil}, Dom, getBits(1, 31, 1) | starBit, ""},
		// {"*/2", bounds{1, 31, nil}, Dom, getBits(1, 31, 2), ""},

		// // Wildcard (Dow)
		// {"*", bounds{0, 6, nil}, Dow, getBits(0, 6, 1) | starBit, ""},
		// {"*/2", bounds{0, 6, nil}, Dow, getBits(0, 6, 2), ""},

		// // Last day/weekday (L)
		// {"L", bounds{28, 31, nil}, Dom, lastDomBit, ""},                     // @lastday
		// {"5L", bounds{0, 6, nil}, Dow, 1<<5 | lastDowBit, ""},               // Last Friday
		// {"1-5L", bounds{0, 6, nil}, Dow, getBits(1, 5, 1) | lastDowBit, ""}, // @lastweekday

		// // Aliases (Month)
		// {"jan", bounds{1, 12, map[string]uint{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6, "jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}}, Month, 1 << 1, ""},
		// {"jan-mar", bounds{1, 12, map[string]uint{"jan": 1, "feb": 2, "mar": 3}}, Month, 1<<1 | 1<<2 | 1<<3, ""},

		// // Errors (Dom)
		// {"5--5", bounds{0, 31, nil}, Dom, zero, "too many hyphens"},
		// {"32", bounds{0, 31, nil}, Dom, zero, "above maximum"},
		// {"0", bounds{1, 31, nil}, Dom, zero, "below minimum"},
		// {"5-3", bounds{0, 31, nil}, Dom, zero, "beyond end of range"},
		// {"*/-12", bounds{0, 31, nil}, Dom, zero, "negative number"},
		// {"*//2", bounds{0, 31, nil}, Dom, zero, "too many slashes"},
		// {"*/0", bounds{0, 31, nil}, Dom, zero, "should be a positive number"},
		// {"jan", bounds{0, 31, nil}, Dom, zero, "failed to parse int from"},

		// // Errors (Dow)
		// {"7", bounds{0, 6, nil}, Dow, zero, "above maximum"},
		// {"mon", bounds{0, 6, nil}, Dow, zero, "failed to parse int from"},

		// // Errors (Second, no L support)
		// {"L", bounds{0, 59, nil}, Second, zero, "last not supported"},
		// {"5L", bounds{0, 59, nil}, Second, zero, "last not supported"},

		// // Errors (Month, no L support)
		// {"L", bounds{1, 12, nil}, Month, zero, "last not supported"},
		// {"janL", bounds{1, 12, map[string]uint{"jan": 1}}, Month, zero, "last not supported"},
	}

	for _, c := range ranges {
		t.Run(c.expr+"_"+c.field.String(), func(t *testing.T) {
			actual, err := getRange(c.expr, c.bounds, c.field)
			if len(c.err) != 0 && (err == nil || !strings.Contains(err.Error(), c.err)) {
				t.Errorf("%s (%s) => expected error containing %q, got %v", c.expr, c.field, c.err, err)
			}
			if len(c.err) == 0 && err != nil {
				t.Errorf("%s (%s) => unexpected error %v", c.expr, c.field, err)
			}
			if actual != c.expected {
				t.Errorf("%s (%s) => expected %d, got %d", c.expr, c.field, c.expected, actual)
			}
		})
	}
}

func TestField(t *testing.T) {
	// zero := uint64(0)

	tests := []struct {
		expr     string
		bounds   bounds
		field    ParseOption
		expected uint64
		err      string
	}{
		// Dom tests
		{"5", bounds{0, 31, nil}, Dom, 1 << 5, ""},
		{"5,6", bounds{0, 31, nil}, Dom, 1<<5 | 1<<6, ""},
		{"5,6,7", bounds{0, 31, nil}, Dom, 1<<5 | 1<<6 | 1<<7, ""},
		// {"1,5-7/2,3", bounds{0, 31, nil}, Dom, 1<<1 | 1<<5 | 1<<7 | 1<<3, ""},
		// {"5,L", bounds{28, 31, nil}, Dom, 1<<5 | lastDomBit, ""}, // @lastday
		// {"*", bounds{1, 31, nil}, Dom, getBits(1, 31, 1) | starBit, ""},
		// {"5,32", bounds{0, 31, nil}, Dom, zero, "above maximum"},
		// {"5,jan", bounds{0, 31, nil}, Dom, zero, "failed to parse int from"},
		// {"5,L", bounds{0, 27, nil}, Dom, zero, "last not supported"}, // L invalid for small range

		// // Dow tests
		// {"5", bounds{0, 6, nil}, Dow, 1 << 5, ""},
		// {"5,6", bounds{0, 6, nil}, Dow, 1<<5 | 1<<6, ""},
		// {"1,3,5", bounds{0, 6, nil}, Dow, 1<<1 | 1<<3 | 1<<5, ""},
		// {"1,3-5/2", bounds{0, 6, nil}, Dow, 1<<1 | 1<<3 | 1<<5, ""},
		// {"5,1-5L", bounds{0, 6, nil}, Dow, 1<<5 | getBits(1, 5, 1) | lastDowBit, ""}, // @lastweekday
		// {"5L", bounds{0, 6, nil}, Dow, 1<<5 | lastDowBit, ""},                        // Last Friday
		// {"*", bounds{0, 6, nil}, Dow, getBits(0, 6, 1) | starBit, ""},
		// {"5,7", bounds{0, 6, nil}, Dow, zero, "above maximum"},
		// {"5,mon", bounds{0, 6, nil}, Dow, zero, "failed to parse int from"},

		// // Month tests (with aliases)
		// {"jan", bounds{1, 12, map[string]uint{"jan": 1, "feb": 2, "mar": 3}}, Month, 1 << 1, ""},
		// {"jan,mar", bounds{1, 12, map[string]uint{"jan": 1, "feb": 2, "mar": 3}}, Month, 1<<1 | 1<<3, ""},
		// {"1,jan-mar", bounds{1, 12, map[string]uint{"jan": 1, "feb": 2, "mar": 3}}, Month, 1<<1 | 1<<2 | 1<<3, ""},
		// {"jan,13", bounds{1, 12, map[string]uint{"jan": 1}}, Month, zero, "above maximum"},
		// {"jan,apr", bounds{1, 12, map[string]uint{"jan": 1}}, Month, zero, "failed to parse alias"},

		// // Second tests (no L support)
		// {"0,30", bounds{0, 59, nil}, Second, 1<<0 | 1<<30, ""},
		// {"0,30-40/2", bounds{0, 59, nil}, Second, 1<<0 | 1<<30 | 1<<32 | 1<<34 | 1<<36 | 1<<38 | 1<<40, ""},
		// {"0,L", bounds{0, 59, nil}, Second, zero, "last not supported"},
		// {"0,30L", bounds{0, 59, nil}, Second, zero, "last not supported"},
	}

	for _, c := range tests {
		t.Run(c.expr+"_"+c.field.String(), func(t *testing.T) {
			actual, err := getField(c.expr, c.bounds, c.field)
			if len(c.err) != 0 && (err == nil || !strings.Contains(err.Error(), c.err)) {
				t.Errorf("%s (%s) => expected error containing %q, got %v", c.expr, c.field, c.err, err)
			}
			if len(c.err) == 0 && err != nil {
				t.Errorf("%s (%s) => unexpected error %v", c.expr, c.field, err)
			}
			if actual != c.expected {
				t.Errorf("%s (%s) => expected %d, got %d", c.expr, c.field, c.expected, actual)
			}
		})
	}
}

func TestAll(t *testing.T) {
	allBits := []struct {
		r        bounds
		expected uint64
	}{
		{minutes, 0xfffffffffffffff}, // 0-59: 60 ones
		{hours, 0xffffff},            // 0-23: 24 ones
		{dom, 0xfffffffe},            // 1-31: 31 ones, 1 zero
		{months, 0x1ffe},             // 1-12: 12 ones, 1 zero
		{dow, 0x7f},                  // 0-6: 7 ones
	}

	for _, c := range allBits {
		actual := all(c.r) // all() adds the starBit, so compensate for that..
		if c.expected|starBit != actual {
			t.Errorf("%d-%d/%d => expected %b, got %b",
				c.r.min, c.r.max, 1, c.expected|starBit, actual)
		}
	}
}

func TestBits(t *testing.T) {
	bits := []struct {
		min, max, step uint
		expected       uint64
	}{
		{0, 0, 1, 0x1},
		{1, 1, 1, 0x2},
		{1, 5, 2, 0x2a}, // 101010
		{1, 4, 2, 0xa},  // 1010
	}

	for _, c := range bits {
		actual := getBits(c.min, c.max, c.step)
		if c.expected != actual {
			t.Errorf("%d-%d/%d => expected %b, got %b",
				c.min, c.max, c.step, c.expected, actual)
		}
	}
}

func TestParseScheduleErrors(t *testing.T) {
	var tests = []struct{ expr, err string }{
		{"* 5 j * * *", "failed to parse int from"},
		{"@every Xm", "failed to parse duration"},
		{"@unrecognized", "unrecognized descriptor"},
		{"* * * *", "expected 5 to 6 fields"},
		{"", "empty spec string"},
	}
	for _, c := range tests {
		actual, err := secondParser.Parse(c.expr)
		if err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s => expected %v, got %v", c.expr, c.err, err)
		}
		if actual != nil {
			t.Errorf("expected nil schedule on error, got %v", actual)
		}
	}
}

func TestParseSchedule(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	entries := []struct {
		parser   Parser
		expr     string
		expected Schedule
	}{
		{secondParser, "0 5 * * * *", every5min(time.Local)},
		{standardParser, "5 * * * *", every5min(time.Local)},
		{secondParser, "CRON_TZ=UTC  0 5 * * * *", every5min(time.UTC)},
		{standardParser, "CRON_TZ=UTC  5 * * * *", every5min(time.UTC)},
		{secondParser, "CRON_TZ=Asia/Tokyo 0 5 * * * *", every5min(tokyo)},
		{secondParser, "@every 5m", ConstantDelaySchedule{Delay: 5 * time.Minute}},
		{secondParser, "@midnight", midnight(time.Local)},
		{secondParser, "TZ=UTC  @midnight", midnight(time.UTC)},
		{secondParser, "TZ=Asia/Tokyo @midnight", midnight(tokyo)},
		{secondParser, "@yearly", annual(time.Local)},
		{secondParser, "@annually", annual(time.Local)},
		{secondParser, "@quarterly", quarterly(time.Local)},
		{secondParser, "@monthly", monthly(time.Local)},
		{secondParser, "@lastday", lastday(time.Local)},
		{
			parser: secondParser,
			expr:   "* 5 * * * *",
			expected: &ScheduleSpec{
				Second:   all(seconds),
				Minute:   1 << 5,
				Hour:     all(hours),
				Dom:      all(dom),
				Month:    all(months),
				Dow:      all(dow),
				Location: time.Local,
			},
		},
	}

	for _, c := range entries {
		actual, err := c.parser.Parse(c.expr)
		if err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => expected %b, got %b", c.expr, c.expected, actual)
		}
	}
}

func TestOptionalSecondSchedule(t *testing.T) {
	parser := NewParser(SecondOptional | Minute | Hour | Dom | Month | Dow | Descriptor)
	entries := []struct {
		expr     string
		expected Schedule
	}{
		{"0 5 * * * *", every5min(time.Local)},
		{"5 5 * * * *", every5min5s(time.Local)},
		{"5 * * * *", every5min(time.Local)},
	}

	for _, c := range entries {
		actual, err := parser.Parse(c.expr)
		if err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => expected %b, got %b", c.expr, c.expected, actual)
		}
	}
}

func TestNormalizeFields(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		options  ParseOption
		expected []string
	}{
		{
			"AllFields_NoOptional",
			[]string{"0", "5", "*", "*", "*", "*"},
			Second | Minute | Hour | Dom | Month | Dow | Descriptor,
			[]string{"0", "5", "*", "*", "*", "*"},
		},
		{
			"AllFields_SecondOptional_Provided",
			[]string{"0", "5", "*", "*", "*", "*"},
			SecondOptional | Minute | Hour | Dom | Month | Dow | Descriptor,
			[]string{"0", "5", "*", "*", "*", "*"},
		},
		{
			"AllFields_SecondOptional_NotProvided",
			[]string{"5", "*", "*", "*", "*"},
			SecondOptional | Minute | Hour | Dom | Month | Dow | Descriptor,
			[]string{"0", "5", "*", "*", "*", "*"},
		},
		{
			"SubsetFields_NoOptional",
			[]string{"5", "15", "*"},
			Hour | Dom | Month,
			[]string{"0", "0", "5", "15", "*", "*"},
		},
		{
			"SubsetFields_DowOptional_Provided",
			[]string{"5", "15", "*", "4"},
			Hour | Dom | Month | DowOptional,
			[]string{"0", "0", "5", "15", "*", "4"},
		},
		{
			"SubsetFields_DowOptional_NotProvided",
			[]string{"5", "15", "*"},
			Hour | Dom | Month | DowOptional,
			[]string{"0", "0", "5", "15", "*", "*"},
		},
		{
			"SubsetFields_SecondOptional_NotProvided",
			[]string{"5", "15", "*"},
			SecondOptional | Hour | Dom | Month,
			[]string{"0", "0", "5", "15", "*", "*"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := normalizeFields(test.input, test.options)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestNormalizeFields_Errors(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		options ParseOption
		err     string
	}{
		{
			"TwoOptionals",
			[]string{"0", "5", "*", "*", "*", "*"},
			SecondOptional | Minute | Hour | Dom | Month | DowOptional,
			"",
		},
		{
			"TooManyFields",
			[]string{"0", "5", "*", "*"},
			SecondOptional | Minute | Hour,
			"",
		},
		{
			"NoFields",
			[]string{},
			SecondOptional | Minute | Hour,
			"",
		},
		{
			"TooFewFields",
			[]string{"*"},
			SecondOptional | Minute | Hour,
			"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := normalizeFields(test.input, test.options)
			if err == nil {
				t.Errorf("expected an error, got none. results: %v", actual)
			}
			if !strings.Contains(err.Error(), test.err) {
				t.Errorf("expected error %q, got %q", test.err, err.Error())
			}
		})
	}
}

func TestStandardScheduleSpec(t *testing.T) {
	entries := []struct {
		expr     string
		expected Schedule
		err      string
	}{
		{
			expr:     "5 * * * *",
			expected: &ScheduleSpec{1 << seconds.min, 1 << 5, all(hours), all(dom), all(months), all(dow), time.Local, false},
		},
		{
			expr:     "@every 5m",
			expected: ConstantDelaySchedule{Delay: time.Duration(5) * time.Minute},
		},
		{
			expr: "5 j * * *",
			err:  "failed to parse int from",
		},
		{
			expr: "* * * *",
			err:  "expected exactly 5 fields",
		},
	}

	for _, c := range entries {
		actual, err := ParseStandard(c.expr)
		if len(c.err) != 0 && (err == nil || !strings.Contains(err.Error(), c.err)) {
			t.Errorf("%s => expected %v, got %v", c.expr, c.err, err)
		}
		if len(c.err) == 0 && err != nil {
			t.Errorf("%s => unexpected error %v", c.expr, err)
		}
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("%s => expected %b, got %b", c.expr, c.expected, actual)
		}
	}
}

func TestNoDescriptorParser(t *testing.T) {
	parser := NewParser(Minute | Hour)
	_, err := parser.Parse("@every 1m")
	if err == nil {
		t.Error("expected an error, got none")
	}
}

func every5min(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{1 << 0, 1 << 5, all(hours), all(dom), all(months), all(dow), loc, false}
}

func every5min5s(loc *time.Location) *ScheduleSpec {
	return &ScheduleSpec{1 << 5, 1 << 5, all(hours), all(dom), all(months), all(dow), loc, false}
}
