package claudeai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/providerapi"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
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

func (c *ClaudeAI) GetOrganizations() ([]providerapi.Organization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orgs []providerapi.Organization
	err = json.NewDecoder(resp.Body).Decode(&orgs)
	if err != nil {
		return nil, err
	}

	return orgs, nil
}

func (c *ClaudeAI) GetProjects(organizationID string, includeArchived bool) ([]providerapi.Project, error) {
	url := fmt.Sprintf("/organizations/%s/projects", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var projects []providerapi.Project
	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, err
	}

	if !includeArchived {
		var activeProjects []providerapi.Project
		for _, p := range projects {
			if p.ArchivedAt == nil {
				activeProjects = append(activeProjects, p)
			}
		}
		projects = activeProjects
	}

	return projects, nil
}

func (c *ClaudeAI) ListFiles(organizationID, projectID string) ([]providerapi.File, error) {
	url := fmt.Sprintf("/organizations/%s/projects/%s/docs", organizationID, projectID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var files []providerapi.File
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

func (c *ClaudeAI) CreateProject(organizationID, name, description string) (providerapi.Project, error) {
	url := fmt.Sprintf("/organizations/%s/projects", organizationID)
	data := map[string]interface{}{
		"name":        name,
		"description": description,
		"is_private":  true,
	}
	resp, err := c.makeRequest("POST", url, data)
	if err != nil {
		return providerapi.Project{}, err
	}
	defer resp.Body.Close()

	var project providerapi.Project
	err = json.NewDecoder(resp.Body).Decode(&project)
	if err != nil {
		return providerapi.Project{}, err
	}

	return project, nil
}

func (c *ClaudeAI) GetChatConversations(organizationID string) ([]providerapi.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var conversations []providerapi.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversations)
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (c *ClaudeAI) GetPublishedArtifacts(organizationID string) ([]providerapi.PublishedArtifact, error) {
	url := fmt.Sprintf("/organizations/%s/published_artifacts", organizationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var artifacts []providerapi.PublishedArtifact
	err = json.NewDecoder(resp.Body).Decode(&artifacts)
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

func (c *ClaudeAI) GetChatConversation(organizationID, conversationID string) (providerapi.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations/%s?rendering_mode=raw", organizationID, conversationID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return providerapi.ChatConversation{}, err
	}
	defer resp.Body.Close()

	var conversation providerapi.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversation)
	if err != nil {
		return providerapi.ChatConversation{}, err
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

func (c *ClaudeAI) CreateChat(organizationID, chatName, projectUUID string) (providerapi.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations", organizationID)
	data := map[string]string{
		"name":         chatName,
		"project_uuid": projectUUID,
	}
	resp, err := c.makeRequest("POST", url, data)
	if err != nil {
		return providerapi.ChatConversation{}, err
	}
	defer resp.Body.Close()

	var conversation providerapi.ChatConversation
	err = json.NewDecoder(resp.Body).Decode(&conversation)
	if err != nil {
		return providerapi.ChatConversation{}, err
	}

	return conversation, nil
}

func (c *ClaudeAI) SendMessage(organizationID, chatID, prompt, timezone string) (<-chan providerapi.MessageEvent, error) {
	url := fmt.Sprintf("%s/organizations/%s/chat_conversations/%s/completion", c.config.Get("claude_api_url").(string), organizationID, chatID)
	data := map[string]interface{}{
		"prompt":      prompt,
		"timezone":    timezone,
		"attachments": []string{},
		"files":       []string{},
	}

	eventChan := make(chan providerapi.MessageEvent)

	go func() {
		defer close(eventChan)

		jsonData, err := json.Marshal(data)
		if err != nil {
			eventChan <- providerapi.MessageEvent{Error: err.Error()}
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			eventChan <- providerapi.MessageEvent{Error: err.Error()}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", c.getSessionKey()))
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.client.Do(req)
		if err != nil {
			eventChan <- providerapi.MessageEvent{Error: err.Error()}
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					eventChan <- providerapi.MessageEvent{Done: true}
					break
				}

				var eventData map[string]interface{}
				err = json.Unmarshal([]byte(data), &eventData)
				if err != nil {
					c.logger.Errorf("Failed to unmarshal SSE event: %v", err)
					continue
				}

				if completion, ok := eventData["completion"].(string); ok {
					eventChan <- providerapi.MessageEvent{Completion: completion}
				}
			} else if strings.HasPrefix(line, "event: error") {
				scanner.Scan() // Move to the next line which should contain the error data
				errorData := strings.TrimPrefix(scanner.Text(), "data: ")
				eventChan <- providerapi.MessageEvent{Error: errorData}
			}
		}

		if err := scanner.Err(); err != nil {
			eventChan <- providerapi.MessageEvent{Error: err.Error()}
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
