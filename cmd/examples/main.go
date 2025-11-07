package main

import (
	"fmt"
	"os"

	"github.com/thiagohmm/integracaocron/examples"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cmd/examples/main.go [example_name]")
		fmt.Println("Available examples:")
		fmt.Println("  - promotion")
		fmt.Println("  - product_integration")
		fmt.Println("  - promotion_normalization")
		fmt.Println("  - complete_integration")
		return
	}

	exampleName := os.Args[1]

	switch exampleName {
	case "promotion":
		examples.RunPromotionExample()
	case "product_integration":
		examples.RunProductIntegrationService()
	case "promotion_normalization":
		examples.RunPromotionNormalizationService()
	case "complete_integration":
		examples.RunCompleteIntegrationExample()
	default:
		fmt.Printf("Unknown example: %s\n", exampleName)
		fmt.Println("Available examples: promotion, product_integration, promotion_normalization, complete_integration")
	}
}
