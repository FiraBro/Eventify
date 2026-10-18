package services

import "testing"

func TestAvailabilityCacheKeyIncludesServiceAndDate(t *testing.T) {
	base := availabilityCacheKey("service-a", "2026-10-19")
	if base != availabilityCacheKey("service-a", "2026-10-19") {
		t.Fatal("equivalent availability inputs must share a key")
	}
	if base == availabilityCacheKey("service-b", "2026-10-19") {
		t.Fatal("service ID must affect availability key")
	}
	if base == availabilityCacheKey("service-a", "2026-10-20") {
		t.Fatal("date must affect availability key")
	}
	if base[:len("local-go:availability:service:v1:")] != "local-go:availability:service:v1:" {
		t.Fatal("availability key namespace missing")
	}
}

func TestSharedReadCacheKeysAreNamespaced(t *testing.T) {
	if servicesListCacheKey() == settingsCacheKey() {
		t.Fatal("unrelated resources must not share a cache key")
	}
	if servicesListCacheKey()[:len("local-go:services:list:v1:")] != "local-go:services:list:v1:" {
		t.Fatal("services namespace missing")
	}
	if settingsCacheKey()[:len("local-go:settings:singleton:v1:")] != "local-go:settings:singleton:v1:" {
		t.Fatal("settings namespace missing")
	}
}
