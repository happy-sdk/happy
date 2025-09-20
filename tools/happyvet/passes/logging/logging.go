// SPDX-License-Identifier: Apache-2.0
//
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
	var attrType types.Type // The type of log/slog.Attr
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
		// Look up log/slog.Attr type from the log/slog package.
		if attrType == nil {
			for _, pkg := range pass.Pkg.Imports() {
				if pkg.Path() == "log/slog" {
					if attr := pkg.Scope().Lookup("Attr"); attr != nil {
						attrType = attr.Type()
						break
					}
				}
			}
			if attrType == nil {
				// Fallback in case log/slog is not imported (unlikely in Happy SDK).
				return
			}
		}

		if isMethodExpr(pass.TypesInfo, call) {
			// Call is to a method value. Skip the first argument.
			skipArgs++
		}
		if len(call.Args) <= skipArgs {
			// Too few args; perhaps there are no k-v pairs.
			return
		}

		// Check this call.
		// The first position should hold a key or Attr.
		pos := key
		var unknownArg ast.Expr // nil or the last unknown argument
		for _, arg := range call.Args[skipArgs:] {
			t := pass.TypesInfo.Types[arg].Type
			switch pos {
			case key:
				// Expect a string or log/slog.Attr.
				switch {
				case t == stringType:
					pos = value
				case types.Identical(t, attrType):
					pos = key
				case types.IsInterface(t):
					// Handle interface types cautiously.
					if types.AssignableTo(stringType, t) {
						// Could be string or Attr; treat as unknown.
						pos = unknown
						continue
					} else if types.AssignableTo(attrType, t) {
						// Assume it’s an Attr.
						pos = key
						continue
					}
					// Definitely an error.
					fallthrough
				default:
					if unknownArg == nil {
						pass.ReportRangef(arg, "%s arg %q should be a string or a log/slog.Attr (possible missing key or value)",
							shortName(fn), analysisinternal.Format(pass.Fset, arg))
					} else {
						pass.ReportRangef(arg, "%s arg %q should probably be a string or a log/slog.Attr (previous arg %q cannot be a key)",
							shortName(fn), analysisinternal.Format(pass.Fset, arg), analysisinternal.Format(pass.Fset, unknownArg))
					}
					return
				}

			case value:
				// Anything can appear in this position.
				pos = key

			case unknown:
				unknownArg = arg
				if t != stringType && !types.Identical(t, attrType) && !types.IsInterface(t) {
					// This argument is definitely not a key.
					pos = key
				}
			}
		}
		if pos == value {
			if unknownArg == nil {
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
	return fmt.Sprintf("%s.%s%s", fn.Pkg().Name(), r, fn.Name())
}

// kvFuncSkipArgs checks if fn is a logging function that takes ...any for key-value pairs.
func kvFuncSkipArgs(fn *types.Func) (int, bool) {
	if pkg := fn.Pkg(); pkg == nil || pkg.Path() != "github.com/happy-sdk/happy/pkg/logging/xlogging" {
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

// kvFuncs defines functions/methods in github.com/happy-sdk/happy/pkg/logging/xlogging that take ...any for key-value pairs.
var kvFuncs = map[string]map[string]int{
	// "": {
	// 	"Debug":        1,
	// 	"Info":         1,
	// 	"Warn":         1,
	// 	"Error":        1,
	// 	"DebugContext": 2,
	// 	"InfoContext":  2,
	// 	"WarnContext":  2,
	// 	"ErrorContext": 2,
	// 	"Log":          3,
	// 	"Logs":         3,
	// 	"Group":        1,
	// },
	"Logger": {
		// "Debug":        1,
		// "Info":         1,
		// "Warn":         1,
		// "Error":        1,
		// "DebugContext": 2,
		// "InfoContext":  2,
		// "WarnContext":  2,
		// "ErrorContext": 2,
		"Log": 3,
		// "With":         0,
	},
	// "Record": {
	// 	"Add": 0,
	// },
}

func isMethodExpr(info *types.Info, c *ast.CallExpr) bool {
	s, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	sel := info.Selections[s]
	return sel != nil && sel.Kind() == types.MethodExpr
}
