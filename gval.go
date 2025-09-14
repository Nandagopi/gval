// Package gval provides a generic expression language.
// All functions, infix and prefix operators can be replaced by composing languages into a new one.
//
// The package contains concrete expression languages for common application in text, arithmetic, decimal arithmetic, propositional logic and so on.
// They can be used as basis for a custom expression language or to evaluate expressions directly.
package gval

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"text/scanner"
	"time"

	"github.com/shopspring/decimal"
)

// Evaluate given parameter with given expression in gval full language
func Evaluate(expression string, parameter interface{}, opts ...Language) (interface{}, error) {
	return EvaluateWithContext(context.Background(), expression, parameter, opts...)
}

// Evaluate given parameter with given expression in gval full language using a context
func EvaluateWithContext(c context.Context, expression string, parameter interface{}, opts ...Language) (interface{}, error) {
	l := full
	if len(opts) > 0 {
		l = NewLanguage(append([]Language{l}, opts...)...)
	}
	return l.EvaluateWithContext(c, expression, parameter)
}

// Full is the union of Arithmetic, Bitmask, Text, PropositionalLogic, TernaryOperator, and Json
//
//	Operator in: a in b is true iff value a is an element of array b
//	Operator ??: a ?? b returns a if a is not false or nil, otherwise n
//
// Function Date: Date(a) parses string a. a must match RFC3339, ISO8601, ruby date, or unix date
func Full(extensions ...Language) Language {
	if len(extensions) == 0 {
		return full
	}
	return NewLanguage(append([]Language{full}, extensions...)...)
}

// TernaryOperator contains following Operator
//
//	?: a ? b : c returns b if bool a is true, otherwise b
func TernaryOperator() Language {
	return ternaryOperator
}

// Arithmetic contains base, plus(+), minus(-), divide(/), power(**), negative(-)
// and numerical order (<=,<,>,>=)
//
// Arithmetic operators expect float64 operands.
// Called with unfitting input, they try to convert the input to float64.
// They can parse strings and convert any type of int or float.
func Arithmetic() Language {
	return arithmetic
}

// DecimalArithmetic contains base, plus(+), minus(-), divide(/), power(**), negative(-)
// and numerical order (<=,<,>,>=)
//
// DecimalArithmetic operators expect decimal.Decimal operands (github.com/shopspring/decimal)
// and are used to calculate money/decimal rather than floating point calculations.
// Called with unfitting input, they try to convert the input to decimal.Decimal.
// They can parse strings and convert any type of int or float.
func DecimalArithmetic() Language {
	return decimalArithmetic
}

// Bitmask contains base, bitwise and(&), bitwise or(|) and bitwise not(^).
//
// Bitmask operators expect float64 operands.
// Called with unfitting input they try to convert the input to float64.
// They can parse strings and convert any type of int or float.
func Bitmask() Language {
	return bitmask
}

// Text contains base, lexical order on strings (<=,<,>,>=),
// regex match (=~) and regex not match (!~)
func Text() Language {
	return text
}

// PropositionalLogic contains base, not(!), and (&&), or (||) and Base.
//
// Propositional operator expect bool operands.
// Called with unfitting input they try to convert the input to bool.
// Numbers other than 0 and the strings "TRUE" and "true" are interpreted as true.
// 0 and the strings "FALSE" and "false" are interpreted as false.
func PropositionalLogic() Language {
	return propositionalLogic
}

// JSON contains json objects ({string:expression,...})
// and json arrays ([expression, ...])
func JSON() Language {
	return ljson
}

// Parentheses contains support for parentheses.
func Parentheses() Language {
	return parentheses
}

// Ident contains support for variables and functions.
func Ident() Language {
	return ident
}

// Base contains equal (==) and not equal (!=), perentheses and general support for variables, constants and functions
// It contains true, false, (floating point) number, string  ("" or ") and char (") constants
func Base() Language {
	return base
}

