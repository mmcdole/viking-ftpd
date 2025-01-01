package authorization

import "fmt"

// ConvertToAccessTrees converts raw data into a map of access trees
func ConvertToAccessTrees(rawData map[string]interface{}) (map[string]*AccessTree, error) {
	result := make(map[string]*AccessTree)

	for username, rawUserTree := range rawData {
		fmt.Printf("DEBUG: Converting tree for user %q\n", username)
		tree, err := convertToAccessTree(rawUserTree.(map[string]interface{}))
		if err != nil {
			return nil, fmt.Errorf("converting tree for user %s: %w", username, err)
		}
		result[username] = tree
	}
	return result, nil
}

// convertToAccessTree converts a raw data map into an access tree
func convertToAccessTree(data map[string]interface{}) (*AccessTree, error) {
	root, groups, err := convertToAccessNode(data)
	if err != nil {
		return nil, err
	}

	return &AccessTree{
		Root:   root,
		Groups: groups,
	}, nil
}

// convertToAccessNode recursively converts a raw data map into an access node
func convertToAccessNode(data map[string]interface{}) (*AccessNode, []string, error) {
	node := &AccessNode{
		DotAccess:  Revoked,
		StarAccess: Revoked,
		Children:   make(map[string]*AccessNode),
	}

	var groups []string

	for key, value := range data {
		fmt.Printf("DEBUG: Processing key %q with value type %T\n", key, value)
		switch key {
		case ".":
			perm, err := convertToPermission(value)
			if err != nil {
				return nil, nil, fmt.Errorf("converting dot access: got %T with value %#v", value, value)
			}
			node.DotAccess = perm
		case "*":
			perm, err := convertToPermission(value)
			if err != nil {
				return nil, nil, fmt.Errorf("converting star access: got %T with value %#v", value, value)
			}
			node.StarAccess = perm
		case "?":
			groupList, ok := value.([]interface{})
			if !ok {
				return nil, nil, fmt.Errorf("invalid group list format")
			}
			for _, group := range groupList {
				groupStr, ok := group.(string)
				if !ok {
					return nil, nil, fmt.Errorf("invalid group name format")
				}
				groups = append(groups, groupStr)
			}
		default:
			switch v := value.(type) {
			case map[string]interface{}:
				child, childGroups, err := convertToAccessNode(v)
				if err != nil {
					return nil, nil, fmt.Errorf("converting child node %s: %w", key, err)
				}
				if len(childGroups) > 0 {
					groups = append(groups, childGroups...)
				}
				node.Children[key] = child
			default:
				// Handle direct permission value
				perm, err := convertToPermission(value)
				if err != nil {
					return nil, nil, fmt.Errorf("converting direct permission for %s: %w", key, err)
				}
				// When a node has a direct permission value, it applies to both the node itself
				// and all nodes below it
				child := &AccessNode{
					DotAccess:  perm, // Permission for the node itself
					StarAccess: perm, // Default permission for children
					Children:   make(map[string]*AccessNode),
				}
				node.Children[key] = child
			}
		}
	}

	return node, groups, nil
}

// convertToPermission converts a raw permission value into a Permission
func convertToPermission(value interface{}) (Permission, error) {
	fmt.Printf("DEBUG: Converting permission value: %#v (type: %T)\n", value, value)
	switch v := value.(type) {
	case int:
		return Permission(v), nil
	case Permission:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid permission format")
	}
}
