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

type Edge struct {
	Id   string `json:"_id"`
	From string `json:"_from"`
	To   string `json:"_to"`
}

type Path struct {
	Edges    []Edge `json:"edges"`
	Vertices []Node `json:"vertices"`
}

type TraversalStep struct {
	Path   Path `json:"path"`
	Vertex Node `json:"vertex"`
}

func main() {
	goptions.ParseAndFail(&options)

	conn, err := NewConnection(options.ArangoDB.String())
	if err != nil {
		log.Fatalf("Could not connect to ArangoDB: %s", err)
	}
	x := []TraversalStep{}
	qry := conn.Database("bt").Query(`FOR p IN TRAVERSAL(nodes, edges, @startid, "outbound", {paths: true}) RETURN p`)
	qry.BindVar("startid", "nodes/27080533")
	cur := qry.Execute()
	if err := All(cur, &x); err != nil {
		log.Fatalf("Could not get nodes: %s", err)
	}
	// d, _ := json.MarshalIndent(x, "", "\t")
	// log.Printf("\n%s", d)
	log.Printf("%#v", x)
}
