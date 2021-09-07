package viewer

import (
	"fmt"
	"strings"
)

func handleQuotingOfInterface(i interface{}) string {
	//switch i.(type) {
	//case string:
	//	breakv
	//}
	// int, float64,
}

type Update struct {
	Values    map[string]interface{} // these are anchors to ensure the right row/col gets updated
	Column    string                 // this is the header
	Update    interface{}            // this is the new cell value
	TableName string
}

func (u *Update) GenerateQuery() (query string) {
	querySkeleton := fmt.Sprintf("UPDATE %s "+
		"SET %s = %s ", u.TableName, u.Column, handleQuotingOfInterface(u.Update))
	whereBuilder := strings.Builder{}
	whereBuilder.WriteString("WHERE ")
	uLen := len(u.Values)
	i := 0
	for k, v := range u.Values {
		// this might not be good enough... may need to give up and just handle the prepared statements
		assertion := fmt.Sprintf("%s = %s ", k, handleQuotingOfInterface(v))
		whereBuilder.WriteString(assertion)
		if uLen > 1 && i < uLen-1 {
			whereBuilder.WriteString("AND ")
		}
		i++
	}
	query = querySkeleton + strings.TrimSpace(whereBuilder.String()) + ";"
	return query
}
