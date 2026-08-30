// Command protoc-gen-gocraft generates every runtime's view of the ABI from
// the schema that defines it.
//
// Writing an event by hand would mean writing it four times — Go, Java, Lua,
// docs — and watching all four drift by month three. One source, codegen for
// the rest, and both sides emitted in the same commit so a Lua event and a Java
// event cannot disagree about what index 3 holds.
//
// It is a protoc plugin, so buf drives it:
//
//	buf generate --template buf.gen.events.yaml
//
// The language is a parameter rather than separate binaries, because the two
// emitters share the model and the vocabulary table and would drift if they
// were separated:
//
//	--gocraft_opt=lang=go     writes core/plugin/events.gen.go
//	--gocraft_opt=lang=java   writes the event classes for gocraft-jvm
//
// Java is emitted from Go with text/template rather than by a protoc plugin
// written in Java: the repository and its CI are already Go, and adding a JVM
// to the generation path would undo what §16 arranges.
package main

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
)

func main() {
	var language string
	var flags flagSet
	flags.set("lang", &language)

	protogen.Options{ParamFunc: flags.parse}.Run(func(plugin *protogen.Plugin) error {
		events, err := collect(plugin.Files)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return fmt.Errorf("protoc-gen-gocraft: no message carries a gc.event option; " +
				"nothing to generate")
		}
		if err := checkUnique(events); err != nil {
			return err
		}
		switch language {
		case "go":
			return generateGo(plugin, events)
		case "java":
			return generateJava(plugin, events)
		case "":
			return fmt.Errorf("protoc-gen-gocraft: lang is required, one of go or java")
		default:
			return fmt.Errorf("protoc-gen-gocraft: unknown lang %q, want go or java", language)
		}
	})
}

// checkUnique refuses two events claiming the same name.
//
// The type is what a manifest subscribes to and what the bus routes on, so a
// duplicate would silently deliver one event to the other's subscribers.
func checkUnique(events []event) error {
	seen := make(map[string]string, len(events))
	for _, declared := range events {
		if first, duplicate := seen[declared.Type]; duplicate {
			return fmt.Errorf("protoc-gen-gocraft: %s and %s both declare the event type %q",
				first, declared.Message, declared.Type)
		}
		seen[declared.Type] = declared.Message
	}
	return nil
}

// flagSet is the smallest thing that reads key=value parameters. The standard
// flag package expects a command line, and protoc hands over one string.
type flagSet struct {
	targets map[string]*string
}

func (f *flagSet) set(name string, target *string) {
	if f.targets == nil {
		f.targets = make(map[string]*string)
	}
	f.targets[name] = target
}

func (f *flagSet) parse(name, value string) error {
	target, known := f.targets[name]
	if !known {
		fmt.Fprintf(os.Stderr, "protoc-gen-gocraft: ignoring unknown parameter %q\n", name)
		return nil
	}
	*target = value
	return nil
}
