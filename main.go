package main

import (
	"log"
	"net/url"

	"github.com/voxelbrain/goptions"
)

var (
	options = struct {
		ArangoDB *url.URL      `goptions:"-d, --database, description='URL of ArangoDB instance', obligatory"`
		Help     goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{}
)

type Node struct {
	Id      string   `json:"_id"`
	Type    string   `json:"type"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func main() {
	goptions.ParseAndFail(&options)

	conn, err := NewConnection(options.ArangoDB.String())
	if err != nil {
		log.Fatalf("Could not connect to ArangoDB: %s", err)
	}
	var x Node
	cur := conn.Database("bt").Query("FOR n IN nodes RETURN n").Execute()
	for cur.More() {
		err := cur.Next(&x)
		if err != nil {
			log.Fatalf("Getting item failed: %s", err)
		}
		log.Printf("%#v", x)
	}
}
