package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"misskey-llm/bot"
	"misskey-llm/llm"
	"misskey-llm/logger"
	"misskey-llm/misskey"
	"misskey-llm/storage"
)

func main() {
	cfg := LoadConfig()

	if err := logger.Init("log", cfg.LogLevel); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	logger.Info("Starting Misskey LLM Bot...")

	if cfg.MisskeyURL == "" || cfg.MisskeyToken == "" {
		logger.Error("MISSKEY_URL and MISSKEY_TOKEN are required")
		os.Exit(1)
	}
	if cfg.LLMEndpoint == "" || cfg.LLMAPIKey == "" {
		logger.Error("LLM_ENDPOINT and LLM_API_KEY are required")
		os.Exit(1)
	}
	if cfg.LLMModel == "" {
		cfg.LLMModel = "mimo-v2.5-pro"
	}

	storageDir := "data/conversations"
	store, err := storage.NewStorage(storageDir)
	if err != nil {
		logger.Error("Failed to initialize storage: %v", err)
		os.Exit(1)
	}

	llmOpts := []llm.ClientOption{
		llm.WithTemperature(cfg.Temperature),
		llm.WithTopP(cfg.TopP),
		llm.WithSearch(cfg.EnableSearch),
		llm.WithThinking(cfg.EnableThinking),
		llm.WithJinaSearch(),
		llm.WithFetchURL(),
	}

	misskeyClient := misskey.NewClient(cfg.MisskeyURL, cfg.MisskeyToken)
	llmClient := llm.NewClient(cfg.LLMEndpoint, cfg.LLMAPIKey, cfg.LLMModel, cfg.MaxTokens, llmOpts...)

	b := bot.New(misskeyClient, llmClient, store, cfg.SystemPrompt)

	logger.Info("Model: %s, MaxTokens: %d, Temperature: %.2f, TopP: %.2f",
		cfg.LLMModel, cfg.MaxTokens, cfg.Temperature, cfg.TopP)
	logger.Info("Web Search: %v, Thinking: %v", cfg.EnableSearch, cfg.EnableThinking)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			if err := misskeyClient.Connect(); err != nil {
				logger.Error("Connection error: %v", err)
				logger.Info("Reconnecting in 5 seconds...")
				time.Sleep(5 * time.Second)
				continue
			}

			logger.Info("Listening for events...")
			if err := misskeyClient.ReadEvents(
				b.HandleNotification,
				b.HandleMention,
				b.HandleReply,
			); err != nil {
				logger.Error("Stream error: %v", err)
				logger.Info("Reconnecting in 3 seconds...")
				time.Sleep(3 * time.Second)
			}
		}
	}()

	sig := <-sigChan
	logger.Info("Received signal %v, shutting down...", sig)
	misskeyClient.Close()
	logger.Info("Bot stopped")
}
