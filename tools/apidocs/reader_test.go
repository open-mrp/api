package main

import (
	"reflect"
	"testing"

	"github.com/augno/api/tools/apidocs/testdata"
)

func TestDocReader_GetTypeDoc(t *testing.T) {
	t.Parallel()
	reader := NewDocReader()
	testType := reflect.TypeOf(testdata.TestStruct{})

	doc := reader.GetTypeDoc(testType)

	if doc.Doc != "TestStruct is a struct for testing DocReader." {
		t.Errorf("expected struct doc 'TestStruct is a struct for testing DocReader.', got '%s'", doc.Doc)
	}

	if doc.Fields["Name"] != "Name is a field doc." {
		t.Errorf("expected field doc 'Name is a field doc.', got '%s'", doc.Fields["Name"])
	}

	if doc.Fields["Age"] != "Age is another field doc." {
		t.Errorf("expected field doc 'Age is another field doc.', got '%s'", doc.Fields["Age"])
	}
}
