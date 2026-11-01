package community

// Requests are authenticated by libp2p and again by signed operations. Sync never trusts a snapshot.
type Request struct {
	Method      string     `json:"method"`
	CommunityID string     `json:"communityId"`
	After       uint64     `json:"after"`
	Head        string     `json:"head"`
	Operation   *Operation `json:"operation,omitempty"`
}
type Response struct {
	Commits  []Commit `json:"commits"`
	Sequence uint64   `json:"sequence"`
	Head     string   `json:"head"`
	Ack      string   `json:"ack"`
	Invite   string   `json:"invite,omitempty"`
	Error    string   `json:"error,omitempty"`
	Code     string   `json:"code,omitempty"`
}
