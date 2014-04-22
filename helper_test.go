package main

import (
	"net/url"
	"reflect"
	"testing"
)

func TestBuildQuery(t *testing.T) {
	type result struct {
		query string
		binds map[string]interface{}
	}
	table := map[string]result{
		"": {
			query: "FOR x IN collection LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"skip":  0,
				"limit": DEFAULT_LIMIT,
			},
		},
		"filter=tags:tag1": {
			query: "FOR x IN collection FILTER @fv0 IN x.@f0 LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"fv0":   "tag1",
				"f0":    "tags",
				"skip":  0,
				"limit": DEFAULT_LIMIT,
			},
		},
		"filter=tags:tag1&filter=tags:tag2": {
			query: "FOR x IN collection FILTER @fv0 IN x.@f0 FILTER @fv1 IN x.@f1 LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"fv0":   "tag1",
				"f0":    "tags",
				"fv1":   "tag2",
				"f1":    "tags",
				"skip":  0,
				"limit": DEFAULT_LIMIT,
			},
		},
		"filter=tags:tag1&filter=tags:tag2&exclude=tags:tag3": {
			query: "FOR x IN collection FILTER @fv0 IN x.@f0 FILTER @fv1 IN x.@f1 FILTER !(@ev0 IN x.@e0) LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"fv0":   "tag1",
				"f0":    "tags",
				"fv1":   "tag2",
				"f1":    "tags",
				"ev0":   "tag3",
				"e0":    "tags",
				"skip":  0,
				"limit": DEFAULT_LIMIT,
			},
		},
		"filter=from:1234&exclude=id:321": {
			query: "FOR x IN collection FILTER @fv0 == x.@f0 FILTER !(@ev0 == x.@e0) LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"fv0":   "1234",
				"f0":    "from",
				"ev0":   "321",
				"e0":    "id",
				"skip":  0,
				"limit": DEFAULT_LIMIT,
			},
		},
		"limit=30&skip=10": {
			query: "FOR x IN collection LIMIT @skip, @limit RETURN x",
			binds: map[string]interface{}{
				"skip":  10,
				"limit": 30,
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
			t.Errorf("Unexpected result for %#v: Expected\n%#v\nGot\n%#v", v, expectedResult, r)
		}
	}
}
