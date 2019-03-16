package ajson

import (
	"strconv"
	"sync/atomic"
)

// Main struct, presents any json Node
type Node struct {
	parent   *Node
	children []*Node
	key      *string
	index    *int
	_type    NodeType
	data     *[]byte
	borders  [2]int
	value    atomic.Value
}

type NodeType int

const (
	Null NodeType = iota
	Numeric
	String
	Bool
	Array
	Object
)

func newNode(parent *Node, buf *buffer, _type NodeType, key **string) (current *Node, err error) {
	current = &Node{
		parent:  parent,
		data:    &buf.data,
		borders: [2]int{buf.index, 0},
		_type:   _type,
		key:     *key,
	}
	if parent != nil {
		if parent.IsArray() {
			size := len(parent.children)
			current.index = &size
			parent.children = append(parent.children, current)
		} else if parent.IsObject() {
			parent.children = append(parent.children, current)
			if *key == nil {
				err = errorSymbol(buf)
			} else {
				*key = nil
			}
		} else {
			err = errorSymbol(buf)
		}
	}
	return
}

func (n *Node) Source() []byte {
	return (*n.data)[n.borders[0]:n.borders[1]]
}

func (n *Node) String() string {
	return string(n.Source())
}

func (n *Node) Type() NodeType {
	return n._type
}

func (n *Node) Key() string {
	return *n.key
}

func (n *Node) Index() int {
	return *n.index
}

func (n *Node) Size() int {
	return len(n.children)
}

func (n *Node) Keys() (result []string) {
	result = make([]string, 0, len(n.children))
	for _, child := range n.children {
		if child.key != nil {
			result = append(result, *child.key)
		}
	}
	return
}

func (n *Node) IsArray() bool {
	return n._type == Array
}

func (n *Node) IsObject() bool {
	return n._type == Object
}

func (n *Node) IsNull() bool {
	return n._type == Null
}

func (n *Node) IsNumeric() bool {
	return n._type == Numeric
}

func (n *Node) IsString() bool {
	return n._type == String
}

func (n *Node) IsBool() bool {
	return n._type == Bool
}

func (n *Node) Value() (value interface{}, err error) {
	value = n.value.Load()
	if value == nil {
		switch n._type {
		case Null:
			return nil, nil
		case Numeric:
			value, err = strconv.ParseFloat(string(n.Source()), 64)
			if err != nil {
				return
			}
			n.value.Store(value)
		case String:
			size := len(n.Source())
			value = string(n.Source()[1 : size-1])
			n.value.Store(value)
		case Bool:
			b := n.Source()[0]
			value = b == 't' || b == 'T'
			n.value.Store(value)
		case Array:
			children := make([]*Node, 0, len(n.children))
			for _, child := range n.children {
				children = append(children, child)
			}
			value = children
			n.value.Store(value)
		case Object:
			result := make(map[string]*Node)
			for _, child := range n.children {
				result[child.Key()] = child
			}
			value = result
			n.value.Store(value)
		}
	}
	return
}

func (n *Node) GetNull() (value interface{}, err error) {
	if n._type != Null {
		return value, errorType()
	}
	return
}

func (n *Node) GetNumeric() (value float64, err error) {
	if n._type != Numeric {
		return value, errorType()
	}
	iValue, err := n.Value()
	if err != nil {
		return 0, err
	}
	value = iValue.(float64)
	return
}

func (n *Node) GetString() (value string, err error) {
	if n._type != String {
		return value, errorType()
	}
	iValue, err := n.Value()
	if err != nil {
		return "", err
	}
	value = iValue.(string)
	return
}

func (n *Node) GetBool() (value bool, err error) {
	if n._type != Bool {
		return value, errorType()
	}
	iValue, err := n.Value()
	if err != nil {
		return false, err
	}
	value = iValue.(bool)
	return
}

func (n *Node) GetArray() (value []*Node, err error) {
	if n._type != Array {
		return value, errorType()
	}
	iValue, err := n.Value()
	if err != nil {
		return nil, err
	}
	value = iValue.([]*Node)
	return
}

func (n *Node) GetObject() (value map[string]*Node, err error) {
	if n._type != Object {
		return value, errorType()
	}
	iValue, err := n.Value()
	if err != nil {
		return nil, err
	}
	value = iValue.(map[string]*Node)
	return
}

func (n *Node) MustNull() (value interface{}) {
	value, err := n.GetNull()
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) MustNumeric() (value float64) {
	value, err := n.GetNumeric()
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) MustString() (value string) {
	value, err := n.GetString()
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) MustBool() (value bool) {
	value, err := n.GetBool()
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) MustArray() (value []*Node) {
	value, err := n.GetArray()
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) MustObject() (value map[string]*Node) {
	value, err := n.GetObject()
	if err != nil {
		panic(err)
	}
	return
}

// Recursive: Unpack value to interface
func (n *Node) Unpack() (value interface{}, err error) {
	switch n._type {
	case Null:
		return nil, nil
	case Numeric:
		value, err = strconv.ParseFloat(string(n.Source()), 64)
		if err != nil {
			return
		}
	case String:
		size := len(n.Source())
		value = string(n.Source()[1 : size-1])
	case Bool:
		b := n.Source()[0]
		value = b == 't' || b == 'T'
	case Array:
		children := make([]interface{}, 0, len(n.children))
		for _, child := range n.children {
			val, err := child.Unpack()
			if err != nil {
				return nil, err
			}
			children = append(children, val)
		}
		value = children
	case Object:
		result := make(map[string]interface{})
		for _, child := range n.children {
			result[child.Key()], err = child.Unpack()
			if err != nil {
				return nil, err
			}
		}
		value = result
	}
	return
}

func (n *Node) GetIndex(index int) (*Node, error) {
	if n._type != Array {
		return nil, errorType()
	}
	if index < 0 || index >= len(n.children) {
		return nil, errorRequest()
	}
	return n.children[index], nil
}

func (n *Node) MustIndex(index int) (value *Node) {
	value, err := n.GetIndex(index)
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) GetKey(key string) (*Node, error) { // TODO: refactor
	if n._type != Object {
		return nil, errorType()
	}
	for _, value := range n.children {
		if value.key != nil && *value.key == key {
			return value, nil
		}
	}
	return nil, errorRequest()
}

func (n *Node) MustKey(key string) (value *Node) {
	value, err := n.GetKey(key)
	if err != nil {
		panic(err)
	}
	return
}

func (n *Node) ready() bool {
	return n.borders[1] != 0
}

func (n *Node) isContainer() bool {
	return n._type == Array || n._type == Object
}
