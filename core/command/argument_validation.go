package command

import (
	"fmt"
	"math"
	"strings"
)

func validateArgument(argument Argument) error {
	if argument.Type != ArgEnum && len(argument.Enum) != 0 {
		return fmt.Errorf("enum values require an enum argument")
	}
	if argument.Type == ArgEnum {
		if len(argument.Enum) == 0 {
			return fmt.Errorf("enum has no values")
		}
		seen := make(map[string]struct{}, len(argument.Enum))
		for _, value := range argument.Enum {
			if _, duplicate := seen[value]; duplicate {
				return fmt.Errorf("enum has duplicate value %q", value)
			}
			seen[value] = struct{}{}
		}
	}
	if argument.Type == ArgCustom {
		if strings.TrimSpace(argument.CustomType) == "" {
			return fmt.Errorf("custom argument has no type id")
		}
	} else if argument.CustomType != "" {
		return fmt.Errorf("custom type id requires a custom argument")
	}
	if argument.Type != ArgInteger && (argument.IntegerMin != nil || argument.IntegerMax != nil) {
		return fmt.Errorf("integer range requires an integer argument")
	}
	if argument.IntegerMin != nil && argument.IntegerMax != nil && *argument.IntegerMin > *argument.IntegerMax {
		return fmt.Errorf("integer minimum exceeds maximum")
	}
	if argument.Type != ArgDecimal && (argument.DecimalMin != nil || argument.DecimalMax != nil) {
		return fmt.Errorf("decimal range requires a decimal argument")
	}
	if argument.DecimalMin != nil && math.IsNaN(*argument.DecimalMin) ||
		argument.DecimalMax != nil && math.IsNaN(*argument.DecimalMax) {
		return fmt.Errorf("decimal range contains NaN")
	}
	if argument.DecimalMin != nil && argument.DecimalMax != nil && *argument.DecimalMin > *argument.DecimalMax {
		return fmt.Errorf("decimal minimum exceeds maximum")
	}
	return nil
}
