package main

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MisskeyURL      string
	MisskeyToken    string
	LLMEndpoint     string
	LLMAPIKey       string
	LLMModel        string
	MaxTokens       int
	Temperature     float64
	TopP            float64
	EnableSearch    bool
	EnableThinking  bool
	LogLevel        string
	SystemPrompt    string
}

func LoadConfig() *Config {
	godotenv.Load()

	maxTokens := 1024
	if v := os.Getenv("MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxTokens = n
		}
	}

	temperature := 1.0
	if v := os.Getenv("TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			temperature = f
		}
	}

	topP := 0.95
	if v := os.Getenv("TOP_P"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			topP = f
		}
	}

	enableSearch := false
	if v := os.Getenv("ENABLE_SEARCH"); v == "true" || v == "1" {
		enableSearch = true
	}

	enableThinking := true
	if v := os.Getenv("ENABLE_THINKING"); v == "false" || v == "0" {
		enableThinking = false
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	systemPrompt := os.Getenv("SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = "あなたは親しみやすいMisskeyボットです。カジュアルでフレンドリーな口調で返答してください。絵文字も適度に使ってください。簡潔に返答してください。"
	}

	return &Config{
		MisskeyURL:     os.Getenv("MISSKEY_URL"),
		MisskeyToken:   os.Getenv("MISSKEY_TOKEN"),
		LLMEndpoint:    os.Getenv("LLM_ENDPOINT"),
		LLMAPIKey:      os.Getenv("LLM_API_KEY"),
		LLMModel:       os.Getenv("LLM_MODEL"),
		MaxTokens:      maxTokens,
		Temperature:    temperature,
		TopP:           topP,
		EnableSearch:   enableSearch,
		EnableThinking: enableThinking,
		LogLevel:       logLevel,
		SystemPrompt:   systemPrompt,
	}
}
