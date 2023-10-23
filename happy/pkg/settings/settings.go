// Copyright 2023 The Happy Authors
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file.

package settings

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"log/slog"

	"github.com/happy-sdk/vars"
)

var (
	ErrEmptyKey         = errors.New("empty key")
	ErrKeyExists        = errors.New("key already exists")
	ErrSchemaComposed   = errors.New("blueprint.Schema can only called once")
	ErrGroupFreezed     = errors.New("settings group already freezed")
	ErrProfile          = errors.New("profile error")
	ErrInvalidGroup     = errors.New("invalid group")
	ErrDuplicateSetting = errors.New("duplicated setting")
	ErrFailedToApply    = errors.New("failed to attach definition")
	ErrNotFound         = errors.New("not found")
	ErrSetting          = errors.New("setting error")
)

// Mutability used to define settings mutability
type Mutability uint8

const (
	// Mutability
	SettingImmutable Mutability = 255
	SettingOnce      Mutability = 254
	SettingMutable   Mutability = 253
)

// Setting represents individual settings used in runtime. Each setting has mutability,
// a kind (data type), a variable, and a default value.
type Setting struct {
	m     Mutability
	k     Kind
	v     vars.Variable
	d     any
	isset bool
}

func (s Setting) String() string {
	return s.v.String()
}

func (s Setting) Attr() slog.Attr {
	return slog.Any(s.v.Name(), s.v.Any())
}

func (s Setting) reset() (Setting, error) {
	v, err := vars.NewAs(s.v.Name(), s.d, s.m != SettingImmutable, s.k.varskind())
	if err != nil {
		return Setting{}, fmt.Errorf("%w: %s with value %v %w", ErrFailedToApply, s.v.Name(), s.d, err)
	}
	return Setting{
		m:     s.m,
		k:     s.k,
		v:     v,
		d:     s.d,
		isset: true,
	}, nil
}

func (s Setting) set(val any) (Setting, error) {
	if s.v.Any() == val {
		return s, nil
	}

	if s.m == SettingImmutable {
		return s, fmt.Errorf("%w: setting %s can not be mutated", ErrSetting, s.v.Name())
	}

	if s.m == SettingOnce && s.isset {
		return s, fmt.Errorf("%w: setting %s can be mutated only once", ErrSetting, s.v.Name())
	}

	s2, err := newSetting(
		Definition{
			m:     s.m,
			key:   s.v.Name(),
			value: val,
			kind:  s.k,
		},
	)
	s2.isset = true
	return s2, err
}

func newSetting(d Definition) (Setting, error) {
	v, err := vars.NewAs(d.key, d.value, d.m != SettingImmutable, d.kind.varskind())
	if err != nil {
		return Setting{}, fmt.Errorf("%w: %s with value %v %w", ErrFailedToApply, d.key, d.value, err)
	}

	return Setting{
		m: d.m,
		k: d.kind,
		v: v,
		d: d.value,
	}, nil
}

// Blueprint Struct: Allows developers to define settings profiles for an application.
// It consists of global settings and groups of settings, which can be defined using the Define method.
// The Schema method is used to create a settings schema.
type Blueprint struct {
	mu      sync.RWMutex
	groups  map[string]*definitionGroup
	global  *definitionGroup
	freezed bool
}

func New() *Blueprint {
	return &Blueprint{}
}

func (b *Blueprint) Schema(version string) (s Schema, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.freezed {
		return s, ErrSchemaComposed
	}
	b.freezed = true
	schema := Schema{
		version: version,
	}

	if b.global != nil {
		schema.global, err = b.global.freeze()
		if err != nil {
			return s, err
		}
	}
	if b.groups != nil {
		schema.groups = make(map[string]SchemaGroup)
		for groupName, group := range b.groups {
			schema.groups[groupName], err = group.freeze()
			if err != nil {
				return s, err
			}
		}
	}

	return schema, nil
}

func (b *Blueprint) Define(d Definition) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.global == nil {
		b.global = &definitionGroup{}
	}
	groupName, key, isGroup := strings.Cut(d.key, "/")
	if isGroup {
		if b.groups == nil {
			b.groups = make(map[string]*definitionGroup)
		}
		d.key = key
		group, ok := b.groups[groupName]
		if !ok {
			group = &definitionGroup{}
			b.groups[groupName] = group
		}
		return group.define(d)
	}
	return b.global.define(d)
}

// Definition defines settings in a Blueprint. It includes information such as mutability,
// the setting key, the default value, and the kind of the setting.
type Definition struct {
	m     Mutability
	key   string
	value any
	kind  Kind
}

func Immutable(key string, value interface{}, kind Kind) Definition {
	return NewDefinition(SettingImmutable, key, value, kind)
}

func Mutable(key string, value interface{}, kind Kind) Definition {
	return NewDefinition(SettingMutable, key, value, kind)
}

func Once(key string, value interface{}, kind Kind) Definition {
	return NewDefinition(SettingOnce, key, value, kind)
}

func NewDefinition(m Mutability, key string, value any, kind Kind) Definition {
	return Definition{
		m:     m,
		key:   key,
		value: value,
		kind:  kind,
	}
}

// definitionGroup A group of settings within a Blueprint, which can be frozen once defined.
type definitionGroup struct {
	name          string
	definositions map[string]Definition
	freezed       bool
}

