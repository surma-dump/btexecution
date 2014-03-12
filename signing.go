package main

// Signing JSONs with HMAC instead of PGP
// https://camlistore.googlesource.com/camlistore/+/master/doc/json-signing/json-signing.txt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
	"unicode"
)

func Sign(o interface{}, signer hash.Hash) ([]byte, error) {
	j, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	t := strings.TrimRightFunc(string(j), unicode.IsSpace)
	if t[len(t)-1] != '}' {
		return nil, fmt.Errorf("Only objects can be signed. Did not find closing brace.")
	}
	t = t[0 : len(t)-1] // Cut brace

	signer.Write([]byte(t))
	rawSig := signer.Sum(nil)

	c := t + `,"signature":"` + base64.StdEncoding.EncodeToString(rawSig) + `"}`
	return []byte(c), nil
}

func Verify(ba []byte, signer hash.Hash, v interface{}) error {
	bas := string(ba)
	idx := strings.LastIndex(bas, `,"signature":`)
	bp, bs := bas[0:idx], bas[idx:]
	bpj := bp + "}"
	bs = "{" + bs[1:]

	var sigObj map[string]interface{}
	if err := json.Unmarshal([]byte(bs), &sigObj); err != nil {
		return err
	}

	signature, ok := sigObj["signature"]
	if len(sigObj) != 1 || !ok {
		return fmt.Errorf("Unexpected data in signature")
	}

	signer.Write([]byte(bp))
	rawSig := signer.Sum(nil)

	if base64.StdEncoding.EncodeToString(rawSig) != signature {
		return fmt.Errorf("Invalid signature")
	}

	return json.Unmarshal([]byte(bpj), v)
}
