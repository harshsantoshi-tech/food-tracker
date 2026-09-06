package food

import "testing"

func floatPtr(f float64) *float64 { return &f }

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		result  ParseResult
		wantErr bool
	}{
		{
			name: "valid single item",
			result: ParseResult{
				Items: []Item{
					{Name: "aloo paratha", Quantity: 2, Unit: "piece", EstimatedWeightG: floatPtr(180), Confidence: 0.85},
				},
			},
			wantErr: false,
		},
		{
			name: "valid clarification request",
			result: ParseResult{
				Items: []Item{
					{Name: "butter chicken", Quantity: 1, Unit: "serving", Confidence: 0.55},
				},
				NeedsClarification:    true,
				ClarificationQuestion: "Approximately how much butter chicken did you eat?",
			},
			wantErr: false,
		},
		{
			name:    "no items and no clarification",
			result:  ParseResult{},
			wantErr: true,
		},
		{
			name: "clarification flag set but no question",
			result: ParseResult{
				Items:              []Item{{Name: "biryani", Quantity: 1, Unit: "plate", Confidence: 0.5}},
				NeedsClarification: true,
			},
			wantErr: true,
		},
		{
			name: "missing name",
			result: ParseResult{
				Items: []Item{{Quantity: 1, Unit: "piece", Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 0, Unit: "piece", Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "negative quantity",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: -2, Unit: "piece", Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "invalid unit",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 2, Unit: "kilogram", Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "confidence too high",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 2, Unit: "piece", Confidence: 1.5}},
			},
			wantErr: true,
		},
		{
			name: "confidence negative",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 2, Unit: "piece", Confidence: -0.1}},
			},
			wantErr: true,
		},
		{
			name: "weight zero",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 2, Unit: "piece", EstimatedWeightG: floatPtr(0), Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "weight absurdly large",
			result: ParseResult{
				Items: []Item{{Name: "roti", Quantity: 2, Unit: "piece", EstimatedWeightG: floatPtr(50000), Confidence: 0.8}},
			},
			wantErr: true,
		},
		{
			name: "nil weight is allowed (unknown portion)",
			result: ParseResult{
				Items: []Item{{Name: "butter chicken", Quantity: 1, Unit: "serving", Confidence: 0.55}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}