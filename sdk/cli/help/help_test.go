// SPDX-License-Identifier: Apache-2.0
//
// Copyright © 2024 The Happy Authors

package help

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/happy-sdk/happy/pkg/vars/varflag"
)

func TestWordWrapWithPrefix(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		prefix     string
		lineLength int
		want       string
	}{
		{"short, no wrap", "hello world", "  ", 80, "hello world"},
		{"empty input", "", "  ", 80, ""},
		{
			"wraps at line length",
			"one two three four five six seven eight nine ten",
			"  ",
			20,
			"one two three four\n  five six seven eight\n  nine ten",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrapWithPrefix(tt.input, tt.prefix, tt.lineLength)
			if got != tt.want {
				t.Errorf("wordWrapWithPrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetMaxNameLength(t *testing.T) {
	cmds := []commandInfo{
		{name: "short"},
		{name: "a-much-longer-name"},
		{name: "mid"},
	}
	got := getMaxNameLength(cmds)
	want := len("a-much-longer-name")
	if got != want {
		t.Errorf("getMaxNameLength() = %d, want %d", got, want)
	}
}

func TestGetMaxNameLengthStripsAnsi(t *testing.T) {
	cmds := []commandInfo{
		{name: "\x1b[1mbold\x1b[0m"},
	}
	got := getMaxNameLength(cmds)
	if got != len("bold") {
		t.Errorf("getMaxNameLength() = %d, want %d (ANSI codes should be stripped)", got, len("bold"))
	}
}

func TestInfoCopyright(t *testing.T) {
	t.Run("empty CopyrightBy", func(t *testing.T) {
		i := &Info{}
		if got := i.copyright(); got != "" {
			t.Errorf("copyright() = %q, want empty", got)
		}
	})

	t.Run("with CopyrightBy", func(t *testing.T) {
		i := &Info{CopyrightBy: "The Happy Authors"}
		got := i.copyright()
		if !strings.Contains(got, "The Happy Authors") {
			t.Errorf("copyright() = %q, want it to contain %q", got, "The Happy Authors")
		}
	})
}

func TestInfoLicense(t *testing.T) {
	t.Run("empty License", func(t *testing.T) {
		i := &Info{}
		if got := i.license(); got != "" {
			t.Errorf("license() = %q, want empty", got)
		}
	})

	t.Run("with License", func(t *testing.T) {
		i := &Info{License: "Apache-2.0"}
		got := i.license()
		if !strings.Contains(got, "Apache-2.0") {
			t.Errorf("license() = %q, want it to contain %q", got, "Apache-2.0")
		}
	})
}

func TestInfoDescription(t *testing.T) {
	t.Run("empty Description", func(t *testing.T) {
		i := &Info{}
		if got := i.description(); got != "" {
			t.Errorf("description() = %q, want empty", got)
		}
	})

	t.Run("with Description", func(t *testing.T) {
		i := &Info{Description: "a test app"}
		got := i.description()
		if !strings.Contains(got, "a test app") {
			t.Errorf("description() = %q, want it to contain %q", got, "a test app")
		}
	})
}

func TestGetCategoryDesc(t *testing.T) {
	h := New(Info{}, Style{})
	h.AddCategoryDescriptions(map[string]string{"mycat": "My Category"})

	if got := h.getCategoryDesc("default"); got != "" {
		t.Errorf("getCategoryDesc(default) = %q, want empty", got)
	}
	if got := h.getCategoryDesc("unknown"); got != "" {
		t.Errorf("getCategoryDesc(unknown) = %q, want empty", got)
	}
}

func TestAddCommand(t *testing.T) {
	h := New(Info{}, Style{})
	h.AddCommand("", "cmd1", "first command")
	h.AddCommand("Tools", "cmd2", "second command")

	if len(h.cmds["default"]) != 1 || h.cmds["default"][0].name != "cmd1" {
		t.Errorf("expected cmd1 in default category, got %v", h.cmds["default"])
	}
	if len(h.cmds["Tools"]) != 1 || h.cmds["Tools"][0].name != "cmd2" {
		t.Errorf("expected cmd2 in Tools category, got %v", h.cmds["Tools"])
	}
}

func newTestFlag(t *testing.T, name string, aliases ...string) varflag.Flag {
	t.Helper()
	f, err := varflag.BoolFunc(name, false, "usage for "+name, aliases...)()
	if err != nil {
		t.Fatalf("failed to create test flag: %v", err)
	}
	return f
}

func TestAddFlags(t *testing.T) {
	h := New(Info{}, Style{})
	globalFlag := newTestFlag(t, "verbose", "v")
	sharedFlag := newTestFlag(t, "debug")
	cmdFlag := newTestFlag(t, "force", "f")

	h.AddGlobalFlags([]varflag.Flag{globalFlag})
	h.AddSharedFlags([]varflag.Flag{sharedFlag})
	h.AddCommandFlags([]varflag.Flag{cmdFlag})

	if len(h.globalFlags) != 1 || h.globalFlags[0].Flag != globalFlag.Flag() {
		t.Errorf("expected global flag %q, got %v", globalFlag.Flag(), h.globalFlags)
	}
	if len(h.sharedFlags) != 1 || h.sharedFlags[0].Flag != sharedFlag.Flag() {
		t.Errorf("expected shared flag %q, got %v", sharedFlag.Flag(), h.sharedFlags)
	}
	if len(h.flags) != 1 || h.flags[0].Flag != cmdFlag.Flag() {
		t.Errorf("expected command flag %q, got %v", cmdFlag.Flag(), h.flags)
	}
}

func TestAddFlagsNil(t *testing.T) {
	h := New(Info{}, Style{})
	h.AddGlobalFlags(nil)
	h.AddSharedFlags(nil)
	h.AddCommandFlags(nil)

	if len(h.globalFlags) != 0 || len(h.sharedFlags) != 0 || len(h.flags) != 0 {
		t.Error("expected no flags to be added for nil input")
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	out := <-done
	return out
}

func TestPrintFullOutput(t *testing.T) {
	h := New(Info{
		Name:           "TestApp",
		Description:    "a test application",
		Version:        "1.0.0",
		CopyrightBy:    "The Happy Authors",
		CopyrightSince: 2020,
		License:        "Apache-2.0",
		Usage:          []string{"testapp [flags]"},
		Info:           []string{"This is a longer info paragraph about the app."},
	}, Style{})

	h.AddCommand("", "hello", "prints hello")
	h.AddCommand("Tools", "build", "builds the project")
	h.AddCategoryDescriptions(map[string]string{"tools": "Build and dev tools"})
	h.AddGlobalFlags([]varflag.Flag{newTestFlag(t, "verbose", "v")})
	h.AddSharedFlags([]varflag.Flag{newTestFlag(t, "debug")})
	h.AddCommandFlags([]varflag.Flag{newTestFlag(t, "force", "f")})

	out := captureStdout(t, func() {
		if err := h.Print(); err != nil {
			t.Fatalf("Print() returned error: %v", err)
		}
	})

	for _, want := range []string{
		"TestApp", "1.0.0", "The Happy Authors", "Apache-2.0",
		"a test application", "testapp [flags]",
		"This is a longer info paragraph",
		"hello", "build", "TOOLS",
		"--verbose", "--debug", "--force",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected Print() output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPrintMinimal(t *testing.T) {
	h := New(Info{Name: "Minimal"}, Style{})
	out := captureStdout(t, func() {
		if err := h.Print(); err != nil {
			t.Fatalf("Print() returned error: %v", err)
		}
	})
	if !strings.Contains(out, "Minimal") {
		t.Errorf("expected output to contain %q, got:\n%s", "Minimal", out)
	}
}
