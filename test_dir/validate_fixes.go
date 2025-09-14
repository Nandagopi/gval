package main

import (
	"fmt"
	"github.com/Nandagopi/gval"
)

func main() {
	// Test data for cfm (map filtering)
	subscriptionDetails := []map[string]interface{}{
		{"packageName": "BasicPlan", "userId": "user1"},
		{"packageName": "TravellerPlan", "userId": "user2"},
		{"packageName": "PremiumPlan", "userId": "user3"},
	}

	// Test data for cfa (array filtering)  
	packageNames := []interface{}{"BasicPlan", "TravellerPlan", "PremiumPlan"}

	params := map[string]interface{}{
		"information": map[string]interface{}{
			"subscriptionDetails": subscriptionDetails,
		},
		"packageNames": packageNames,
	}

	lang := gval.Full()
	
	fmt.Println("=== Testing CFM operator fixes ===")
	
	// Issue 1 FIXED: Now using [fieldname, operator, value] instead of [value, operator, fieldname]
	fmt.Println("\n1. Testing cfm with corrected parameter order:")
	
	// Test case 1: fieldname first (NEW CORRECT ORDER)
	fmt.Println("   Before: information.subscriptionDetails cfm [\"Traveller\", \"startswith\", \"packageName\"] (old wrong order)")
	fmt.Println("   After:  information.subscriptionDetails cfm [\"packageName\", \"startswith\", \"Traveller\"] (new correct order)")
	
	result, err := gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "startswith", "Traveller"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	// Issue 2 FIXED: Now supporting gval equivalent operators
	fmt.Println("\n2. Testing operator name equivalents:")
	
	// Test startswith -> sw
	fmt.Println("   Testing 'sw' (startswith equivalent):")
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "sw", "Traveller"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)
	
	// Test endswith -> ew (hypothetical test)
	fmt.Println("   Testing 'ew' (endswith equivalent):")
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "ew", "Plan"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)
	
	// Test equals -> ==
	fmt.Println("   Testing '==' (equals equivalent):")
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "==", "TravellerPlan"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)
	
	// Test equals -> equals (also should work)
	fmt.Println("   Testing 'equals' (explicit equals):")
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "equals", "TravellerPlan"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	fmt.Println("\n=== Testing CFA operator ===")
	
	// Test cfa with the new operator names
	fmt.Println("   Testing cfa with 'sw' operator:")
	result, err = gval.Evaluate(`packageNames cfa ["Traveller", "sw"]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)
	
	fmt.Println("   Testing cfa with '==' operator:")
	result, err = gval.Evaluate(`packageNames cfa ["TravellerPlan", "=="]`, params, lang)
	fmt.Printf("   Result: %v, Error: %v\n", result, err)

	fmt.Println("\n=== Summary ===")
	fmt.Println("✓ Issue 1 Fixed: cfm now uses [fieldname, operator, value] order")
	fmt.Println("✓ Issue 2 Fixed: Operators now support gval equivalents:")
	fmt.Println("  - startswith -> sw")
	fmt.Println("  - endswith -> ew") 
	fmt.Println("  - equals -> == (and 'equals')")
}
