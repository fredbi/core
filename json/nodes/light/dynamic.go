package light

/*
func (n *Node) FromDynamicJSON(s stores.Store, value any, opts DecodeOptions) error {
	*n = nullNode

	return n.decodeDynamicJSON(s, value, opts)
}

func (n Node) ToDynamicJSON(s stores.Store, opts EncodeOptions) any {
	return n.encodeDynamicJSON(s, opts)
}

func (n *Node) decodeDynamicJSON(s stores.Store, value any, opts DecodeOptions) error {
	if value == nil {
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindNull
		n.value = s.PutNull()

		return nil
	}


*/
/* TODO(fred): implement support for hooks from dynamic JSON ...
if opts.NodeHook != nil {
	// hook: callback before a node is processed
	skip, err := opts.NodeHook(l, tok)
	if err != nil {
		l.SetErr(err)
		return
	}

	if skip {
		continue
	}
}
*/

/*
	switch source := value.(type) {
	case map[string]any:
		n.keysIndex = make(map[stores.InternedKey]int, len(source))
		n.children = make([]Node, 0, len(source))
		n.value = s.PutNull()
		n.kind = nodes.KindObject

		for key, val := range source {
			var node Node
			if err := node.decodeDynamicJSON(s, val, opts); err != nil {
				return err
			}
			k := stores.MakeInternedKey(key)
			node.key = k
			n.children = append(n.children, node)
			n.keysIndex[k] = len(n.children) - 1
		}

	case []any:
		n.keysIndex = nil
		n.children = make([]Node, 0, len(source))
		n.value = s.PutNull()
		n.kind = nodes.KindArray

		for _, val := range source {
			var node Node
			if err := node.decodeDynamicJSON(s, val, opts); err != nil {
				return err
			}

			n.children = append(n.children, node)
		}

	case string:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeStringValue(source))

	case float64:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeFloatValue(source))

	case float32:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeFloatValue(source))
	case int64:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeIntegerValue(source))

	case uint64:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeUintegerValue(source))
	case bool:
		n.keysIndex = nil
		n.children = nil
		n.kind = nodes.KindScalar
		n.value = s.PutValue(stores.MakeBoolValue(source))
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}

	return nil
}

func (n Node) encodeDynamicJSON(s stores.Store, opts EncodeOptions) any {
	switch n.kind {
	case nodes.KindObject:
		value := make(map[string]any, len(n.children))
		for _, pair := range n.children {
			value[pair.key.String()] = pair.encodeDynamicJSON(s, opts)
		}

		return value

	case nodes.KindArray:
		value := make([]any, 0, len(n.children))
		for _, elem := range n.children {
			value = append(value, elem.encodeDynamicJSON(s, opts))
		}

		return value

	case nodes.KindScalar:
		v := s.Get(n.value)
		switch v.Kind() {
		case token.String:
			return v.String()
		case token.Boolean:
			return v.Bool()
		case token.Number:
			return 0 // TODO(fred): convert number to native type
		default:
			panic(fmt.Errorf("invalid value type. Got: %v", v.Kind()))
		}

	case nodes.KindNull:
		return nil

	default:
		panic(fmt.Errorf("invalid node type. Got: %v", n.kind))
	}
}

*/