// cfaOperator handles custom filtering for arrays/slices
// Parameters: [value, operator] where operator can be "equal", "startswith", "endswith", "contains", "notequal"
// Returns: true if match found and slice was modified in-place, false if no match found
func cfaOperator(a, b interface{}) (interface{}, error) {
	// b must be []interface{} with at least 2 elements: [value, operator]
	bSlice, ok := b.([]interface{})
	if !ok || len(bSlice) < 2 {
		return false, nil
	}
	
	targetValue, ok := bSlice[0].(string)
	if !ok {
		return false, nil
	}
	
	operator, ok := bSlice[1].(string)
	if !ok {
		return false, nil
	}

	// Handle [][]interface{} (slice of slices)
	if sliceOfSlices, ok := a.([][]interface{}); ok {
		if len(sliceOfSlices) == 0 {
			return false, nil
		}
		
		for i, elem := range sliceOfSlices {
			// Check if any element in the slice matches based on operator
			for _, val := range elem {
				if strVal, ok := val.(string); ok {
					if matchesCondition(strVal, targetValue, operator) {
						// Swap with first element (modifies original slice in-place)
						sliceOfSlices[0], sliceOfSlices[i] = sliceOfSlices[i], sliceOfSlices[0]
						return true, nil
					}
				}
			}
		}
		return false, nil
	}

	// Handle []interface{} (slice of individual values)
	if slice, ok := a.([]interface{}); ok {
		if len(slice) == 0 {
			return false, nil
		}
		
		for i, val := range slice {
			if strVal, ok := val.(string); ok {
				if matchesCondition(strVal, targetValue, operator) {
					// Swap with first element (modifies original slice in-place)
					slice[0], slice[i] = slice[i], slice[0]
					return true, nil
				}
			}
		}
		return false, nil
	}

	return false, nil
}

// cfmOperator handles custom filtering for maps
// Parameters: [fieldname, operator, value] where operator can be "equal", "startswith", "endswith", "contains", "notequal"
// Returns: true if match found and slice was modified in-place, false if no match found
func cfmOperator(a, b interface{}) (interface{}, error) {
	// b must be []interface{} with exactly 3 elements: [fieldname, operator, value]
	bSlice, ok := b.([]interface{})
	if !ok || len(bSlice) < 3 {
		return false, nil
	}
	
	fieldName, ok := bSlice[0].(string)
	if !ok {
		return false, nil
	}
	
	operator, ok := bSlice[1].(string)
	if !ok {
		return false, nil
	}
	
	targetValue, ok := bSlice[2].(string)
	if !ok {
		return false, nil
	}

	// Handle []map[string]interface{} (slice of maps)
	if sliceOfMaps, ok := a.([]map[string]interface{}); ok {
		if len(sliceOfMaps) == 0 {
			return false, nil
		}
		
		for i, m := range sliceOfMaps {
			if val, exists := m[fieldName]; exists {
				if strVal, ok := val.(string); ok {
					if matchesCondition(strVal, targetValue, operator) {
						// Swap with first map (modifies original slice in-place)
						sliceOfMaps[0], sliceOfMaps[i] = sliceOfMaps[i], sliceOfMaps[0]
						return true, nil
					}
				}
			}
		}
		return false, nil
	}

	// Handle []interface{} where each element could be a map
	if slice, ok := a.([]interface{}); ok {
		if len(slice) == 0 {
			return false, nil
		}
		
		for i, item := range slice {
			if m, ok := item.(map[string]interface{}); ok {
				if val, exists := m[fieldName]; exists {
					if strVal, ok := val.(string); ok {
						if matchesCondition(strVal, targetValue, operator) {
							// Swap with first element (modifies original slice in-place)
							slice[0], slice[i] = slice[i], slice[0]
							return true, nil
						}
					}
				}
			}
		}
		return false, nil
	}

	return false, nil
}

// matchesCondition checks if value matches target based on the operator
func matchesCondition(value, target, operator string) bool {
	switch operator {
	case "equal", "eq", "==":
		return value == target
	case "notequal", "ne", "!=":
		return value != target
	case "startswith", "sw":
		return strings.HasPrefix(value, target)
	case "endswith", "ew":
		return strings.HasSuffix(value, target)
	case "contains", "co":
		return strings.Contains(value, target)
	default:
		return value == target // default to equal
	}
}

var full = NewLanguage(arithmetic, bitmask, text, propositionalLogic, ljson,

	InfixOperator("in", inArray),

	InfixShortCircuit("??", func(a interface{}) (interface{}, bool) {
		v := reflect.ValueOf(a)
		return a, a != nil && !v.IsZero()
	}),
	InfixOperator("??", func(a, b interface{}) (interface{}, error) {
		if v := reflect.ValueOf(a); a == nil || v.IsZero() {
			return b, nil
		}
		return a, nil
	}),

	// Custom filter operators
	InfixOperator("cfa", cfaOperator),
	InfixOperator("cfm", cfmOperator),

	ternaryOperator,

	Function("date", func(arguments ...interface{}) (interface{}, error) {
		if len(arguments) != 1 {
			return nil, fmt.Errorf("date() expects exactly one string argument")
		}
		s, ok := arguments[0].(string)
		if !ok {
			return nil, fmt.Errorf("date() expects exactly one string argument")
		}
		for _, format := range [...]string{
			time.ANSIC,
			time.UnixDate,
			time.RubyDate,
			time.Kitchen,
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02",                         // RFC 3339
			"2006-01-02 15:04",                   // RFC 3339 with minutes
			"2006-01-02 15:04:05",                // RFC 3339 with seconds
			"2006-01-02 15:04:05-07:00",          // RFC 3339 with seconds and timezone
			"2006-01-02T15Z0700",                 // ISO8601 with hour
			"2006-01-02T15:04Z0700",              // ISO8601 with minutes
			"2006-01-02T15:04:05Z0700",           // ISO8601 with seconds
			"2006-01-02T15:04:05.999999999Z0700", // ISO8601 with nanoseconds
		} {
			ret, err := time.ParseInLocation(format, s, time.Local)
			if err == nil {
				return ret, nil
			}
		}
		return nil, fmt.Errorf("date() could not parse %s", s)
	}),
)

