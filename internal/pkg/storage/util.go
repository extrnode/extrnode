package storage

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
)

type LikeAny map[string]interface{}

func (l LikeAny) ToSql() (sql string, args []interface{}, err error) {
	var exprs []string

	for key := range l {
		val, ok := l[key].([]string)
		if !ok {
			return sql, args, fmt.Errorf("fail cast to []string")
		}

		for _, v := range val {
			args = append(args, v)
		}
		exprs = append(exprs, fmt.Sprintf("%s LIKE ANY (ARRAY [%s])", key, squirrel.Placeholders(len(val))))
	}

	sql = strings.Join(exprs, " AND ")

	return
}
