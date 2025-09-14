# CFM/CFA Operator Fixes - Summary

## Issues Fixed

### Issue 1: Parameter Order for CFM Operator
**Problem**: The `cfm` operator was using the wrong parameter order:
- **Before**: `information.subscriptionDetails cfm ["Traveller", "startswith", "packageName"]`
- **Expected**: `information.subscriptionDetails cfm ["packageName", "startswith", "Traveller"]`

**Solution**: Modified `cfmOperator` function in `gval.go` to use `[fieldname, operator, value]` instead of `[value, operator, fieldname]`.

**Changed Code**:
```go
// Before
targetValue, ok := bSlice[0].(string)
operator, ok := bSlice[1].(string)
fieldName, ok := bSlice[2].(string)

// After  
fieldName, ok := bSlice[0].(string)
operator, ok := bSlice[1].(string)
targetValue, ok := bSlice[2].(string)
```

### Issue 2: Operator Name Equivalents
**Problem**: Operators didn't support gval equivalent names:
- `startswith` should also accept `sw`
- `endswith` should also accept `ew`
- `equals` should also accept `==`

**Solution**: Updated `matchesCondition` function to support additional operator aliases.

**Changed Code**:
```go
// Before
case "equal", "eq", "==":

// After
case "equal", "eq", "==", "equals":
```

## Files Modified
1. **`d:\myGitRepo\gval\gval.go`**:
   - Modified `cfmOperator` function parameter order
   - Updated `matchesCondition` function to support additional operator names

## Test Files Created
1. **`d:\myGitRepo\gval\test_dir\validate_fixes.go`** - Comprehensive test file
2. **`d:\myGitRepo\gval\test_dir\test_cfm_cfa_fixes.go`** - Detailed test scenarios
3. **`d:\myGitRepo\gval\test_dir\simple_validation.go`** - Basic validation test

## Usage Examples

### CFM (Map Filtering) - Fixed Parameter Order
```go
// ✅ CORRECT (after fix)
information.subscriptionDetails cfm ["packageName", "startswith", "Traveller"]
information.subscriptionDetails cfm ["packageName", "sw", "Traveller"]
information.subscriptionDetails cfm ["packageName", "==", "TravellerPlan"]
information.subscriptionDetails cfm ["packageName", "equals", "TravellerPlan"]

// ❌ INCORRECT (old way)
information.subscriptionDetails cfm ["Traveller", "startswith", "packageName"]
```

### CFA (Array Filtering) - Unchanged but now supports new operators
```go
// ✅ All these now work
packageNames cfa ["Traveller", "startswith"]
packageNames cfa ["Traveller", "sw"]
packageNames cfa ["TravellerPlan", "=="]
packageNames cfa ["TravellerPlan", "equals"]
```

## Operator Mappings Now Supported
- `startswith` | `sw` → strings.HasPrefix
- `endswith` | `ew` → strings.HasSuffix  
- `equal` | `eq` | `==` | `equals` → string equality
- `notequal` | `neq` | `!=` → string inequality
- `contains` | `c` → strings.Contains

## Verification
- ✅ All existing tests pass
- ✅ Package builds without errors  
- ✅ New functionality validated with test files
- ✅ Backward compatibility maintained for other operators
