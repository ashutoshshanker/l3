package main

import (
)

var params_dir string

type ARPServiceHandler struct {
}

func NewARPServiceHandler(paramsDir string) *ARPServiceHandler {
	params_dir = paramsDir
	initARPhandlerParams()
	return &ARPServiceHandler{}
}
