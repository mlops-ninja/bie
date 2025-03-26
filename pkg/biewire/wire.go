package biewire

const (
	OpPing = iota
	OpGet
)

// ClientRequest represents a request from a client
type ClientRequest struct {
	AuthToken string `json:"auth_token"`
	Intention string `json:"intention"`
}

type ClientResponse struct {
	Token string `json:"token"`
}

// Intention can be send or get for example
