package gval

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// MissingFieldBehavior defines how missing fields should be handled
type MissingFieldBehavior int

const (
	// ErrorOnMissingField is the default behavior - throw an error
	ErrorOnMissingField MissingFieldBehavior = iota
	// FalseOnMissingField treats missing fields as false in boolean contexts
	FalseOnMissingField
	// NilOnMissingField treats missing fields as nil
	NilOnMissingField
)

// WithMissingFieldBehavior creates a language that handles missing fields according to the specified behavior
func WithMissingFieldBehavior(behavior MissingFieldBehavior) Language {
	return VariableSelector(func(path Evaluables) Evaluable {
		return func(c context.Context, v interface{}) (interface{}, error) {
			keys, err := path.EvalStrings(c, v)
			if err != nil {
				return nil, err
			}
			for i, k := range keys {
				switch o := v.(type) {
				case Selector:
					v, err = o.SelectGVal(c, k)
					if err != nil {
						return nil, fmt.Errorf("failed to select '%s' on %T: %w", k, o, err)
					}
					continue
				case map[interface{}]interface{}:
					if val, exists := o[k]; exists {
						v = val
					} else {
						return handleMissingField(behavior, keys[:i+1])
					}
					continue
				case map[string]interface{}:
					if val, exists := o[k]; exists {
						v = val
					} else {
						return handleMissingField(behavior, keys[:i+1])
					}
					continue
				case []interface{}:
					if idx, err := strconv.Atoi(k); err == nil && idx >= 0 && len(o) > idx {
						v = o[idx]
						continue
					}
					return handleMissingField(behavior, keys[:i+1])
				default:
					var ok bool
					v, ok = reflectSelect(k, o)
					if !ok {
						return handleMissingField(behavior, keys[:i+1])
					}
				}
			}
			return v, nil
		}
	})
}

func handleMissingField(behavior MissingFieldBehavior, keyPath []string) (interface{}, error) {
	switch behavior {
	case FalseOnMissingField:
		return false, nil
	case NilOnMissingField:
		return nil, nil
	default: // ErrorOnMissingField
		return nil, fmt.Errorf("unknown parameter %s", strings.Join(keyPath, "."))
	}
}

// TolerantFull creates a Full language that treats missing fields as false
// This is the recommended approach for handling missing fields in logical expressions
func TolerantFull() Language {
	return NewLanguage(
		// Core language features
		arithmetic, bitmask, text, propositionalLogic, ljson,
		
		// Additional operators
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
			// Date parsing logic would go here - simplified for brevity
			return s, nil
		}),
		
		// Missing field behavior - treat as false
		WithMissingFieldBehavior(FalseOnMissingField),
		
		// Enhanced comparison operators that handle boolean values gracefully
		enhancedComparisons(),
	)
}

