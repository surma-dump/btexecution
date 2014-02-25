package main

import (
	"reflect"
	"testing"
)

func TestCursor_All_String(t *testing.T) {
	data := []string{"a", "b", "c", "d"}
	ctr := 0
	cm := &CursorMock{
		MoreFunc: func() bool {
			return ctr < len(data)
		},
		NextFunc: func(v interface{}) error {
			typ := reflect.TypeOf(v)
			if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.String {
				t.Fatalf("Invalid type: %s", typ)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(data[ctr]))
			ctr++
			return nil
		},
	}

	var result []string
	if err := All(cm, &result); err != nil {
		t.Fatalf("Cursor iteration failed: %s", err)
	}

	if !reflect.DeepEqual(data, result) {
		t.Fatalf("Expected %#v, got %#v", data, result)
	}
}

func TestCursor_All_Struct(t *testing.T) {
	type someStruct struct {
		A string
		B int
	}
	data := []someStruct{{"A", 1}, {"B", 2}, {"C", 3}}
	ctr := 0
	cm := &CursorMock{
		MoreFunc: func() bool {
			return ctr < len(data)
		},
		NextFunc: func(v interface{}) error {
			typ := reflect.TypeOf(v)
			if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
				t.Fatalf("Invalid type: %s", typ)
			}
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(data[ctr]))
			ctr++
			return nil
		},
	}

	var result []someStruct
	if err := All(cm, &result); err != nil {
		t.Fatalf("Cursor iteration failed: %s", err)
	}

	if !reflect.DeepEqual(data, result) {
		t.Fatalf("Expected %#v, got %#v", data, result)
	}
}
