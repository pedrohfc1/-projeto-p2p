package community

import "projetop2p/internal/chat"

// Requests are authenticated by libp2p and again by signed operations. Sync never trusts a snapshot.
type InvitationStatus struct {
	Nonce     string `json:"nonce"`
	Used      uint32 `json:"used"`
	Remaining int64  `json:"remaining"`
	Revoked   bool   `json:"revoked"`
	Sequence  uint64 `json:"sequence"`
}
type Request struct {
	ActivityRoom  *chat.Room     `json:"activityRoom,omitempty"`
	Activity      *chat.Event    `json:"activity,omitempty"`
	ActivityAfter uint64         `json:"activityAfter,omitempty"`
	PeerInvite    string         `json:"peerInvite,omitempty"`
	PeerText      *PeerText      `json:"peerText,omitempty"`
	InviteToken   string         `json:"inviteToken,omitempty"`
	InviteOptions *InviteOptions `json:"inviteOptions,omitempty"`
	Method        string         `json:"method"`
	CommunityID   string         `json:"communityId"`
	After         uint64         `json:"after"`
	ControlAfter  uint64         `json:"controlAfter,omitempty"`
	Head          string         `json:"head"`
	Operation     *Operation     `json:"operation,omitempty"`
}
type Response struct {
	Activities       []chat.Entry      `json:"activities,omitempty"`
	ActivityCursor   uint64            `json:"activityCursor,omitempty"`
	PeerInvites      []string          `json:"peerInvites,omitempty"`
	PeerTexts        []PeerTextEntry   `json:"peerTexts,omitempty"`
	PeerCursor       uint64            `json:"peerCursor,omitempty"`
	InvitationStatus *InvitationStatus `json:"invitationStatus,omitempty"`
	Commits          []Commit          `json:"commits"`
	Sequence         uint64            `json:"sequence"`
	Head             string            `json:"head"`
	Ack              string            `json:"ack"`
	Invite           string            `json:"invite,omitempty"`
	Error            string            `json:"error,omitempty"`
	Code             string            `json:"code,omitempty"`
}