// enhancedComparisons provides comparison operators that handle false values properly
func enhancedComparisons() Language {
	return NewLanguage(
		// Use InfixEvalOperator to completely override the operators
		InfixEvalOperator("==", func(a, b Evaluable) (Evaluable, error) {
			return func(c context.Context, v interface{}) (interface{}, error) {
				aVal, err := a(c, v)
				if err != nil {
					return nil, err
				}
				bVal, err := b(c, v)
				if err != nil {
					return nil, err
				}
				
				// Treat missing field (represented as false) equal to nil specifically
				if (aVal == nil && bVal == false) || (aVal == false && bVal == nil) {
					return true, nil
				}
				// nil == nil is true
				if aVal == nil && bVal == nil {
					return true, nil
				}
				// false == false should be true
				if aVal == false && bVal == false {
					return true, nil
				}
				// If one side is nil or false, check if they're the exact same value
				if aVal == nil || bVal == nil || aVal == false || bVal == false {
					// Only equal if they are the exact same value (handled above) or false/nil combo (handled above)
					return false, nil
				}
				
				// Handle numeric comparisons (int vs float64 etc.)
				if aFloat, aOk := convertToFloat(aVal); aOk {
					if bFloat, bOk := convertToFloat(bVal); bOk {
						return aFloat == bFloat, nil
					}
				}
				
				// Handle string comparisons
				if aStr, aOk := aVal.(string); aOk {
					if bStr, bOk := bVal.(string); bOk {
						return aStr == bStr, nil
					}
				}
				
				// Handle boolean comparisons
				if aBool, aOk := aVal.(bool); aOk {
					if bBool, bOk := bVal.(bool); bOk {
						return aBool == bBool, nil
					}
				}
				
				// Fall back to reflect.DeepEqual for complex types
				return reflect.DeepEqual(aVal, bVal), nil
			}, nil
		}),
		
		InfixEvalOperator("!=", func(a, b Evaluable) (Evaluable, error) {
			return func(c context.Context, v interface{}) (interface{}, error) {
				aVal, err := a(c, v)
				if err != nil {
					return nil, err
				}
				bVal, err := b(c, v)
				if err != nil {
					return nil, err
				}
				
				// false (missing) vs nil should be equal, so != is false
				if (aVal == nil && bVal == false) || (aVal == false && bVal == nil) {
					return false, nil
				}
				// nil != nil is false
				if aVal == nil && bVal == nil {
					return false, nil
				}
				// false != false is false
				if aVal == false && bVal == false {
					return false, nil
				}
				// If exactly one side is nil or false (missing), they are not equal
				if (aVal == nil && bVal != nil) || (bVal == nil && aVal != nil) || (aVal == false && bVal != false) || (bVal == false && aVal != false) {
					return true, nil
				}
				
				// Handle numeric comparisons (int vs float64 etc.)
				if aFloat, aOk := convertToFloat(aVal); aOk {
					if bFloat, bOk := convertToFloat(bVal); bOk {
						return aFloat != bFloat, nil
					}
				}
				
				// Handle string comparisons
				if aStr, aOk := aVal.(string); aOk {
					if bStr, bOk := bVal.(string); bOk {
						return aStr != bStr, nil
					}
				}
				
				// Handle boolean comparisons
				if aBool, aOk := aVal.(bool); aOk {
					if bBool, bOk := bVal.(bool); bOk {
						return aBool != bBool, nil
					}
				}
				
				// Fall back to reflect.DeepEqual for complex types
				return !reflect.DeepEqual(aVal, bVal), nil
			}, nil
		}),
		
		// Override comparison operators to handle false (from missing fields) properly
		InfixOperator(">", func(a, b interface{}) (interface{}, error) {
			// If either operand is false (from missing field), comparison is false
			if a == false || b == false {
				return false, nil
			}
			// Try numeric comparison
			if aFloat, aOk := convertToFloat(a); aOk {
				if bFloat, bOk := convertToFloat(b); bOk {
					return aFloat > bFloat, nil
				}
			}
			// Fall back to string comparison
			return fmt.Sprintf("%v", a) > fmt.Sprintf("%v", b), nil
		}),
		
		InfixOperator(">=", func(a, b interface{}) (interface{}, error) {
			if a == false || b == false {
				return false, nil
			}
			if aFloat, aOk := convertToFloat(a); aOk {
				if bFloat, bOk := convertToFloat(b); bOk {
					return aFloat >= bFloat, nil
				}
			}
			return fmt.Sprintf("%v", a) >= fmt.Sprintf("%v", b), nil
		}),
		
		InfixOperator("<", func(a, b interface{}) (interface{}, error) {
			if a == false || b == false {
				return false, nil
			}
			if aFloat, aOk := convertToFloat(a); aOk {
				if bFloat, bOk := convertToFloat(b); bOk {
					return aFloat < bFloat, nil
				}
			}
			return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b), nil
		}),
		
		InfixOperator("<=", func(a, b interface{}) (interface{}, error) {
			if a == false || b == false {
				return false, nil
			}
			if aFloat, aOk := convertToFloat(a); aOk {
				if bFloat, bOk := convertToFloat(b); bOk {
					return aFloat <= bFloat, nil
				}
			}
			return fmt.Sprintf("%v", a) <= fmt.Sprintf("%v", b), nil
		}),
	)
}
