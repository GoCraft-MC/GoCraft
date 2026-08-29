package command

import "fmt"

const (
	wireNodeLiteral     = 1
	wireNodeArgument    = 2
	maximumCommandNodes = 4096
	maximumCommandDepth = 64
)

type wireNode struct {
	kind         uint64
	name         string
	permission   string
	argumentType uint64
	enum         []string
	executor     uint64
	children     []Node
	customType   string
	integerMin   *int64
	integerMax   *int64
	decimalMin   *float64
	decimalMax   *float64
}

func (decoded wireNode) commandNode() (Node, error) {
	if decoded.executor > uint64(^uint32(0)) {
		return nil, fmt.Errorf("command node %q: executor id overflows uint32", decoded.name)
	}
	switch decoded.kind {
	case wireNodeLiteral:
		if decoded.argumentType != 0 || len(decoded.enum) != 0 || decoded.customType != "" ||
			decoded.integerMin != nil || decoded.integerMax != nil ||
			decoded.decimalMin != nil || decoded.decimalMax != nil {
			return nil, fmt.Errorf("command literal %q contains argument fields", decoded.name)
		}
		return Literal{
			Name: decoded.name, Permission: decoded.permission,
			Children: decoded.children, Exec: ExecID(decoded.executor),
		}, nil
	case wireNodeArgument:
		if decoded.permission != "" {
			return nil, fmt.Errorf("command argument %q contains a permission", decoded.name)
		}
		if decoded.argumentType > uint64(ArgCustom) {
			return nil, fmt.Errorf("command argument %q has invalid type %d", decoded.name, decoded.argumentType)
		}
		return Argument{
			Name: decoded.name, Type: ArgType(decoded.argumentType), Enum: decoded.enum,
			CustomType: decoded.customType, IntegerMin: decoded.integerMin, IntegerMax: decoded.integerMax,
			DecimalMin: decoded.decimalMin, DecimalMax: decoded.decimalMax,
			Children: decoded.children, Exec: ExecID(decoded.executor),
		}, nil
	default:
		return nil, fmt.Errorf("command node %q has invalid kind %d", decoded.name, decoded.kind)
	}
}
