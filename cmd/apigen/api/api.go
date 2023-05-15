package api

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

type AccessRecord struct {
	Date     time.Time
	Checksum [32]byte
}

type Endpoints map[Category][]Endpoint

func (e Endpoints) String() string {
	builder := strings.Builder{}

	for category, endpoints := range e {
		_, err := fmt.Fprintf(&builder, "Category: %v\nEndpoints:\n", category)
		if err != nil {
			panic(fmt.Errorf("could not print to string buffer: %v", err))
		}
		for _, endpoint := range endpoints {
			_, err := fmt.Fprintf(&builder, "%v", endpoint)
			if err != nil {
				panic(fmt.Errorf("could not print to string buffer: %v", err))
			}
		}

		_, err = fmt.Fprintf(&builder, "\n")
		if err != nil {
			panic(fmt.Errorf("could not print to string buffer: %v", err))
		}
	}
	_, err := fmt.Fprintf(&builder, "\n")
	if err != nil {
		panic(fmt.Errorf("could not print to string buffer: %v", err))
	}

	return builder.String()
}

type Category struct {
	LinkName     string
	ReadableName string
	Desc         string
}

func (c Category) String() string {
	return fmt.Sprintf("(%v) %v:\n%v", c.LinkName, c.ReadableName, c.Desc)
}

type Endpoint struct {
	LinkName     string
	ReadableName string
	Desc         string
	Function     string
	Premium      bool
	Params       []Parameter
}

func (e Endpoint) String() string {
	var premium string
	if e.Premium {
		premium = "[PREMIUM] "
	} else {
		premium = "          "
	}

	var str bytes.Buffer

	_, err := fmt.Fprintf(&str, "%v %v (%v / %v) - %v:\n\n",
		premium, e.ReadableName, e.LinkName, e.Function, e.Desc)
	if err != nil {
		panic(fmt.Errorf("could not write to string buffer: %v", err))
	}

	for _, param := range e.Params {
		_, err := fmt.Fprintf(&str, "%v\n", param)
		if err != nil {
			panic(fmt.Errorf("could not write to string buffer: %v", err))
		}
	}

	_, err = fmt.Fprintf(&str, "\n")
	if err != nil {
		panic(fmt.Errorf("could not write to string buffer: %v", err))
	}
	return str.String()
}

type Parameter struct {
	Required bool
	Name     string
	Desc     string
}

func (p Parameter) String() string {
	var requiredOrOptional string
	if p.Required {
		requiredOrOptional = "Required: "
	} else {
		requiredOrOptional = "Optional: "
	}

	return fmt.Sprintf("\t- %v %v: %v", requiredOrOptional, p.Name, p.Desc)
}
