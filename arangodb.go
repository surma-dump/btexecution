package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
)

type Connection interface {
	Database(s string) Database
	fmt.Stringer
}

type connection struct {
	Host string
}

func NewConnection(s string) (Connection, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("Only http(s) connections are supported")
	}

	return &connection{
		Host: u.String(),
	}, nil
}

func (c *connection) Database(s string) Database {
	return &database{
		Name: s,
		c:    c,
	}
}

func (c *connection) String() string {
	return c.Host
}

type Database interface {
	Query(s string) Query
	ApiRoot() string
}

type database struct {
	Name string
	c    Connection
}

func (db *database) Query(s string) Query {
	return &query{
		QueryString: s,
		BatchSize:   5,
		db:          db,
	}
}

func (db *database) ApiRoot() string {
	return db.c.String() + path.Join("/_db", db.Name, "_api")
}

type Query interface {
	SetBatchSize(i int) Query
	Execute() Cursor
	BindVar(key string, val interface{}) Query
	Marshal() string
	Database() Database
}

type query struct {
	QueryString string                 `json:"query"`
	BatchSize   int                    `json:"batchSize,omitempty"`
	Count       bool                   `json:"count"`
	db          Database               `json:"-"`
	BindVars    map[string]interface{} `json:"bindVars,omitempty"`
}

func (q *query) SetBatchSize(i int) Query {
	q.BatchSize = i
	return q
}

func (q *query) Execute() Cursor {
	return &cursor{
		HasMore: true,
		Query:   q,
	}
}

func (q *query) Marshal() string {
	q.Count = true
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(q)
	return string(buf.Bytes())
}

func (q *query) BindVar(key string, val interface{}) Query {
	if q.BindVars == nil {
		q.BindVars = make(map[string]interface{})
	}
	q.BindVars[key] = val
	return q
}

func (q *query) Database() Database {
	return q.db
}

type Cursor interface {
	More() bool
	Next(v interface{}) error
}

type cursor struct {
	HasMore bool          `json:"hasMore"`
	Error   interface{}   `json:"error"`
	Id      string        `json:"id"`
	Code    int           `json:"code"`
	Result  []interface{} `josn:"result"`

	Count    int   `json:"count"`
	Query    Query `json:"-"`
	Position int   `json:"-"`
}

func (c *cursor) More() bool {
	return c.HasMore || c.Position < c.Count
}

func (c *cursor) Next(v interface{}) error {
	if c.Position == c.Count {
		if err := c.nextBatch(); err != nil {
			return err
		}
	}
	if c.Count > 0 {
		err := remarshal(v, c.Result[c.Position])
		c.Position++
		return err
	}
	return nil
}

func All(c Cursor, v interface{}) error {
	slicePtr := reflect.ValueOf(v)
	if slicePtr.Kind() != reflect.Ptr {
		return fmt.Errorf("Expected pointer")
	}
	if slicePtr.Type().Elem().Kind() != reflect.Slice {
		return fmt.Errorf("Expected pointer to slice value")
	}
	elemType := slicePtr.Type().Elem().Elem()

	slice := reflect.MakeSlice(slicePtr.Type().Elem(), 0, 5)
	for c.More() {
		v := reflect.New(elemType)
		err := c.Next(v.Interface())
		if err != nil {
			return err
		}
		slice = reflect.Append(slice, v.Elem())
	}
	slicePtr.Elem().Set(slice)
	return nil
}

func (c *cursor) nextBatch() error {
	var req *http.Request
	var err error
	if c.Id == "" {
		req, err = http.NewRequest("POST", c.Query.Database().ApiRoot()+"/cursor", strings.NewReader(c.Query.Marshal()))
	} else {
		req, err = http.NewRequest("PUT", c.Query.Database().ApiRoot()+"/cursor/"+c.Id, nil)
	}
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(c); err != nil {
		return err
	}
	c.Position = 0

	if b, ok := c.Error.(bool); ok && !b {
		return nil
	}
	switch x := c.Error.(type) {
	case string:
		return fmt.Errorf(x)
	default:
		return fmt.Errorf("Some error occured: %#v", c)
	}
	panic("Unreachable")
}

func remarshal(v1 interface{}, v2 interface{}) error {
	b, err := json.Marshal(v2)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v1)
}