var ternaryOperator = PostfixOperator("?", parseIf)

var ljson = NewLanguage(
	PrefixExtension('[', parseJSONArray),
	PrefixExtension('{', parseJSONObject),
)

var arithmetic = NewLanguage(
	InfixNumberOperator("+", func(a, b float64) (interface{}, error) { return a + b, nil }),
	InfixNumberOperator("-", func(a, b float64) (interface{}, error) { return a - b, nil }),
	InfixNumberOperator("*", func(a, b float64) (interface{}, error) { return a * b, nil }),
	InfixNumberOperator("/", func(a, b float64) (interface{}, error) { return a / b, nil }),
	InfixNumberOperator("%", func(a, b float64) (interface{}, error) { return math.Mod(a, b), nil }),
	InfixNumberOperator("**", func(a, b float64) (interface{}, error) { return math.Pow(a, b), nil }),

	InfixNumberOperator(">", func(a, b float64) (interface{}, error) { return a > b, nil }),
	InfixNumberOperator(">=", func(a, b float64) (interface{}, error) { return a >= b, nil }),
	InfixNumberOperator("<", func(a, b float64) (interface{}, error) { return a < b, nil }),
	InfixNumberOperator("<=", func(a, b float64) (interface{}, error) { return a <= b, nil }),

	InfixNumberOperator("==", func(a, b float64) (interface{}, error) { return a == b, nil }),
	InfixNumberOperator("!=", func(a, b float64) (interface{}, error) { return a != b, nil }),

	base,
)

var decimalArithmetic = NewLanguage(
	InfixDecimalOperator("+", func(a, b decimal.Decimal) (interface{}, error) { return a.Add(b), nil }),
	InfixDecimalOperator("-", func(a, b decimal.Decimal) (interface{}, error) { return a.Sub(b), nil }),
	InfixDecimalOperator("*", func(a, b decimal.Decimal) (interface{}, error) { return a.Mul(b), nil }),
	InfixDecimalOperator("/", func(a, b decimal.Decimal) (interface{}, error) { return a.Div(b), nil }),
	InfixDecimalOperator("%", func(a, b decimal.Decimal) (interface{}, error) { return a.Mod(b), nil }),
	InfixDecimalOperator("**", func(a, b decimal.Decimal) (interface{}, error) { return a.Pow(b), nil }),

	InfixDecimalOperator(">", func(a, b decimal.Decimal) (interface{}, error) { return a.GreaterThan(b), nil }),
	InfixDecimalOperator(">=", func(a, b decimal.Decimal) (interface{}, error) { return a.GreaterThanOrEqual(b), nil }),
	InfixDecimalOperator("<", func(a, b decimal.Decimal) (interface{}, error) { return a.LessThan(b), nil }),
	InfixDecimalOperator("<=", func(a, b decimal.Decimal) (interface{}, error) { return a.LessThanOrEqual(b), nil }),

	InfixDecimalOperator("==", func(a, b decimal.Decimal) (interface{}, error) { return a.Equal(b), nil }),
	InfixDecimalOperator("!=", func(a, b decimal.Decimal) (interface{}, error) { return !a.Equal(b), nil }),
	base,
	//Base is before these overrides so that the Base options are overridden
	PrefixExtension(scanner.Int, parseDecimal),
	PrefixExtension(scanner.Float, parseDecimal),
	PrefixOperator("-", func(c context.Context, v interface{}) (interface{}, error) {
		i, ok := convertToFloat(v)
		if !ok {
			return nil, fmt.Errorf("unexpected %v(%T) expected number", v, v)
		}
		return decimal.NewFromFloat(i).Neg(), nil
	}),
)

