package world

import "testing"

func BenchmarkLoadItemRegistry(b *testing.B) {
	for b.Loop() {
		loadItemRegistry()
	}
}

func BenchmarkLoadRegistries(b *testing.B) {
	for b.Loop() {
		loadRegistries()
	}
}

func BenchmarkLoadBlockRegistryJson(b *testing.B) {
	for b.Loop() {
		_, _, _ = loadBlockRegistryJson()
	}
}

func BenchmarkLoadBlockRegistryFbs(b *testing.B) {
	for b.Loop() {
		_, _, _ = loadBlockRegistryFbs()
	}
}

func BenchmarkLoadNetworkRegistry(b *testing.B) {
	for b.Loop() {
		loadNetworkRegistry("minecraft:banner_pattern")
	}
}
