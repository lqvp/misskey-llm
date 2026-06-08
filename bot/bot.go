package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"misskey-llm/jina"
	"misskey-llm/llm"
	"misskey-llm/logger"
	"misskey-llm/misskey"
	"misskey-llm/storage"
)

type Bot struct {
	misskeyClient *misskey.Client
	llmClient     *llm.Client
	jinaClient    *jina.Client
	storage       *storage.Storage
	systemPrompt  string
	processedMu   sync.Mutex
	processed     map[string]bool
}

func New(
	misskeyClient *misskey.Client,
	llmClient *llm.Client,
	storage *storage.Storage,
	systemPrompt string,
) *Bot {
	return &Bot{
		misskeyClient: misskeyClient,
		llmClient:     llmClient,
		jinaClient:    jina.NewClient(),
		storage:       storage,
		systemPrompt:  systemPrompt,
		processed:     make(map[string]bool),
	}
}

func (b *Bot) getToolHandlers() map[string]llm.ToolHandler {
	return map[string]llm.ToolHandler{
		"jina_search": func(args string) (string, error) {
			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			logger.Info("Jina Search: %s", params.Query)
			return b.jinaClient.Search(params.Query)
		},
		"fetch_url": func(args string) (string, error) {
			var params struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal([]byte(args), &params); err != nil {
				return "", fmt.Errorf("parse args: %w", err)
			}
			logger.Info("Fetch URL: %s", params.URL)
			return b.jinaClient.FetchURL(params.URL)
		},
	}
}

func (b *Bot) isProcessed(noteID string) bool {
	b.processedMu.Lock()
	defer b.processedMu.Unlock()
	return b.processed[noteID]
}

func (b *Bot) markProcessed(noteID string) {
	b.processedMu.Lock()
	defer b.processedMu.Unlock()
	b.processed[noteID] = true
	if len(b.processed) > 1000 {
		b.processed = make(map[string]bool)
	}
}

func (b *Bot) mentionUser(user *misskey.User) string {
	if user.Host != "" {
		return fmt.Sprintf("@%s@%s", user.Username, user.Host)
	}
	return fmt.Sprintf("@%s", user.Username)
}

func (b *Bot) HandleNotification(n misskey.Notification) {
	switch n.Type {
	case "quote":
		if n.Note != nil && n.User != nil {
			b.handleQuote(n.Note, n.User)
		}
	case "mention", "reply":
		return
	default:
		logger.Debug("Notification: %s", n.Type)
	}
}

func (b *Bot) HandleMention(note misskey.Note) {
	if note.User == nil {
		return
	}
	if b.isProcessed(note.ID) {
		return
	}
	b.markProcessed(note.ID)
	b.handleMention(&note, note.User)
}

func (b *Bot) HandleReply(note misskey.Note) {
	if note.User == nil {
		return
	}
	if b.isProcessed(note.ID) {
		return
	}
	b.markProcessed(note.ID)
	b.handleReply(&note, note.User)
}

func (b *Bot) handleMention(note *misskey.Note, user *misskey.User) {
	userText := note.Text
	if userText == "" {
		userText = note.CW
	}
	if userText == "" {
		return
	}

	logger.Info("Mention from @%s: %s", user.Username, userText)

	renoteContext := b.extractRenoteContext(note)
	b.processAndReply(user, note.ID, userText, renoteContext)
}

func (b *Bot) handleReply(note *misskey.Note, user *misskey.User) {
	userText := note.Text
	if userText == "" {
		userText = note.CW
	}
	if userText == "" {
		return
	}

	logger.Info("Reply from @%s: %s", user.Username, userText)

	renoteContext := b.extractRenoteContext(note)
	b.processAndReply(user, note.ID, userText, renoteContext)
}

func (b *Bot) handleQuote(note *misskey.Note, user *misskey.User) {
	if b.isProcessed(note.ID) {
		return
	}
	b.markProcessed(note.ID)

	userText := note.Text
	if userText == "" {
		return
	}

	logger.Info("Quote from @%s: %s", user.Username, userText)

	cleanText := strings.ReplaceAll(userText, "@", "")
	cleanText = strings.TrimSpace(cleanText)

	renoteContext := b.extractRenoteContext(note)

	if cleanText == "" {
		cleanText = "この投稿についてどう思う？"
	}

	prompt := cleanText
	if renoteContext != "" {
		prompt = cleanText + renoteContext
	}

	b.processAndReply(user, note.ID, prompt, "")
}

func (b *Bot) extractRenoteContext(note *misskey.Note) string {
	if note.Renote == nil {
		return ""
	}

	originalNote := note.Renote
	originalText := originalNote.Text
	if originalText == "" {
		originalText = originalNote.CW
	}
	if originalText == "" {
		return ""
	}

	if originalNote.User != nil {
		mention := b.mentionUser(originalNote.User)
		logger.Info("Found renote context from %s: %s", mention, originalText)
		return fmt.Sprintf("\n\n--- Referenced Post ---\n%s:\n%s", mention, originalText)
	}

	logger.Info("Found renote context: %s", originalText)
	return fmt.Sprintf("\n\n--- Referenced Post ---\n%s", originalText)
}

func (b *Bot) processAndReply(user *misskey.User, noteID string, userText string, extraContext string) {
	userID := user.ID
	mention := b.mentionUser(user)

	prompt := userText
	if extraContext != "" {
		prompt = userText + extraContext
	}

	urls := b.jinaClient.ExtractURLs(prompt)
	if len(urls) > 0 {
		logger.Info("Found %d URL(s), fetching content...", len(urls))
		urlContext := b.jinaClient.BuildURLContext(urls)
		if urlContext != "" {
			prompt = prompt + urlContext
			logger.Info("Fetched URL content (%d chars)", len(urlContext))
		}
	}

	history := b.storage.GetHistory(userID)

	response, err := b.llmClient.ChatWithTools(b.systemPrompt, history, prompt, b.getToolHandlers())
	if err != nil {
		logger.Error("LLM error: %v", err)
		response = "ごめん、ちょっと考えがまとまらなかった… 😅"
		replyText := fmt.Sprintf("%s %s", mention, response)
		b.misskeyClient.CreateNote(replyText, noteID, "")
		return
	}

	if err := b.storage.AddMessage(userID, user.Username, "user", userText); err != nil {
		logger.Error("Failed to save user message: %v", err)
	}
	if err := b.storage.AddMessage(userID, user.Username, "assistant", response); err != nil {
		logger.Error("Failed to save assistant message: %v", err)
	}

	replyText := fmt.Sprintf("%s %s", mention, response)
	if err := b.misskeyClient.CreateNote(replyText, noteID, ""); err != nil {
		logger.Error("Failed to create note: %v", err)
	} else {
		logger.Info("Replied to %s", mention)
	}
}
