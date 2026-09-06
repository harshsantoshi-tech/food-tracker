package food

import "fmt"

// validUnits are the household/measurement units we accept from the LLM.
// Anything outside this set is rejected rather than silently accepted,
// since an unrecognized unit means we can't reliably convert to grams.
var validUnits = map[string]bool{
	"piece":   true,
	"serving": true,
	"bowl":    true,
	"katori":  true,
	"plate":   true,
	"glass":   true,
	"cup":     true,
	"spoon":   true,
	"gram":    true,
	"g":       true,
	"ml":      true,
}

// maxReasonableWeightG guards against an LLM hallucinating an absurd
// portion size (e.g. 50000g of paratha). No single food item in a
// single meal should reasonably exceed this.
const maxReasonableWeightG = 3000

// Validate checks a ParseResult against the constraints described in
// the product spec: required fields, numeric ranges, valid units, and
// internal consistency between needs_clarification and its fields.
func (r ParseResult) Validate() error {
	if !r.NeedsClarification && len(r.Items) == 0 {
		return fmt.Errorf("parse result has no items and does not request clarification")
	}

	if r.NeedsClarification && r.ClarificationQuestion == "" {
		return fmt.Errorf("needs_clarification is true but clarification_question is empty")
	}

	for i, item := range r.Items {
		if item.Name == "" {
			return fmt.Errorf("item %d: name is required", i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d (%s): quantity must be positive, got %v", i, item.Name, item.Quantity)
		}
		if !validUnits[item.Unit] {
			return fmt.Errorf("item %d (%s): unrecognized unit %q", i, item.Name, item.Unit)
		}
		if item.Confidence < 0 || item.Confidence > 1 {
			return fmt.Errorf("item %d (%s): confidence must be between 0 and 1, got %v", i, item.Name, item.Confidence)
		}
		if item.EstimatedWeightG != nil {
			if *item.EstimatedWeightG <= 0 {
				return fmt.Errorf("item %d (%s): estimated_weight_g must be positive, got %v", i, item.Name, *item.EstimatedWeightG)
			}
			if *item.EstimatedWeightG > maxReasonableWeightG {
				return fmt.Errorf("item %d (%s): estimated_weight_g %v exceeds max reasonable portion (%vg)", i, item.Name, *item.EstimatedWeightG, maxReasonableWeightG)
			}
		}
	}

	return nil
}