var bitmask = NewLanguage(
	InfixNumberOperator("^", func(a, b float64) (interface{}, error) { return float64(int64(a) ^ int64(b)), nil }),
	InfixNumberOperator("&", func(a, b float64) (interface{}, error) { return float64(int64(a) & int64(b)), nil }),
	InfixNumberOperator("|", func(a, b float64) (interface{}, error) { return float64(int64(a) | int64(b)), nil }),
	InfixNumberOperator("<<", func(a, b float64) (interface{}, error) { return float64(int64(a) << uint64(b)), nil }),
	InfixNumberOperator(">>", func(a, b float64) (interface{}, error) { return float64(int64(a) >> uint64(b)), nil }),

	PrefixOperator("~", func(c context.Context, v interface{}) (interface{}, error) {
		i, ok := convertToFloat(v)
		if !ok {
			return nil, fmt.Errorf("unexpected %T expected number", v)
		}
		return float64(^int64(i)), nil
	}),
)

var text = NewLanguage(
	InfixTextOperator("+", func(a, b string) (interface{}, error) { return fmt.Sprintf("%v%v", a, b), nil }),

	InfixTextOperator("<", func(a, b string) (interface{}, error) { return a < b, nil }),
	InfixTextOperator("<=", func(a, b string) (interface{}, error) { return a <= b, nil }),
	InfixTextOperator(">", func(a, b string) (interface{}, error) { return a > b, nil }),
	InfixTextOperator(">=", func(a, b string) (interface{}, error) { return a >= b, nil }),
	InfixTextOperator("sw", startsWithOp),
	InfixTextOperator("co", containsOp),
	InfixTextOperator("ew", endsWithOp),
	InfixTextOperator("mw", matchOp),

	InfixEvalOperator("=~", regEx),
	InfixEvalOperator("!~", notRegEx),
	base,
)

var propositionalLogic = NewLanguage(
	PrefixOperator("!", func(c context.Context, v interface{}) (interface{}, error) {
		b, ok := convertToBool(v)
		if !ok {
			return nil, fmt.Errorf("unexpected %T expected bool", v)
		}
		return !b, nil
	}),

	InfixShortCircuit("&&", func(a interface{}) (interface{}, bool) { return false, a == false }),
	InfixBoolOperator("&&", func(a, b bool) (interface{}, error) { return a && b, nil }),
	InfixShortCircuit("||", func(a interface{}) (interface{}, bool) { return true, a == true }),
	InfixBoolOperator("||", func(a, b bool) (interface{}, error) { return a || b, nil }),

	InfixBoolOperator("==", func(a, b bool) (interface{}, error) { return a == b, nil }),
	InfixBoolOperator("!=", func(a, b bool) (interface{}, error) { return a != b, nil }),

	base,
)

var parentheses = NewLanguage(
	PrefixExtension('(', parseParentheses),
)

var ident = NewLanguage(
	PrefixMetaPrefix(scanner.Ident, parseIdent),
)

var base = NewLanguage(
	PrefixExtension(scanner.Int, parseNumber),
	PrefixExtension(scanner.Float, parseNumber),
	PrefixOperator("-", func(c context.Context, v interface{}) (interface{}, error) {
		i, ok := convertToFloat(v)
		if !ok {
			return nil, fmt.Errorf("unexpected %v(%T) expected number", v, v)
		}
		return -i, nil
	}),

	PrefixExtension(scanner.String, parseString),
	PrefixExtension(scanner.Char, parseString),
	PrefixExtension(scanner.RawString, parseString),

	Constant("true", true),
	Constant("false", false),
	Constant("nil", nil),

	InfixOperator("==", func(a, b interface{}) (interface{}, error) { 
		// Handle nil comparisons correctly
		if a == nil && b == nil {
			return true, nil
		}
		if a == nil || b == nil {
			return false, nil
		}
		return reflect.DeepEqual(a, b), nil 
	}),
	InfixOperator("!=", func(a, b interface{}) (interface{}, error) { 
		// Handle nil comparisons correctly
		if a == nil && b == nil {
			return false, nil
		}
		if a == nil || b == nil {
			return true, nil
		}
		return !reflect.DeepEqual(a, b), nil 
	}),
	parentheses,

	Precedence("??", 0),

	Precedence("||", 20),
	Precedence("&&", 21),

	Precedence("==", 40),
	Precedence("!=", 40),
	Precedence(">", 40),
	Precedence(">=", 40),
	Precedence("<", 40),
	Precedence("<=", 40),
	Precedence("=~", 40),
	Precedence("!~", 40),
	Precedence("in", 40),
	Precedence("sw", 40),
	Precedence("co", 40),
	Precedence("ew", 40),
	Precedence("mw", 40),
	Precedence("cfa", 40),
	Precedence("cfm", 40),

	Precedence("^", 60),
	Precedence("&", 60),
	Precedence("|", 60),

	Precedence("<<", 90),
	Precedence(">>", 90),

	Precedence("+", 120),
	Precedence("-", 120),

	Precedence("*", 150),
	Precedence("/", 150),
	Precedence("%", 150),

	Precedence("**", 200),

	ident,
)
