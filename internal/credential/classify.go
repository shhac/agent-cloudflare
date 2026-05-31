package credential

import "strings"

const UnknownType = "unknown"

func Type(token string) string {
	switch {
	case strings.HasPrefix(token, "cfut_"):
		return "cfut"
	case strings.HasPrefix(token, "v1.0-"):
		return "legacy_api_token"
	case token != "":
		return UnknownType
	default:
		return ""
	}
}
