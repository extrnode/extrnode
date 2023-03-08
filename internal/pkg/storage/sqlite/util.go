package sqlite

import (
	"fmt"
	"strings"
)

type (
	ArrayContain map[string]interface{}
)

func (a ArrayContain) ToSql() (sql string, args []interface{}, err error) {
	var exprs []string

	for key := range a {
		val, ok := a[key].([]string)
		if !ok {
			return sql, args, fmt.Errorf("fail cast to []string")
		}

		for _, v := range val {
			args = append(args, v)
		}
		exprs = append(exprs, fmt.Sprintf("json_array_length(json_group_array(%s)) == %d", key, len(val)))
	}

	sql = strings.Join(exprs, " AND ")

	return
}
