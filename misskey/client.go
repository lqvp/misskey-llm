package misskey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"misskey-llm/logger"

	"github.com/gorilla/websocket"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	wsConn     *websocket.Conn
	wsMu       sync.Mutex
}

type NotificationHandler func(Notification)
type NoteHandler func(Note)

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) wsURL() string {
	u, _ := url.Parse(c.baseURL)
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = "/streaming"
	q := u.Query()
	q.Set("i", c.token)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) Connect() error {
	wsURL := c.wsURL()
	logger.Info("Connecting to Misskey WebSocket: %s", c.baseURL+"/streaming")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	c.wsConn = conn

	connectMsg := map[string]interface{}{
		"type": "connect",
		"body": map[string]interface{}{
			"channel": "main",
			"id":      "main",
		},
	}

	if err := conn.WriteJSON(connectMsg); err != nil {
		conn.Close()
		return fmt.Errorf("send connect message: %w", err)
	}

	logger.Info("Connected to Misskey streaming API (main channel)")
	return nil
}

func (c *Client) ReadEvents(notifHandler NotificationHandler, mentionHandler NoteHandler, replyHandler NoteHandler) error {
	for {
		_, message, err := c.wsConn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		logger.Info("Raw WebSocket message: %s", string(message))

		var msg StreamingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Info("Failed to unmarshal message: %v", err)
			continue
		}

		if msg.Type != "channel" {
			continue
		}

		var channelMsg ChannelMessage
		if err := json.Unmarshal(msg.Body, &channelMsg); err != nil {
			logger.Info("Failed to unmarshal channel message: %v", err)
			continue
		}

		if channelMsg.ID != "main" {
			continue
		}

		logger.Info("Channel message type: %s, body: %s", channelMsg.Type, string(channelMsg.Body))

		switch channelMsg.Type {
		case "notification":
			var notification Notification
			if err := json.Unmarshal(channelMsg.Body, &notification); err != nil {
				logger.Info("Failed to unmarshal notification: %v", err)
				continue
			}
			if notifHandler != nil {
				go notifHandler(notification)
			}

		case "mention":
			var note Note
			if err := json.Unmarshal(channelMsg.Body, &note); err != nil {
				logger.Info("Failed to unmarshal mention: %v", err)
				continue
			}
			if mentionHandler != nil {
				go mentionHandler(note)
			}

		case "reply":
			var note Note
			if err := json.Unmarshal(channelMsg.Body, &note); err != nil {
				logger.Info("Failed to unmarshal reply: %v", err)
				continue
			}
			if replyHandler != nil {
				go replyHandler(note)
			}

		case "renote":
			// renoteは無視（quoteの場合はnotificationで来る）
			continue
		}
	}
}

func (c *Client) CreateNote(text, replyID, renoteID string) error {
	reqBody := map[string]interface{}{
		"i":    c.token,
		"text": text,
	}
	if replyID != "" {
		reqBody["replyId"] = replyID
	}
	if renoteID != "" {
		reqBody["renoteId"] = renoteID
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	apiURL := c.baseURL + "/api/notes/create"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create note request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("create note failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) GetConversation(noteID string, limit int) ([]Note, error) {
	reqBody := map[string]interface{}{
		"noteId": noteID,
		"limit":  limit,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := c.baseURL + "/api/notes/conversation"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get conversation request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get conversation failed (status %d): %s", resp.StatusCode, string(body))
	}

	var notes []Note
	if err := json.Unmarshal(body, &notes); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return notes, nil
}

func (c *Client) GetNote(noteID string) (*Note, error) {
	reqBody := map[string]interface{}{
		"noteId": noteID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := c.baseURL + "/api/notes/show"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get note request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get note failed (status %d): %s", resp.StatusCode, string(body))
	}

	var note Note
	if err := json.Unmarshal(body, &note); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &note, nil
}

func (c *Client) Reconnect() error {
	c.Close()
	time.Sleep(2 * time.Second)
	return c.Connect()
}

func (c *Client) Close() {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.wsConn != nil {
		c.wsConn.Close()
		c.wsConn = nil
	}
}
