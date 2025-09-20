// SPDX-License-Identifier: Apache-2.0
// Copyright © 2025 The Happy Authors

package logging

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/happy-sdk/happy/tools/happyvet/internal/analysisinternal"
	"github.com/happy-sdk/happy/tools/happyvet/internal/typesinternal"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

//go:embed doc.go
var doc string

var MustExtractDoc = analysisinternal.MustExtractDoc

var Analyzer = &analysis.Analyzer{
	Name:     "happy_logging",
	Doc:      analysisinternal.MustExtractDoc(doc, "logging"),
	URL:      "https://pkg.go.dev/github.com/happy-sdk/happy/tools/happyvet/passes/logging",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var stringType = types.Universe.Lookup("string").Type()

// A position describes what is expected to appear in an argument position.
type position int

const (
	// key is an argument position that should hold a string key or an Attr.
	key position = iota
	// value is an argument position that should hold a value.
	value
	// unknown represents that we do not know if position should hold a key or a value.
	unknown
)

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		call := node.(*ast.CallExpr)
		fn := typeutil.StaticCallee(pass.TypesInfo, call)
		if fn == nil {
			return // not a static call
		}
		if call.Ellipsis != token.NoPos {
			return // skip calls with "..." args
		}
		skipArgs, ok := kvFuncSkipArgs(fn)
		if !ok {
			// Not a logger function that takes key-value pairs.
			return
		}
		// Happy SDK logging doesn't use slog.Attr, just string keys

		if isMethodExpr(pass.TypesInfo, call) {
			// Call is to a method value. Skip the first argument.
			skipArgs++
		}
		if len(call.Args) <= skipArgs {
			// Too few args; perhaps there are no k-v pairs.
			return
		}

		// Check this call.
		// The first position should hold a key.
		pos := key
		var badKeyArg ast.Expr // tracks the most recent bad key argument
		for _, arg := range call.Args[skipArgs:] {
			t := pass.TypesInfo.Types[arg].Type
			switch pos {
			case key:
				// Expect a string key.
				if t == stringType {
					pos = value
					badKeyArg = nil // reset since we found a good key
				} else if types.IsInterface(t) && types.AssignableTo(stringType, t) {
					// Could be a string, treat as unknown
					pos = unknown
				} else {
					// Definitely not a valid key, but continue processing
					badKeyArg = arg
					pos = value
				}

			case value:
				// Check if this value is orphaned due to a bad key
				if badKeyArg != nil {
					pass.ReportRangef(arg, "%s arg %q should be a string (previous arg %q cannot be a key)",
						shortName(fn), analysisinternal.Format(pass.Fset, arg), analysisinternal.Format(pass.Fset, badKeyArg))
					return
				}
				// Normal value, anything can appear here
				pos = key

			case unknown:
				if t != stringType && !types.IsInterface(t) {
					// This argument is definitely not a key, so treat it as a value
					pos = key
				}
				// If it could still be a string, remain in unknown state
			}
		}
		if pos == value {
			if badKeyArg == nil {
				pass.ReportRangef(call, "call to %s missing a final value", shortName(fn))
			} else {
				pass.ReportRangef(call, "call to %s has a missing or misplaced value", shortName(fn))
			}
		}
	})
	return nil, nil
}

// shortName returns a name for the function that is shorter than FullName.
func shortName(fn *types.Func) string {
	var r string
	if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
		if _, named := typesinternal.ReceiverNamed(recv); named != nil {
			r = named.Obj().Name()
		} else {
			r = recv.Type().String() // anon struct/interface
		}
		r += "."
	}
	return fmt.Sprintf("%s.%s%s", fn.Pkg().Path(), r, fn.Name())
}

// kvFuncSkipArgs checks if fn is a logging function that takes ...any for key-value pairs.
func kvFuncSkipArgs(fn *types.Func) (int, bool) {
	pkg := fn.Pkg()
	if pkg == nil {
		return 0, false
	}
	// Accept both the real package path and test module paths
	pkgPath := pkg.Path()
	if pkgPath != "github.com/happy-sdk/happy/pkg/logging" && pkgPath != "happy/pkg/logging" {
		return 0, false
	}

	var recvName string
	if recv := fn.Type().(*types.Signature).Recv(); recv != nil {
		_, named := typesinternal.ReceiverNamed(recv)
		if named == nil {
			return 0, false // anon struct/interface
		}
		recvName = named.Obj().Name()
	}
	skip, ok := kvFuncs[recvName][fn.Name()]
	return skip, ok
}

// kvFuncs defines functions/methods in github.com/happy-sdk/happy/pkg/logging that take ...any for key-value pairs.
var kvFuncs = map[string]map[string]int{
	"": {
		"Debug":        1,
		"Info":         1,
		"Warn":         1,
		"Error":        1,
		"DebugContext": 2,
		"InfoContext":  2,
		"WarnContext":  2,
		"ErrorContext": 2,
		"Log":          3,
		"Logs":         3,
		"Group":        1,
	},
	"Logger": {
		"Debug":        1,
		"Info":         1,
		"Warn":         1,
		"Error":        1,
		"DebugContext": 2,
		"InfoContext":  2,
		"WarnContext":  2,
		"ErrorContext": 2,
		"Log":          3,
		"With":         0,
	},
	"Record": {
		"Add": 0,
	},
}

func isMethodExpr(info *types.Info, c *ast.CallExpr) bool {
	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	sel := info.Selections[s]
	return sel != nil && sel.Kind() == types.MethodExpr
}
