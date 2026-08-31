package gocraft

import (
	"fmt"

	abi "GoCraft/abi/v1"
)

func listOf(value abi.Value, length int, name string) ([]abi.Value, error) {
	if value.Kind != abi.ValueList || len(value.List) != length {
		return nil, fmt.Errorf("gocraft: %s must contain %d values", name, length)
	}
	return value.List, nil
}

func stringFrom(value abi.Value, name string) (string, error) {
	if value.Kind != abi.ValueString {
		return "", fmt.Errorf("gocraft: %s is not a string", name)
	}
	return value.String, nil
}

func permissionsFrom(value abi.Value) (map[string]bool, error) {
	if value.Kind != abi.ValueList {
		return nil, fmt.Errorf("gocraft: permissions are not a list")
	}
	permissions := make(map[string]bool, len(value.List))
	for _, entry := range value.List {
		pair, err := listOf(entry, 2, "permission")
		if err != nil {
			return nil, err
		}
		node, err := stringFrom(pair[0], "permission node")
		if err != nil {
			return nil, err
		}
		if pair[1].Kind != abi.ValueBool {
			return nil, fmt.Errorf("gocraft: permission %s is not boolean", node)
		}
		permissions[node] = pair[1].Bool
	}
	return permissions, nil
}

func propertiesFrom(value abi.Value) (map[string]string, error) {
	if value.Kind != abi.ValueList {
		return nil, fmt.Errorf("gocraft: block properties are not a list")
	}
	properties := make(map[string]string, len(value.List))
	for _, entry := range value.List {
		pair, err := listOf(entry, 2, "block property")
		if err != nil {
			return nil, err
		}
		key, err := stringFrom(pair[0], "block property name")
		if err != nil {
			return nil, err
		}
		property, err := stringFrom(pair[1], "block property value")
		if err != nil {
			return nil, err
		}
		properties[key] = property
	}
	return properties, nil
}
