// Copyright 2023 Marko Kungla
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package logging

import (
	"runtime"
	"strconv"
	"time"
)

type RecordTimestampKind int

const (
	RecordTimestampLayout RecordTimestampKind = iota
	RecordTimestampUnix
	RecordTimestampUnixMilli
	RecordTimestampUnixMicro
	RecordTimestampUnixNano
)

type RecordFlag uint8

const (
	WithRecordTimestamp RecordFlag = 1 << iota
	WithRecordLevel
	WithRecordMessage
	WithRecordError
	WithRecordData
	WithRecordSource

	WithAllFields = WithRecordTimestamp | WithRecordLevel |
		WithRecordMessage | WithRecordError |
		WithRecordData | WithRecordSource
)

func (rf RecordFlag) Has(flag RecordFlag) bool {
	return rf&flag != 0
}

type rawRecord struct {
	level Level
	ts    time.Time
	msg   string
	err   error
	args  []any

	src string
}

func newRawRecord(ts time.Time, level Level, msg string, err error, args ...any) *rawRecord {
	r := &rawRecord{
		ts:    ts,
		level: level,
		msg:   msg,
		err:   err,
		args:  args,
	}
	return r
}

func (rr *rawRecord) setSource(calldepth int) {
	var pcs [1]uintptr
	runtime.Callers(calldepth+3, pcs[:])
	fs := runtime.CallersFrames([]uintptr{pcs[0]})
	f, _ := fs.Next()
	if f.File != "" {
		rr.src = f.File + ":" + strconv.Itoa(f.Line)
	}
}

func (rr *rawRecord) record(flags RecordFlag, tlay string, rfunc []AttrReplaceFunc) (Record, error) {
	var groups []string

	r := Record{
		Time:  rr.ts,
		level: rr.level,
	}

	if flags.Has(WithRecordTimestamp) {
		r.TimeString = NewAttr(TimeKey, rr.ts.Format(tlay))
	}
	if flags.Has(WithRecordLevel) {
		r.Level = NewAttr(LevelKey, rr.level)
	}
	if flags.Has(WithRecordMessage) && rr.msg != "" {
		r.Message = NewAttr(MessageKey, rr.msg)
	}
	if flags.Has(WithRecordError) && rr.err != nil {
		r.Error = NewAttr(ErrorKey, rr.err)
	}
	if flags.Has(WithRecordSource) && rr.src != "" {
		r.Source = NewAttr(SourceKey, rr.src)
	}

	var err error
	for _, rfunc := range rfunc {
		if r.TimeString.Kind() != AttrOmittedKind {
			if r.TimeString, err = rfunc(groups, r.TimeString); err != nil {
				return r, err
			}
		}
		if r.Level.Kind() != AttrOmittedKind {
			if r.Level, err = rfunc(groups, r.Level); err != nil {
				return r, err
			}
		}
		if r.Message.Kind() != AttrOmittedKind {
			if r.Message, err = rfunc(groups, r.Message); err != nil {
				return r, err
			}
		}
		if r.Error.Kind() != AttrOmittedKind {
			if r.Error, err = rfunc(groups, r.Error); err != nil {
				return r, err
			}
		}
		if r.Source.Kind() != AttrOmittedKind {
			if r.Source, err = rfunc(groups, r.Source); err != nil {
				return r, err
			}
		}
	}

	if len(rr.args) > 0 {
		attrs := attrsFromArgs(rr.args)
		for i, attr := range attrs {
			for _, rfunc := range rfunc {
				switch attr.Kind() {
				case AttrSingleKind:
					if attr, err = rfunc(groups, attr); err != nil {
						return r, err
					}
					continue
				case AttrObjectKind:
					var values []Attr
					for _, el := range attr.Value.Object() {
						e, err := rfunc(groups, el)
						if err != nil {
							return r, err
						}
						values = append(values, e)
					}
					attr.Value = AttrValue{
						kind:  AttrObjectKind,
						value: values,
					}
				}
			}
			attrs[i] = attr
		}

		r.Data = NewAttr(DataKey, attrs)
	}

	return r, nil
}

type Record struct {
	level      Level
	Time       time.Time
	TimeString Attr
	Level      Attr
	Error      Attr
	Message    Attr
	Source     Attr
	Data       Attr
}
