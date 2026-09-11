package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func scalarMember(kind string) string {
	switch kind {
	case "bool":
		return "Bool"
	case "int64":
		return "Int64"
	case "double":
		return "Double"
	case "string":
		return "String"
	}
	panic("validated mutable field is not a scalar: " + kind)
}

func nativeCancellation(file *protogen.GeneratedFile, events []event) {
	file.P("func nativeCancellable(eventType string) bool {")
	file.P("switch eventType {")
	for _, declared := range events {
		if declared.Cancellable {
			file.P("case ", declared.ConstName(), ": return true")
		}
	}
	file.P("}")
	file.P("return false")
	file.P("}")
}

// The host accepts only the paths and kinds explicitly declared in the schema.
// No nested write can bypass an immutable player, block or permissions field.
func nativeMutationPaths(file *protogen.GeneratedFile, events []event) {
	file.P("func nativeMutationAllowed(eventType string, mutation abi.Mutation) bool {")
	file.P("if len(mutation.Path) != 1 { return false }")
	file.P("switch eventType {")
	for _, declared := range events {
		for _, f := range declared.Fields {
			if f.Mutable {
				file.P("case ", declared.ConstName(), ":")
				break
			}
		}
		for _, f := range declared.Fields {
			if f.Mutable {
				file.P(fmt.Sprintf("if mutation.Path[0] == %d { return mutation.Value.Kind == abi.Value%s }", f.Index, scalarMember(f.Kind)))
			}
		}
	}
	file.P("}")
	file.P("return false")
	file.P("}")
}

func sdkMutations(file *protogen.GeneratedFile, events []event) {
	file.P("// nativeMutations collects only fields the common schema permits writing.")
	file.P("func nativeMutations(event Event, before []abi.Value) []abi.Mutation {")
	file.P("var mutations []abi.Mutation")
	file.P("switch event := event.(type) {")
	for _, declared := range events {
		for _, f := range declared.Fields {
			if f.Mutable {
				file.P("case *", declared.SDKType(), ":")
				break
			}
		}
		for _, f := range declared.Fields {
			if f.Mutable {
				file.P(fmt.Sprintf("if value := abi.%s(event.%s); !abi.Equal(before[%d], value) {", scalarMember(f.Kind), f.GoName(), f.Index))
				file.P(fmt.Sprintf("mutations = append(mutations, abi.Mutation{Path: []uint32{%d}, Value: value})", f.Index))
				file.P("}")
			}
		}
	}
	file.P("}")
	file.P("return mutations")
	file.P("}")
}
