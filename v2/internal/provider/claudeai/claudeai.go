package claudeai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/internal/provider"
	"github.com/r3labs/sse/v2"
	"github.com/sirupsen/logrus"
)

type ClaudeAI struct {
	config *config.Config
	client *http.Client
	logger *logrus.Logger
}

func NewClaudeAIProvider(cfg *config.Config) *ClaudeAI {
	return &ClaudeAI{
		config: cfg,
		client: &http.Client{},
		logger: logrus.New(),
	}
}

func (c *ClaudeAI) Login() (string, time.Time, error) {
	// Implement login logic here
	// This might involve prompting the user for credentials and making an API call
	// Return the session key and expiry time
	return "", time.Time{}, fmt.Errorf("not implemented")
}

func (c *ClaudeAI) GetOrganizations() ([]provider.Organization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orgs []provider.Organization
	err = json.NewDecoder(resp.Body).Decode(&orgs)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

func (c *ClaudeAI) GetProjects(organizationID string, includeArchived bool) ([]provider.Project, error) {
	url := fmt.Sprintf("/organizations/%s/projects", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var projects []provider.Project
	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, err
	}

	if !includeArchived {
		var activeProjects []provider.Project
		for _, p := range projects {
			if p.ArchivedAt == nil {
				activeProjects = append(activeProjects, p)
			}
		}
		projects = activeProjects
	}

	return projects, nil
}

func (c *ClaudeAI) ListFiles(organizationID, projectID string) ([]provider.File, error) {
	url := fmt.Sprintf("/organizations/%s/projects/%s/docs", organizationID, projectID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var files []provider.File
	err = json.NewDecoder(resp.Body).Decode(&files)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (c *ClaudeAI) UploadFile(organizationID, projectID, fileName, content string) error {
	url := fmt.Sprintf("/organizations/%s/projects/%s/docs", organizationID, projectID)
	data := map[string]string{
		"file_name": fileName,
		"content":   content,
	}
	_, err := c.makeRequest("POST", url, data)
	return err
}

func (c *ClaudeAI) DeleteFile(organizationID, projectID, fileUUID string) error {
	url := fmt.Sprintf("/organizations/%s/projects/%s/docs/%s", organizationID, projectID, fileUUID)
	_, err := c.makeRequest("DELETE", url, nil)
	return err
}

func (c *ClaudeAI) ArchiveProject(organizationID, projectID string) error {
	url := fmt.Sprintf("/organizations/%s/projects/%s", organizationID, projectID)
	data := map[string]bool{
		"is_archived": true,
	}
	_, err := c.makeRequest("PUT", url, data)
	return err
}

func (c *ClaudeAI) CreateProject(organizationID, name, description string) (provider.Project, error) {
	url := fmt.Sprintf("/organizations/%s/projects", organizationID)
	data := map[string]interface{}{
		"name":        name,
		"description": description,
		"is_private":  true,
	}
	resp, err := c.makeRequest("POST", url, data)
	if err != nil {
		return provider.Project{}, err
	}
	defer resp.Body.Close()

	var project provider.Project
	err = json.NewDecoder(resp.Body).Decode(&project)
	if err != nil {
		return provider.Project{}, err
	}

	return project, nil
}

func (c *ClaudeAI) GetChatConversations(organizationID string) ([]provider.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var conversations []provider.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversations)
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (c *ClaudeAI) GetPublishedArtifacts(organizationID string) ([]provider.PublishedArtifact, error) {
	url := fmt.Sprintf("/organizations/%s/published_artifacts", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var artifacts []provider.PublishedArtifact
	err = json.NewDecoder(resp.Body).Decode(&artifacts)
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

func (c *ClaudeAI) GetChatConversation(organizationID, conversationID string) (provider.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations/%s?rendering_mode=raw", organizationID, conversationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return provider.ChatConversation{}, err
	}
	defer resp.Body.Close()

	var conversation provider.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversation)
	if err != nil {
		return provider.ChatConversation{}, err
	}

	return conversation, nil
}

func (c *ClaudeAI) GetArtifactContent(organizationID, artifactUUID string) (string, error) {
	artifacts, err := c.GetPublishedArtifacts(organizationID)
	if err != nil {
		return "", err
	}

	for _, artifact := range artifacts {
		if artifact.UUID == artifactUUID {
			return artifact.Content, nil
		}
	}

	return "", fmt.Errorf("artifact with UUID %s not found", artifactUUID)
}

func (c *ClaudeAI) DeleteChat(organizationID string, conversationUUIDs []string) error {
	url := fmt.Sprintf("/organizations/%s/chat_conversations/delete_many", organizationID)
	data := map[string][]string{
		"conversation_uuids": conversationUUIDs,
	}
	_, err := c.makeRequest("POST", url, data)
	return err
}

func (c *ClaudeAI) CreateChat(organizationID, chatName, projectUUID string) (provider.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations", organizationID)
	data := map[string]string{
		"name":         chatName,
		"project_uuid": projectUUID,
	}
	resp, err := c.makeRequest("POST", url, data)
	if err != nil {
		return provider.ChatConversation{}, err
	}
	defer resp.Body.Close()

	var conversation provider.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversation)
	if err != nil {
		return provider.ChatConversation{}, err
	}

	return conversation, nil
}

func (c *ClaudeAI) SendMessage(organizationID, chatID, prompt, timezone string) (<-chan provider.MessageEvent, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations/%s/completion", organizationID, chatID)
	data := map[string]interface{}{
		"prompt":      prompt,
		"timezone":    timezone,
		"attachments": []string{},
		"files":       []string{},
	}

	eventChan := make(chan provider.MessageEvent)

	go func() {
		defer close(eventChan)

		client := sse.NewClient(url)
		client.Headers["Content-Type"] = "application/json"
		client.Headers["Cookie"] = fmt.Sprintf("sessionKey=%s", c.getSessionKey())

		err := client.SubscribeRaw(func(msg *sse.Event) {
			if msg.Event == "error" {
				eventChan <- provider.MessageEvent{Error: string(msg.Data)}
				return
			}

			if msg.Event == "done" {
				eventChan <- provider.MessageEvent{Done: true}
				return
			}

			var event map[string]interface{}
			err := json.Unmarshal(msg.Data, &event)
			if err != nil {
				c.logger.Errorf("Failed to unmarshal SSE event: %v", err)
				return
			}

			if completion, ok := event["completion"].(string); ok {
				eventChan <- provider.MessageEvent{Completion: completion}
			}
		})

		if err != nil {
			c.logger.Errorf("SSE subscription error: %v", err)
			eventChan <- provider.MessageEvent{Error: err.Error()}
		}
	}()

	return eventChan, nil
}

func (c *ClaudeAI) makeRequest(method, endpoint string, data interface{}) (*http.Response, error) {
	url := c.config.Get("claude_api_url").(string) + endpoint
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", c.getSessionKey()))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

func (c *ClaudeAI) getSessionKey() string {
	sessionKey, _, _ := c.config.GetSessionKey("claude.ai")
	return sessionKey
}
