package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const cozeBotClientName = "CozeBotClient"

type CozeBotClient struct {
	nopCloser

	client         *http.Client
	baseURL        string
	token          string
	botID          string
	conversationID string
}

type CozeBotConfig struct {
	BaseURL        string
	Token          string
	BotID          string
	ConversationID string
}

func (c *CozeBotClient) Configure(config IAIConfig) error {
	token := config.GetPassword()
	baseURL := config.GetBaseURL()
	botID := config.GetExtraConfig()["bot_id"]
	conversationID := "123457"

	if baseURL == "" || token == "" || botID == "" || conversationID == "" {
		return errors.New("missing required configuration values")
	}

	c.client = &http.Client{}
	c.baseURL = baseURL
	c.token = token
	c.botID = botID
	c.conversationID = conversationID
	return nil
}

type CozeBotMessage struct {
	Role        string `json:"role"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

type CozeBotResponse struct {
	Messages       []CozeBotMessage `json:"messages"`
	ConversationID string           `json:"conversation_id"`
	Code           int              `json:"code"`
	Msg            string           `json:"msg"`
}

func (c *CozeBotClient) GetCompletion(ctx context.Context, query string) (string, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"conversation_id": c.conversationID,
		"bot_id":          c.botID,
		"user":            "12333333",
		"query":           query,
		"stream":          false,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/open_api/v2/chat", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Host", c.baseURL[7:])
	req.Header.Set("Connection", "keep-alive")
	var lastError error
	for retries := 0; retries < 3; retries++ {
		response, err := func() (string, error) {
			resp, err := c.client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				serverError := errors.New("received non-OK response status: " + resp.Status)
				return "", serverError
			}

			var response CozeBotResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return "", err
			}
			//fmt.Printf("reponse is %v", response)
			if response.Code != 0 {
				if strings.Contains(response.Msg, "There are too many users now") {
					fmt.Println("Too many users, retrying...")
					time.Sleep(2 * time.Second)
					return "", errors.New("too many users")
				}
				return "", errors.New("received error from server: " + response.Msg)
			}

			for _, message := range response.Messages {
				if message.Role == "assistant" && message.Type == "answer" {
					return message.Content, nil
				}
			}

			return "", errors.New("no answer found in response")
		}()
		if err != nil {
			lastError = err
		} else {
			return response, nil
		}
	}
	return "", lastError
}

func (c *CozeBotClient) GetName() string {
	return cozeBotClientName
}
