package viewer

type Query interface {
	GetValues() map[string]interface{}
}

type Update struct {
	v    map[string]interface{} // these are anchors to ensure the right row/col gets updated
	Column    string                 // this is the header
	Update    interface{}            // this is the new cell value
	TableName string
}

func (u *Update) GetValues() map[string]interface{} {
	return u.v
}