package validators

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.Map = minMaxValidator{}

type minMaxValidator struct {
	min, max int
}

func (v minMaxValidator) Description(_ context.Context) string {
	return fmt.Sprintf("List size must be between %d and %d", v.min, v.max)
}

func (v minMaxValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v minMaxValidator) ValidateMap(ctx context.Context, request validator.MapRequest, response *validator.MapResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value, _ := request.ConfigValue.ToMapValue(ctx)
	if len(value.Elements()) < v.min || len(value.Elements()) > v.max {

		response.Diagnostics.Append(validatordiag.InvalidAttributeValueLengthDiagnostic(
			request.Path,
			v.Description(ctx),
			fmt.Sprintf("%d", len(value.Elements())),
		))

		return
	}
}

func MinMax(min, max int) validator.Map {
	if min < 0 || min > max {
		return nil
	}

	return minMaxValidator{
		min: min,
		max: max,
	}
}
