package subst

import (
	"encoding/json"
	"fmt"
	"reflect"
)

var DefaultMarshaller = &JSONMarshaller{}

type Marshaller interface {
	Marshal(interface{}) (string, error)
	Unmarshal(string, interface{}) error
}

type JSONMarshaller struct {
}

func (m *JSONMarshaller) Marshal(x interface{}) (string, error) {
	bs, err := json.Marshal(x)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func (m *JSONMarshaller) Unmarshal(s string, x interface{}) error {
	return json.Unmarshal([]byte(s), x)
}

type StringMarshaller struct {
}

func (m *StringMarshaller) Marshal(x interface{}) (string, error) {
	s, is := x.(string)
	if !is {
		return "", fmt.Errorf("%#v (%T) isn't a %T", x, x, s)
	}
	return s, nil
}

func (m *StringMarshaller) Unmarshal(s string, x interface{}) error {
	rv := reflect.ValueOf(x)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &json.InvalidUnmarshalError{reflect.TypeOf(x)}
	}
	rv.Set(reflect.ValueOf(s))

	return nil
}
