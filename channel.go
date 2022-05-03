package chancon

type Channel struct {
	Name       string      `json:"name"`
	Connection *Connection `json:"-"`
}
