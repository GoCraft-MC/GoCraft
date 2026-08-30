package abi

func Bool(value bool) Value {
	return Value{Kind: ValueBool, Bool: value}
}

func Int64(value int64) Value {
	return Value{Kind: ValueInt64, Int64: value}
}

func Double(value float64) Value {
	return Value{Kind: ValueDouble, Double: value}
}

func String(value string) Value {
	return Value{Kind: ValueString, String: value}
}

func Bytes(value []byte) Value {
	return Value{Kind: ValueBytes, Bytes: append([]byte(nil), value...)}
}

func List(values ...Value) Value {
	return Value{Kind: ValueList, List: append([]Value(nil), values...)}
}
