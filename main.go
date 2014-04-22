package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

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

const (
	DEFAULT_LIMIT = 100
)

type Node struct {
	Id      string   `json:"_id, omitempty"`
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

	r := mux.NewRouter()

	r.PathPrefix("/nodes").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		qryStr, binds := buildQuery("nodes", r.Form)

		qry := db.Query(qryStr)
		for k, v := range binds {
			qry.BindVar(k, v)
		}
		cur := qry.Execute()
		defer cur.Close()

		nodes := []Node{}
		if err := All(cur, &nodes); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(nodes)
	})

	r.PathPrefix("/nodes/{id:[0-9]+}").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var n Node
		if err := db.Get("nodes/"+vars["id"], &n); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		}

		json.NewEncoder(w).Encode(n)
	})

	r.PathPrefix("/nodes").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var n Node
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		n.Id = ""
		n.Children = nil
		id, err := db.Insert("nodes", n)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(id))
	})

	r.PathPrefix("/").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if err := http.ListenAndServe(options.Listen, r); err != nil {
		log.Fatalf("Could not start webserver: %s", err)
	}
}

func buildQuery(collection string, v url.Values) (string, map[string]interface{}) {
	qryStr := "FOR x IN " + collection
	binds := map[string]interface{}{}

	for i, expr := range v["filter"] {
		f := strings.SplitN(expr, ":", 2)
		if len(f) < 2 {
			continue
		}
		fieldVar := fmt.Sprintf("f%d", i)
		valueVar := fmt.Sprintf("fv%d", i)
		comp := "=="
		if f[0] == "tags" {
			comp = "IN"
		}

		qryStr += fmt.Sprintf(" FILTER @%s %s x.@%s", valueVar, comp, fieldVar)
		binds[fieldVar] = f[0]
		binds[valueVar] = f[1]
	}

	for i, expr := range v["exclude"] {
		f := strings.SplitN(expr, ":", 2)
		if len(f) < 2 {
			continue
		}
		fieldVar := fmt.Sprintf("e%d", i)
		valueVar := fmt.Sprintf("ev%d", i)
		comp := "=="
		if f[0] == "tags" {
			comp = "IN"
		}

		qryStr += fmt.Sprintf(" FILTER !(@%s %s x.@%s)", valueVar, comp, fieldVar)
		binds[fieldVar] = f[0]
		binds[valueVar] = f[1]
	}

	binds["skip"] = 0
	if skipStr := v.Get("skip"); skipStr != "" {
		skip, err := strconv.ParseInt(skipStr, 10, 64)
		if err == nil {
			binds["skip"] = int(skip)
		}
	}
	binds["limit"] = DEFAULT_LIMIT
	if limitStr := v.Get("limit"); limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err == nil {
			binds["limit"] = int(limit)
		}
	}

	qryStr += " LIMIT @skip, @limit RETURN x"

	log.Printf("\"%s\" %#v", qryStr, binds)
	return qryStr, binds
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
