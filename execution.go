package main

import (
	"fmt"
)

const (
	POOL_SIZE = 10
)

func ExecuteTree(node *Node, doc interface{}) map[string]*NodeResult {
	if impl, ok := nodeImplementations[node.Type]; ok {
		return impl(node, doc)
	}
	return map[string]*NodeResult{
		node.Id: &NodeResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown node type %s on %s", node.Type, node.Id),
		},
	}
}

type NodeTypeFunc func(*Node, interface{}) map[string]*NodeResult

type NodeResult struct {
	Success bool     `json:"success"`
	Tags    []string `json:"tags"`
	Error   string   `json:"error"`
}

var nodeImplementations map[string]NodeTypeFunc

func init() {
	nodeImplementations = map[string]NodeTypeFunc{
		"all": func(node *Node, doc interface{}) map[string]*NodeResult {
			errors := map[string]*NodeResult{
				node.Id: &NodeResult{
					Success: true,
					Tags:    node.Tags,
				},
			}
			for _, child := range node.Children {
				err := ExecuteTree(child, doc)
				for cnode, cresult := range err {
					errors[cnode] = cresult
				}
				errors[node.Id].Success = errors[node.Id].Success && errors[child.Id].Success
			}
			return errors
		},
		"any": func(node *Node, doc interface{}) map[string]*NodeResult {
			errors := map[string]*NodeResult{
				node.Id: &NodeResult{
					Success: false,
					Tags:    node.Tags,
				},
			}
			for _, child := range node.Children {
				err := ExecuteTree(child, doc)
				for cnode, cresult := range err {
					errors[cnode] = cresult
				}
				errors[node.Id].Success = errors[node.Id].Success || errors[child.Id].Success
			}
			return errors
		},
		"script": func(node *Node, doc interface{}) map[string]*NodeResult {
			env := GetJSEnv()
			defer PutJSEnv(env)
			env.Set("doc", doc)
			defer env.Set("doc", nil)
			defer env.Set("error", nil)

			r := map[string]*NodeResult{
				node.Id: &NodeResult{
					Tags: node.Tags,
				},
			}
			v, err := env.Execute(node.Content)
			if err != nil {
				r[node.Id].Success = false
				r[node.Id].Error = err.Error()
				return r
			}
			if b, ok := v.(bool); ok {
				r[node.Id].Success = b
				errv, _ := env.Get("error")
				if err, ok := errv.(string); ok {
					r[node.Id].Error = err
				}
				return r
			}
			r[node.Id].Success = false
			r[node.Id].Error = "Invalid return value"
			return r
		},
	}
}
