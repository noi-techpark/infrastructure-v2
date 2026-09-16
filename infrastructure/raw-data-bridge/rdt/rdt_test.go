// SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>
//
// SPDX-License-Identifier: CC0-1.0

package rdt

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidCollectionName(t *testing.T) {
	valid := []string{
		"enrichment",
		"parking",
		"parking.skidata",
		"rawdata.key",
		"meta.station_code",
		"a-b.c_d",
		"x",
	}
	for _, s := range valid {
		if !ValidCollectionName(s) {
			t.Errorf("ValidCollectionName(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",               // empty
		"$where",         // operator
		"rawdata.$key",   // operator in a path segment
		".leading",       // leading dot
		"trailing.",      // trailing dot leaves an empty segment
		"a..b",           // empty segment
		"a b",            // space
		"a\x00b",         // null byte
		"admin;drop",     // punctuation
		"{\"$ne\":null}", // json injection attempt
	}
	for _, s := range invalid {
		if ValidCollectionName(s) {
			t.Errorf("ValidCollectionName(%q) = true, want false", s)
		}
	}

	// length bound
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	if ValidCollectionName(string(long)) {
		t.Error("ValidCollectionName(200 chars) = true, want false")
	}
}

func TestValidFieldName(t *testing.T) {
	for _, s := range []string{"key", "station_code", "a-b", "k1"} {
		if !ValidFieldName(s) {
			t.Errorf("ValidFieldName(%q) = false, want true", s)
		}
	}
	// Keys are root-level only: a dotted path is rejected even though the same
	// string is a valid collection name.
	for _, s := range []string{"", "rawdata.key", "query.key", "$where", "a b", ".x", "x."} {
		if ValidFieldName(s) {
			t.Errorf("ValidFieldName(%q) = true, want false", s)
		}
	}
	if !ValidCollectionName("parking.skidata") {
		t.Error("dotted collection names must still be accepted")
	}
}

func TestDocumentMarshalStripsStorageFields(t *testing.T) {
	doc := Document{"_id": "x", "bsontimestamp": "y", "provider": "p"}
	b, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	s := string(b)
	if want := `{"provider":"p"}`; s != want {
		t.Errorf("MarshalJSON = %s, want %s", s, want)
	}
}

// A listed document has to be addressable: the storage spelling stays hidden,
// but the id comes back so a caller can build the `/urns/` permalink for it.
func TestDocumentMarshalExposesTheObjectId(t *testing.T) {
	id, err := primitive.ObjectIDFromHex("683ff7249237c6dd345cbf29")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Document{"_id": id, "bsontimestamp": "y", "provider": "p"}.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["id"] != id.Hex() {
		t.Errorf("id = %v, want %s", got["id"], id.Hex())
	}
	if _, ok := got["_id"]; ok {
		t.Error("the storage spelling `_id` leaked into the output")
	}
	if _, ok := got["bsontimestamp"]; ok {
		t.Error("bsontimestamp leaked into the output")
	}
}

// Dots separate a namespace, and MongoDB allows them in a collection name only.
func TestValidDatabaseNameRejectsDots(t *testing.T) {
	for _, s := range []string{"parking.skidata", "a.b", ".x", "x."} {
		if ValidDatabaseName(s) {
			t.Errorf("ValidDatabaseName(%q) = true; the driver would fail this as InvalidNamespace", s)
		}
		if s == "parking.skidata" && !ValidCollectionName(s) {
			t.Errorf("ValidCollectionName(%q) = false; dotted collections are legitimate", s)
		}
	}
	for _, s := range []string{"enrichment", "a-b", "x_1"} {
		if !ValidDatabaseName(s) {
			t.Errorf("ValidDatabaseName(%q) = false", s)
		}
	}
}
