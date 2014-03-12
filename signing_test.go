package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"reflect"
	"testing"
)

type someStruct struct {
	A int     `json:"a"`
	B string  `json:"b"`
	C float64 `json:"c"`
}

func TestSignVerify(t *testing.T) {
	in := someStruct{
		A: 4,
		B: "hai",
		C: 3.145,
	}

	data, err := Sign(in, hmac.New(sha512.New, []byte("someKey")))
	if err != nil {
		t.Fatalf("Signing failed: %s", err)
	}

	var out someStruct
	if err := Verify(data, hmac.New(sha512.New, []byte("someKey")), &out); err != nil {
		t.Fatalf("Verify failed: %s", err)
	}

	if !reflect.DeepEqual(in, out) {
		t.Fatalf("Objects differ: %#v vs %#v", in, out)
	}

}
