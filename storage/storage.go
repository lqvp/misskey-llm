package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"misskey-llm/llm"
)

type ConversationHistory struct {
	UserID    string        `json:"userId"`
	Username  string        `json:"username"`
	Messages  []llm.Message `json:"messages"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

type Storage struct {
	dir      string
	mu       sync.RWMutex
	histories map[string]*ConversationHistory
}

func NewStorage(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Storage{
		dir:       dir,
		histories: make(map[string]*ConversationHistory),
	}

	if err := s.loadAll(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) loadAll() error {
	pattern := filepath.Join(s.dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var history ConversationHistory
		if err := json.Unmarshal(data, &history); err != nil {
			continue
		}

		s.histories[history.UserID] = &history
	}

	return nil
}

func (s *Storage) GetHistory(userID string) []llm.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if h, ok := s.histories[userID]; ok {
		return h.Messages
	}
	return nil
}

func (s *Storage) AddMessage(userID, username string, role, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	history, ok := s.histories[userID]
	if !ok {
		history = &ConversationHistory{
			UserID:   userID,
			Username: username,
			Messages: []llm.Message{},
		}
		s.histories[userID] = history
	}

	history.Messages = append(history.Messages, llm.Message{
		Role:    role,
		Content: content,
	})
	history.UpdatedAt = time.Now()

	if len(history.Messages) > 20 {
		history.Messages = history.Messages[len(history.Messages)-20:]
	}

	return s.save(history)
}

func (s *Storage) save(history *ConversationHistory) error {
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.dir, history.UserID+".json")
	return os.WriteFile(path, data, 0644)
}
