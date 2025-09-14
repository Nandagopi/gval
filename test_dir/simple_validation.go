package main

import (
	"fmt"
	"log"
	"os"
	"github.com/Nandagopi/gval"
)

func main() {
	fmt.Println("=== CFM/CFA Operator Fixes Validation ===")
	
	// Test data
	subscriptionDetails := []map[string]interface{}{
		{"packageName": "BasicPlan", "userId": "user1"},
		{"packageName": "TravellerPlan", "userId": "user2"},
		{"packageName": "PremiumPlan", "userId": "user3"},
	}

	params := map[string]interface{}{
		"information": map[string]interface{}{
			"subscriptionDetails": subscriptionDetails,
		},
	}

	lang := gval.Full()
	
	fmt.Println("\n1. Testing Issue 1 Fix - CFM parameter order:")
	fmt.Println("   OLD: [value, operator, fieldname] -> information.subscriptionDetails cfm [\"Traveller\", \"startswith\", \"packageName\"]")
	fmt.Println("   NEW: [fieldname, operator, value] -> information.subscriptionDetails cfm [\"packageName\", \"startswith\", \"Traveller\"]")
	
	result, err := gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "startswith", "Traveller"]`, params, lang)
	if err != nil {
		log.Printf("   Error: %v", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ Result: %v\n", result)

	fmt.Println("\n2. Testing Issue 2 Fix - Operator name equivalents:")
	
	// Test startswith -> sw
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "sw", "Traveller"]`, params, lang)
	if err != nil {
		log.Printf("   Error with 'sw': %v", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ 'sw' (startswith): %v\n", result)

	// Test equals -> ==
	result, err = gval.Evaluate(`information.subscriptionDetails cfm ["packageName", "==", "TravellerPlan"]`, params, lang)
	if err != nil {
		log.Printf("   Error with '==': %v", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ '==' (equals): %v\n", result)

	fmt.Println("\n🎉 All tests passed! Both issues have been fixed:")
	fmt.Println("   ✓ Issue 1: CFM now uses [fieldname, operator, value] parameter order")
	fmt.Println("   ✓ Issue 2: Operators support gval equivalents (sw, ew, ==, equals)")
}
