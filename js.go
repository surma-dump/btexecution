package main

import (
	"github.com/robertkrimen/otto"
)

var (
	jsEnvPool = make(chan JSEnv, POOL_SIZE)
)

func NewJSEnv() JSEnv {
	return &ottoJSEnv{otto.New()}
}

func GetJSEnv() JSEnv {
	// TODO: Pooling is not viable until we find a way
	// to clean the context after usage.
	// select {
	// case env := <-jsEnvPool:
	// 	return env
	// default:
	// }
	return NewJSEnv()
}

func PutJSEnv(env JSEnv) {
	select {
	case jsEnvPool <- env:
	default:
	}
}

type JSEnv interface {
	Get(name string) (interface{}, error)
	Set(name string, val interface{}) error
	Execute(code string) (interface{}, error)
}

type ottoJSEnv struct {
	*otto.Otto
}

func (o *ottoJSEnv) Get(name string) (interface{}, error) {
	v, err := o.Otto.Get(name)
	if err != nil {
		return v, err
	}
	return v.Export()
}

func (o *ottoJSEnv) Set(name string, val interface{}) error {
	return o.Otto.Set(name, val)
}

func (o *ottoJSEnv) Execute(code string) (interface{}, error) {
	v, err := o.Otto.Run(code)
	if err != nil {
		return v, err
	}
	return v.Export()
}
