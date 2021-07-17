package bexp

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBalanced(t *testing.T) {
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   12,
		Pre:   "pre",
		Body:  "in{nest}",
		Post:  "post",
	}, Balanced("{", "}", "pre{in{nest}}post"))
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 8,
		End:   11,
		Pre:   "{{{{{{{{",
		Body:  "in",
		Post:  "post",
	}, Balanced("{", "}", "{{{{{{{{{in}post"))
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 8,
		End:   11,
		Pre:   "pre{body",
		Body:  "in",
		Post:  "post",
	}, Balanced("{", "}", "pre{body{in}post"))

	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 4,
		End:   13,
		Pre:   "pre}",
		Body:  "in{nest}",
		Post:  "post",
	}, Balanced("{", "}", "pre}{in{nest}}post"))
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   8,
		Pre:   "pre",
		Body:  "body",
		Post:  "between{body2}post",
	}, Balanced("{", "}", "pre{body}between{body2}post"))

	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   19,
		Pre:   "pre",
		Body:  "in<b>nest</b>",
		Post:  "post",
	}, Balanced("<b>", "</b>", "pre<b>in<b>nest</b></b>post"))

	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 7,
		End:   23,
		Pre:   "pre</b>",
		Body:  "in<b>nest</b>",
		Post:  "post",
	}, Balanced("<b>", "</b>", "pre</b><b>in<b>nest</b></b>post"))

	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   9,
		Pre:   "pre",
		Body:  "{in}",
		Post:  "post",
	}, Balanced("{{", "}}", "pre{{{in}}}post"))
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   8,
		Pre:   "pre",
		Body:  "in",
		Post:  "}post",
	}, Balanced("{{{", "}}", "pre{{{in}}}post"))
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 4,
		End:   10,
		Pre:   "pre{",
		Body:  "first",
		Post:  "in{second}post",
	}, Balanced("{", "}", "pre{{first}in{second}post"))

	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   4,
		Pre:   "pre",
		Body:  "",
		Post:  "post",
	}, Balanced("<?", "?>", "pre<?>post"))

	regStart := regexp.MustCompile(`\{`)
	regEnd := regexp.MustCompile(`\}`)
	assert.Equal(t, BalancedResult{}, Balanced(regStart, regEnd, "nope"))

	regStart = regexp.MustCompile(`\s+\{\s+`)
	regEnd = regexp.MustCompile(`\s+\}\s+`)
	assert.Equal(t, BalancedResult{
		Valid: true,
		Start: 3,
		End:   17,
		Pre:   "pre",
		Body:  "in{nest}",
		Post:  "post",
	}, Balanced(regStart, regEnd, "pre  {   in{nest}   }  post"))

	assert.Equal(t, BalancedResult{}, Balanced("{", "}", "nope"))
}
