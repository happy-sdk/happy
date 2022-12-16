package bexp

import (
	"regexp"
	"testing"
)

type balancedTest struct {
	Result    BalancedResult
	A, B, Str string
}

func TestBalanced(t *testing.T) { //nolint: funlen
	regStart := regexp.MustCompile(`\{`)
	regEnd := regexp.MustCompile(`\}`)

	var balancedTests = []balancedTest{
		{BalancedResult{}, regStart.String(), regEnd.String(), "nope"},
		{BalancedResult{}, "{", "}", "nope"},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   12,
				Pre:   "pre",
				Body:  "in{nest}",
				Post:  "post",
			}, "{", "}", "pre{in{nest}}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 8,
				End:   11,
				Pre:   "{{{{{{{{",
				Body:  "in",
				Post:  "post",
			}, "{", "}", "{{{{{{{{{in}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 8,
				End:   11,
				Pre:   "pre{body",
				Body:  "in",
				Post:  "post",
			}, "{", "}", "pre{body{in}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 4,
				End:   13,
				Pre:   "pre}",
				Body:  "in{nest}",
				Post:  "post",
			}, "{", "}", "pre}{in{nest}}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   8,
				Pre:   "pre",
				Body:  "body",
				Post:  "between{body2}post",
			}, "{", "}", "pre{body}between{body2}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   19,
				Pre:   "pre",
				Body:  "in<b>nest</b>",
				Post:  "post",
			}, "<b>", "</b>", "pre<b>in<b>nest</b></b>post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 7,
				End:   23,
				Pre:   "pre</b>",
				Body:  "in<b>nest</b>",
				Post:  "post",
			}, "<b>", "</b>", "pre</b><b>in<b>nest</b></b>post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   9,
				Pre:   "pre",
				Body:  "{in}",
				Post:  "post",
			}, "{{", "}}", "pre{{{in}}}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   8,
				Pre:   "pre",
				Body:  "in",
				Post:  "}post",
			}, "{{{", "}}", "pre{{{in}}}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 4,
				End:   10,
				Pre:   "pre{",
				Body:  "first",
				Post:  "in{second}post",
			}, "{", "}", "pre{{first}in{second}post",
		},
		{
			BalancedResult{
				Valid: true,
				Start: 3,
				End:   4,
				Pre:   "pre",
				Body:  "",
				Post:  "post",
			}, "<?", "?>", "pre<?>post",
		},
	}

	for _, test := range balancedTests {
		res := Balanced(test.A, test.B, test.Str)
		if test.Result.Valid != res.Valid {
			t.Errorf(".Valid want(%t) got(%t)", test.Result.Valid, res.Valid)
		}
		if test.Result.Start != res.Start {
			t.Errorf(".Start want(%d) got(%d)", test.Result.Start, res.Start)
		}
		if test.Result.End != res.End {
			t.Errorf(".End want(%d) got(%d)", test.Result.End, res.End)
		}
		if test.Result.Pre != res.Pre {
			t.Errorf(".Pre want(%s) got(%s)", test.Result.Pre, res.Pre)
		}
		if test.Result.Body != res.Body {
			t.Errorf(".Body want(%s) got(%s)", test.Result.Body, res.Body)
		}
		if test.Result.Post != res.Post {
			t.Errorf(".Body want(%s) got(%s)", test.Result.Post, res.Post)
		}
	}
}
