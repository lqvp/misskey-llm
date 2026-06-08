package misskey

import (
	"encoding/json"
	"time"
)

type StreamingMessage struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body,omitempty"`
}

type ConnectMessage struct {
	Type string        `json:"type"`
	Body ConnectBody   `json:"body"`
}

type ConnectBody struct {
	Channel string         `json:"channel"`
	ID      string         `json:"id"`
	Params  ConnectParams  `json:"params"`
}

type ConnectParams struct {
	Token string `json:"token"`
}

type ChannelMessage struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	User      *User     `json:"user,omitempty"`
	Note      *Note     `json:"note,omitempty"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Host     string `json:"host,omitempty"`
}

type Note struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CW        string    `json:"cw,omitempty"`
	UserID    string    `json:"userId"`
	User      *User     `json:"user,omitempty"`
	ReplyID   string    `json:"replyId,omitempty"`
	RenoteID  string    `json:"renoteId,omitempty"`
	Renote    *Note     `json:"renote,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type NoteCreateRequest struct {
	I        string `json:"i"`
	Text     string `json:"text,omitempty"`
	ReplyID  string `json:"replyId,omitempty"`
	RenoteID string `json:"renoteId,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}
