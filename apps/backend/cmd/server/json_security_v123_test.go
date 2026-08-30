package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeStrictJSONAcceptsSingleKnownObject(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"demo"}`))
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(req, &body); err != nil {
		t.Fatalf("decodeStrictJSON() error = %v", err)
	}
	if body.Name != "demo" {
		t.Fatalf("name = %q, want demo", body.Name)
	}
}

func TestDecodeStrictJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"demo","admin":true}`))
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(req, &body); err == nil {
		t.Fatal("decodeStrictJSON() accepted an unknown field")
	}
}

func TestDecodeStrictJSONRejectsTrailingJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"demo"}{"name":"second"}`))
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(req, &body); err == nil {
		t.Fatal("decodeStrictJSON() accepted multiple JSON values")
	}
}

func TestDecodeStrictJSONRejectsTrailingGarbage(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"demo"} trailing`))
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeStrictJSON(req, &body); err == nil {
		t.Fatal("decodeStrictJSON() accepted trailing garbage")
	}
}
