package dsl

import (
	"encoding/json"
	"fmt"

	"github.com/Comcast/plax/subst"
	"gopkg.in/yaml.v3"
)

// Serialization is a enum of possible (de)serializations.
type Serialization string

var (
	serJSON   Serialization = "JSON"
	serString Serialization = "string"

	// DefaultSerialization is of course the default Serialization
	// (for pub and recv operation).
	DefaultSerialization = serJSON

	// Serializations is a dictionary of supported Serializations.
	Serializations = map[string]Serialization{
		string(serJSON):   serJSON,
		string(serString): serString,
	}
)

func NewSerialization(name string) (*Serialization, error) {
	ser, have := Serializations[name]
	if !have {
		allowed := make([]string, 0, len(Serializations))
		for name := range Serializations {
			allowed = append(allowed, name)
		}
		return nil, fmt.Errorf("requested serialization '%s' isn't one of %v", name, allowed)
	}
	return &ser, nil
}

func (s *Serialization) UnmarshalYAML(value *yaml.Node) error {
	var name string
	if err := value.Decode(&name); err != nil {
		return err
	}
	ser, err := NewSerialization(name)
	if err != nil {
		return err
	}
	*s = *ser
	return nil
}

// Serialize attempts to render the given argument.
func (s *Serialization) Serialize(x interface{}) (string, error) {
	var (
		ser = DefaultSerialization
		err error
		dst string
	)

	if s != nil {
		ser = *s
	}

	switch ser {
	case serJSON:
		var js []byte
		if js, err = subst.JSONMarshal(&x); err == nil {
			dst = string(js)
		}
	case serString:
		if str, is := x.(string); is {
			dst = str
		} else {
			fmt.Errorf("can't serialize %s from a %T", *s, x)
		}
	default:
		err = fmt.Errorf("internal error: unknown Serialization %#v", s)
	}

	return dst, err
}

// Deserialize attempts to deserialize the given string.
func (s *Serialization) Deserialize(str string) (interface{}, error) {
	var (
		ser = DefaultSerialization
		err error
		dst interface{}
	)

	if s != nil {
		ser = *s
	}

	switch ser {
	case serJSON:
		err = json.Unmarshal([]byte(str), &dst)
	case serString:
		dst = str
	default:
		err = fmt.Errorf("internal error: unknown Serialization %#v", s)
	}

	return dst, err
}
