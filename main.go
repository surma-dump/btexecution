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
	// Not stored in database, populated at runtime
	Children []*Node `json:"-"`
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
	Path   Path  `json:"path"`
	Vertex *Node `json:"vertex"`
}

func main() {
	goptions.ParseAndFail(&options)

	conn, err := NewConnection(options.ArangoDB.String())
	if err != nil {
		log.Fatalf("Could not connect to ArangoDB: %s", err)
	}
	db := conn.Database("bt")

	id, err := db.Insert("some", map[string]string{"a":"av"});
	if err != nil {
		log.Fatalf("Insert Error: %s", err)
	}
	if err := db.Update(id, map[string]string{"b":"bv"}); err != nil {
		log.Fatalf("Update Errpor: %s", err)
	}
	return

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}
		var doc interface{}
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&doc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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
			return
		}

		nodeList := traverseGraph(db, startNode)
		tree := buildTree(nodeList)

		results := ExecuteTree(tree, doc)

		enc := json.NewEncoder(w)
		enc.Encode(results)
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

	var node Node
	if err := cur.Next(&node); err != nil {
		return nil, fmt.Errorf("No node with name label \"%s\"", name)
	}
	return &node, nil
}

func traverseGraph(db Database, startNode *Node) []TraversalStep {
	qry := db.Query(`FOR p IN TRAVERSAL(nodes, edges, @startid, "outbound", {paths: true}) RETURN p`)
	qry.BindVar("startid", startNode.Id)
	cur := qry.Execute()
	defer cur.Close()

	var nodeList []TraversalStep
	All(cur, &nodeList)
	return nodeList
}

func buildTree(list []TraversalStep) *Node {
	lookup := map[string]*Node{}
	for i, step := range list {
		lookup[step.Vertex.Id] = step.Vertex
		if i == 0 {
			continue
		}
		parentNodeId := step.Path.Edges[len(step.Path.Edges)-1].From
		parentNode := lookup[parentNodeId]
		if parentNode.Children == nil {
			parentNode.Children = make([]*Node, 0)
		}
		parentNode.Children = append(parentNode.Children, step.Vertex)
	}
	return list[0].Vertex
}
