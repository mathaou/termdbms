package viewer

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

type Update struct {
	Values    map[string]interface{} // these are anchors to ensure the right row/col gets updated
	Column    string                 // this is the header
	Update    interface{}            // this is the new cell value
	TableName string
}

func getPlaceholderForDatabaseType(db Database) (string, error) {
	switch db.(type) {
	case *SQLite: // MySQL eventually
		return "?", nil
		break
	}

	return "", errors.New("unsupported database type")
}

func (u *Update) GenerateQuery(db Database) (string, []string) {
	var (
		query string
		querySkeleton string
		valueOrder []string
	)

	placeholder, err := getPlaceholderForDatabaseType(db)

	querySkeleton = fmt.Sprintf("UPDATE %s"+
		" SET %s=%s ", u.TableName, u.Column, placeholder)
	valueOrder = append(valueOrder, u.Column)

	whereBuilder := strings.Builder{}
	whereBuilder.WriteString(" WHERE ")
	uLen := len(u.Values)
	i := 0
	for k := range u.Values { // keep track of order since maps aren't deterministic
		if err != nil {
			log.Fatalf("%v", err)
		}
		assertion := fmt.Sprintf("%s=%s ", k, placeholder)
		valueOrder = append(valueOrder, k)
		whereBuilder.WriteString(assertion)
		if uLen > 1 && i < uLen-1 {
			whereBuilder.WriteString("AND ")
		}
		i++
	}
	query = querySkeleton + strings.TrimSpace(whereBuilder.String()) + ";"
	return query, valueOrder
}
