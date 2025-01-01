package authorization

import "fmt"

// ConvertToAccessTrees converts raw data into a map of access trees
func ConvertToAccessTrees(rawData map[string]interface{}) (map[string]*AccessTree, error) {
	result := make(map[string]*AccessTree)

	// Look for access_map key
	accessMap, ok := rawData["access_map"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("access_map not found or invalid format")
	}

	for username, rawUserTree := range accessMap {
		userMap, ok := rawUserTree.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid user tree format for %s: expected map[string]interface{}, got %T", username, rawUserTree)
		}
		tree, err := convertToAccessTree(userMap)
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
		switch key {
		case ".":
			perm, err := convertToPermission(value)
			if err != nil {
				return nil, nil, fmt.Errorf("converting dot access: %w", err)
			}
			node.DotAccess = perm
		case "*":
			// Star access can be either a direct permission or a directory node
			if childMap, ok := value.(map[string]interface{}); ok {
				child, childGroups, err := convertToAccessNode(childMap)
				if err != nil {
					return nil, nil, fmt.Errorf("converting star directory: %w", err)
				}
				node.Children["*"] = child
				groups = append(groups, childGroups...)
			} else {
				perm, err := convertToPermission(value)
				if err != nil {
					return nil, nil, fmt.Errorf("converting star access: %w", err)
				}
				node.StarAccess = perm
			}
		case "?":
			groupList, ok := value.([]interface{})
			if !ok {
				return nil, nil, fmt.Errorf("invalid group list format: expected []interface{}, got %T", value)
			}
			for _, group := range groupList {
				groupStr, ok := group.(string)
				if !ok {
					return nil, nil, fmt.Errorf("invalid group name format: expected string, got %T", group)
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
	switch v := value.(type) {
	case int:
		return Permission(v), nil
	case Permission:
		return v, nil
	default:
		return 0, fmt.Errorf("expected permission value to be an integer, got %T", value)
	}
}