func (g *definitionGroup) define(d Definition) error {
	if d.key == "" {
		return ErrEmptyKey
	}

	if _, ok := g.definositions[d.key]; ok {
		return fmt.Errorf("%w: %s", ErrKeyExists, d.key)
	}
	if g.definositions == nil {
		g.definositions = make(map[string]Definition)
	}
	g.definositions[d.key] = d
	return nil
}

func (g *definitionGroup) freeze() (SchemaGroup, error) {

	if g.freezed {
		return SchemaGroup{}, fmt.Errorf("%w: %s", ErrGroupFreezed, g.name)
	}
	g.freezed = true

	group := SchemaGroup{
		valid: true,
		name:  g.name,
	}
	if group.settings == nil {
		group.settings = make(map[string]Definition)
	}
	for name, defintion := range g.definositions {
		group.settings[name] = defintion
	}
	return group, nil
}

// SchemaGroup represents a group of settings within a settings schema.
// It contains a collection of settings and a name.
type SchemaGroup struct {
	valid    bool
	name     string
	settings map[string]Definition
}

type group struct {
	name     string
	settings map[string]Setting
}

func newGroup(g SchemaGroup) (*group, error) {
	group := &group{
		name: g.name,
	}

	for _, s := range g.settings {
		if err := group.attach(s); err != nil {
			return nil, err
		}
	}
	return group, nil
}

func (g *group) attach(d Definition) (err error) {
	if g.settings == nil {
		g.settings = make(map[string]Setting)
	}
	if _, ok := g.settings[d.key]; ok {
		return fmt.Errorf("%w: %s in group %s", ErrDuplicateSetting, d.key, g.name)
	}

	g.settings[d.key], err = newSetting(d)
	return nil
}

func (g *group) has(key string) (found bool) {
	_, found = g.settings[key]
	return
}

func (g *group) get(key string) (Setting, error) {
	setting, ok := g.settings[key]
	if !ok {
		return Setting{}, fmt.Errorf("%w: setting %s in group %s", ErrNotFound, key, g.name)
	}
	return setting, nil
}

func (g *group) set(key string, val any) error {

	s, err := g.get(key)
	if err != nil {
		return err
	}

	g.settings[key], err = s.set(val)
	if err != nil {
		return fmt.Errorf("%s: %w", g.name, err)
	}
	return nil
}

func (g *group) reset(key string) error {
	s, err := g.get(key)
	if err != nil {
		return err
	}

	g.settings[key], err = s.reset()
	return err
}

// Kind enumerates different kinds of setting values (e.g., bool, int, string).
type Kind uint8

const (
	// Settings Value types
	KindBool    Kind = Kind(vars.KindBool)
	KindInt     Kind = Kind(vars.KindInt)
	KindUint    Kind = Kind(vars.KindUint)
	KindFloat32 Kind = Kind(vars.KindFloat32)
	KindFloat64 Kind = Kind(vars.KindFloat64)

	KindString   Kind = Kind(vars.KindString)
	KindDuration Kind = Kind(vars.KindDuration)
	KindTime     Kind = Kind(vars.KindTime)
)

func (k Kind) varskind() vars.Kind {
	return vars.Kind(k)
}

// Preferences allows you to create a runtime profile with user settings,
// e.g., loading settings from a file.
type Preferences struct{}

var Default = Preferences{}

// Profile represents the active state of settings for the application.
// You can get, set and reset settings using this structure.
type Profile struct {
	mu       sync.RWMutex
	version  string
	global   *group
	settings map[string]*group
}

func (p *Profile) Version() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.version
}

func (p *Profile) Get(key string) (Setting, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	group, k, err := p.getGroup(key)
	if err != nil {
		return Setting{}, err
	}
	return group.get(k)
}

func (p *Profile) Set(key string, value any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	group, k, err := p.getGroup(key)
	if err != nil {
		return err
	}

	return group.set(k, value)
}

func (p *Profile) Reset(key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	group, k, err := p.getGroup(key)
	if err != nil {
		return err
	}

	return group.reset(k)
}

func (p *Profile) getGroup(key string) (g *group, k string, err error) {
	groupName, k, found := strings.Cut(key, "/")

	if !found {
		return p.global, key, err
	}

	group, hasGroup := p.settings[groupName]
	if !hasGroup {
		return nil, key, fmt.Errorf("%w: group named %s", ErrNotFound, groupName)
	}
	return group, k, nil
}

func (p *Profile) apply(g SchemaGroup) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !g.valid {
		return fmt.Errorf("%w: %s applied to settings profile", ErrInvalidGroup, g.name)
	}

	if g.name == "" {
		p.global, err = newGroup(g)
		if err != nil {
			return err
		}
	} else {
		group, err := newGroup(g)
		if err != nil {
			return err
		}
		p.settings[group.name] = group
	}
	return nil
}

// Schema defines the state of a Blueprint and helps with migrations between
// different blueprint/app versions. It contains global settings and groups of settings.
type Schema struct {
	version string
	groups  map[string]SchemaGroup
	global  SchemaGroup
}

func (s Schema) Profile(p Preferences) (*Profile, error) {
	profile := &Profile{
		version: s.version,
	}
	if s.global.valid {
		if err := profile.apply(s.global); err != nil {
			return nil, err
		}
	}

	for name, group := range s.groups {
		if err := profile.apply(group); err != nil {
			return nil, fmt.Errorf("%w: failed to apply schema group %s: %w", ErrProfile, name, err)
		}
	}
	return profile, nil
}
