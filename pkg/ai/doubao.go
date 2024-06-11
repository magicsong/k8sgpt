package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

const arkAIClientName = "arkai"

type ArkAIClient struct {
	nopCloser

	client      *http.Client
	apiKey      string
	endpoint    string
	model       string
	temperature float32
}

type ArkChatCompletionRequest struct {
	Model       string                         `json:"model"`
	Messages    []openai.ChatCompletionMessage `json:"messages"`
	Temperature float32                        `json:"temperature,omitempty"`
	Stream      bool                           `json:"stream"`
}

type ArkChatCompletionResponse struct {
	Choices []struct {
		Message openai.ChatCompletionMessage `json:"message"`
	} `json:"choices"`
}

func (c *ArkAIClient) Configure(config IAIConfig) error {
	c.apiKey = config.GetPassword()
	c.endpoint = config.GetBaseURL()
	c.model = config.GetModel()
	c.temperature = config.GetTemperature()
	c.client = &http.Client{}
	return nil
}

func (c *ArkAIClient) GetCompletion(ctx context.Context, prompt string) (string, error) {
	// Create a completion request
	requestBody := ArkChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Stream: false,
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/v3/chat/completions", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("error making request to Ark AI")
	}

	var completionResponse ArkChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResponse); err != nil {
		return "", err
	}

	if len(completionResponse.Choices) == 0 {
		return "", errors.New("no completion choices returned")
	}

	return completionResponse.Choices[0].Message.Content, nil
}

func (c *ArkAIClient) GetName() string {
	return arkAIClientName
}
