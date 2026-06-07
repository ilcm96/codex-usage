package usage

import (
	"fmt"
	"strings"
)

func addDateFilters(filters Filters, where *[]string, args *[]any, column string) {
	if filters.From != "" {
		*where = append(*where, fmt.Sprintf("%s >= %s::date", column, appendSQLArg(args, filters.From)))
	}
	if filters.To != "" {
		*where = append(*where, fmt.Sprintf("%s < (%s::date + INTERVAL '1 day')", column, appendSQLArg(args, filters.To)))
	}
}

func addIDFilter(value string, where *[]string, args *[]any, column string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if !isUUIDText(value) {
		*where = append(*where, "1=0")
		return
	}
	*where = append(*where, fmt.Sprintf("%s = %s::uuid", column, appendSQLArg(args, value)))
}

func addTextFilter(value string, where *[]string, args *[]any, column string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*where = append(*where, fmt.Sprintf("%s = %s", column, appendSQLArg(args, value)))
}

func appendSQLArg(args *[]any, value any) string {
	*args = append(*args, value)
	return fmt.Sprintf("$%d", len(*args))
}

func isUUIDText(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') && !(r >= 'A' && r <= 'F') {
				return false
			}
		}
	}
	return true
}
