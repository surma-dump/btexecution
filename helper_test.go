package main

import (
	"net/url"
	"reflect"
	"testing"
)

func TestBuildQuery(t *testing.T) {
	type result struct {
		query string
		binds map[string]string
	}
	table := map[string]result{
		"": {
			query: "FOR n IN collection RETURN n",
			binds: map[string]string{},
		},
		"tag=tag1": {
			query: "FOR n IN collection FILTER @f0 IN n.tags RETURN n",
			binds: map[string]string{
				"f0": "tag1",
			},
		},
		"tag=tag1&tag=tag2": {
			query: "FOR n IN collection FILTER @f0 IN n.tags && @f1 IN n.tags RETURN n",
			binds: map[string]string{
				"f0": "tag1",
				"f1": "tag2",
			},
		},
	}
	for qry, expectedResult := range table {
		v, err := url.ParseQuery(qry)
		if err != nil {
			t.Fatalf("Invalid query %s: %s", qry, err)
		}
		query, binds := buildQuery("collection", v)
		r := result{query, binds}
		if !reflect.DeepEqual(r, expectedResult) {
			t.Fatalf("Unexpected result for %#v: %#v", v, r)
		}
	}
}
