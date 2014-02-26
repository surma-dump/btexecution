package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/voxelbrain/goptions"
)

var (
	options = struct {
		Listen   string        `goptions:"-l, --listen, description='Address to bind webserver to'"`
		ArangoDB *url.URL      `goptions:"-d, --database, description='URL of ArangoDB instance', obligatory"`
		Help     goptions.Help `goptions:"-h, --help, description='Show this help'"`
	}{
		Listen: "localhost:8080",
	}
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
	db := conn.Database("bt")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}

		nodeName := r.URL.Path[1:]
		if len(nodeName) <= 0 {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		startNode, err := findNodeByName(db, nodeName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}

		nodeList := traverseTree(db, startNode)

		enc := json.NewEncoder(w)
		enc.Encode(nodeList)
	})

	log.Printf("Starting webserver on %s...", options.Listen)
	if err := http.ListenAndServe(options.Listen, nil); err != nil {
		log.Fatalf("Could not start webserver: %s", err)
	}
}

func findNodeByName(db Database, name string) (*Node, error) {
	qry := db.Query(`FOR n IN nodes FOR t IN n.tags FILTER t == @name RETURN n`)
	qry.BindVar("name", "name:"+name)
	cur := qry.Execute()
	defer cur.Close()
	if !cur.More() {
		return nil, fmt.Errorf("No node with name label \"%s\"", name)
	}

	var node Node
	cur.Next(&node)
	return &node, nil
}

func traverseTree(db Database, startNode *Node) []TraversalStep {
	qry := db.Query(`FOR p IN TRAVERSAL(nodes, edges, @startid, "outbound", {paths: true}) RETURN p`)
	qry.BindVar("startid", startNode.Id)
	cur := qry.Execute()
	defer cur.Close()

	var nodeList []TraversalStep
	All(cur, &nodeList)
	return nodeList
}
