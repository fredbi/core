package spec

import "github.com/fredbi/core/json"

type OpenAPIVersion uint8

const (
	OpenAPIVersion20 OpenAPIVersion = iota + 2
	OpenAPIVersion30
	OpenAPIVersion31
)

const (
	SwaggerVersion2 = OpenAPIVersion20
)

type Spec struct {
	json.Document
}

func Make() Spec {
	return Spec{}
}

func New() *Spec {
	s := Make()

	return &s
}
