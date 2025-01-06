package authorization

import "fmt"

// BuildAccessTrees constructs a map of access trees from raw data
func BuildAccessTrees(rawData map[string]interface{}) (map[string]*AccessTree, error) {
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
		tree, err := buildAccessTree(userMap)
		if err != nil {
			return nil, fmt.Errorf("building tree for user %s: %w", username, err)
		}
		result[username] = tree
	}
	return result, nil
}

// buildAccessTree constructs an access tree from raw data
func buildAccessTree(data map[string]interface{}) (*AccessTree, error) {
	root, groups, err := buildAccessNode(data)
	if err != nil {
		return nil, err
	}

	return &AccessTree{
		Root:   root,
		Groups: groups,
	}, nil
}

// buildAccessNode recursively constructs an access node from raw data
func buildAccessNode(data map[string]interface{}) (*AccessNode, []string, error) {
	node := &AccessNode{
		DotAccess:  Revoked,
		StarAccess: Revoked,
		Children:   make(map[string]*AccessNode),
	}

	var groups []string

	for key, value := range data {
		switch key {
		case ".":
			perm, err := parsePermission(value)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing dot access: %w", err)
			}
			node.DotAccess = perm
		case "*":
			// Star access can be either a direct permission or a directory node
			if childMap, ok := value.(map[string]interface{}); ok {
				child, childGroups, err := buildAccessNode(childMap)
				if err != nil {
					return nil, nil, fmt.Errorf("building star directory: %w", err)
				}
				node.Children["*"] = child
				groups = append(groups, childGroups...)
			} else {
				perm, err := parsePermission(value)
				if err != nil {
					return nil, nil, fmt.Errorf("parsing star access: %w", err)
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
				child, childGroups, err := buildAccessNode(v)
				if err != nil {
					return nil, nil, fmt.Errorf("building child node %s: %w", key, err)
				}
				if len(childGroups) > 0 {
					groups = append(groups, childGroups...)
				}
				node.Children[key] = child
			default:
				// Handle direct permission value
				perm, err := parsePermission(value)
				if err != nil {
					return nil, nil, fmt.Errorf("parsing permission for %s: %w", key, err)
				}
				child := &AccessNode{
					DotAccess: perm,
					StarAccess: Revoked,
					Children:   make(map[string]*AccessNode),
				}
				node.Children[key] = child
			}
		}
	}
	return node, groups, nil
}

// parsePermission converts a raw permission value into a Permission
func parsePermission(value interface{}) (Permission, error) {
	switch v := value.(type) {
	case float64:
		return Permission(int(v)), nil
	case int:
		return Permission(v), nil
	default:
		return Revoked, fmt.Errorf("invalid permission format: expected number, got %T", value)
	}
}
