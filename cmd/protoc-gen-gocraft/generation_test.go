package main

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	wire "github.com/GoCraft-MC/gocraft-abi/abi/v1/wire"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/pluginpb"
)

func schemaGenerator(t *testing.T) (*protogen.Plugin, []event) {
	t.Helper()
	req := &pluginpb.CodeGeneratorRequest{FileToGenerate: []string{"abi/v1/events.proto"}}
	seen := map[string]bool{}
	var include func(protoreflect.FileDescriptor)
	include = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		for i := 0; i < fd.Imports().Len(); i++ {
			include(fd.Imports().Get(i).FileDescriptor)
		}
		req.ProtoFile = append(req.ProtoFile, protodesc.ToFileDescriptorProto(fd))
	}
	include(wire.File_abi_v1_events_proto)
	plugin, err := (protogen.Options{}).New(req)
	if err != nil {
		t.Fatal(err)
	}
	events, err := collect(plugin.Files)
	if err != nil {
		t.Fatal(err)
	}
	return plugin, events
}

func TestGeneratedHostMatchesCommonSchema(t *testing.T) {
	plugin, events := schemaGenerator(t)
	if err := generateGo(plugin, events); err != nil {
		t.Fatal(err)
	}
	for _, file := range plugin.Response().File {
		stored, err := os.ReadFile("../../" + file.GetName())
		if err != nil || strings.ReplaceAll(string(stored), "\r\n", "\n") != file.GetContent() {
			t.Fatalf("regenerate %s from the pinned ABI: %v", file.GetName(), err)
		}
	}
}

func TestAllTargetsGenerateFromOneSchema(t *testing.T) {
	for _, generate := range []func(*protogen.Plugin, []event) error{generateSDK, generateJava} {
		plugin, events := schemaGenerator(t)
		if len(events) < 14 {
			t.Fatal("missing fundamental native events")
		}
		if err := generate(plugin, events); err != nil {
			t.Fatal(err)
		}
		for _, file := range plugin.Response().File {
			if file.GetName() == "events.gen.go" {
				if _, err := parser.ParseFile(token.NewFileSet(), file.GetName(), file.GetContent(), parser.AllErrors); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestGeneratedJavaDocumentsSnapshotOwnership(t *testing.T) {
	plugin, events := schemaGenerator(t)
	if err := generateJava(plugin, events); err != nil {
		t.Fatal(err)
	}
	for _, file := range plugin.Response().File {
		if strings.HasSuffix(file.GetName(), "/GeneratedEvents.java") {
			continue
		}
		content := file.GetContent()
		for _, documentation := range []string{"event-owned working copy", "caller's baseline stays unchanged", "@return the current value"} {
			if !strings.Contains(content, documentation) {
				t.Errorf("%s omits %q", file.GetName(), documentation)
			}
		}
		if strings.Contains(content, "public void set") && !strings.Contains(content, "host validates it") {
			t.Errorf("%s does not explain mutation validation", file.GetName())
		}
	}
}
