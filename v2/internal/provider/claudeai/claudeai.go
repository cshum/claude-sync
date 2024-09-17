package claudeai

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/cshum/claude-sync/v2/internal/config"
	"github.com/cshum/claude-sync/v2/providerapi"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/charmap"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode/utf8"
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
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your Claude.ai session key: ")
	sessionKey, err := reader.ReadString('\n')
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read session key: %v", err)
	}
	sessionKey = strings.TrimSpace(sessionKey)

	fmt.Print("Enter the session expiry time (format: RFC3339, e.g., 2023-06-01T15:04:05Z): ")
	expiryStr, err := reader.ReadString('\n')
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read expiry time: %v", err)
	}
	expiryStr = strings.TrimSpace(expiryStr)

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid expiry time format: %v", err)
	}

	// Verify the session key by making a test API call
	err = c.verifySessionKey(sessionKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("session key verification failed: %v", err)
	}

	// Store the session key and expiry in the config
	err = c.config.SetSessionKey("claude.ai", sessionKey, expiry)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to store session key: %v", err)
	}

	return sessionKey, expiry, nil
}

func (c *ClaudeAI) verifySessionKey(sessionKey string) error {
	// Store the current session key
	currentSessionKey, err := c.getSessionKey()
	if err != nil {
		// If there's no current session key, that's fine, we'll just set it back to empty later
		currentSessionKey = ""
	}

	// Temporarily set the new session key
	c.setSessionKey(sessionKey)

	// Make a test API call to verify the session key
	_, err = c.GetOrganizations()

	// Restore the original session key
	c.setSessionKey(currentSessionKey)

	if err != nil {
		return fmt.Errorf("failed to verify session key: %v", err)
	}

	return nil
}

func (c *ClaudeAI) setSessionKey(sessionKey string) {
	c.config.Set("claude_ai_session_key", sessionKey, false)
}

func (c *ClaudeAI) getSessionKey() (string, error) {
	sessionKey, _, err := c.config.GetSessionKey("claude.ai")
	if err != nil {
		return "", fmt.Errorf("session key not found. Please run 'claudesync auth login' first")
	}
	if sessionKey == "" {
		return "", fmt.Errorf("empty session key. Please run 'claudesync auth login' to set a valid session key")
	}
	return sessionKey, nil
}

func (c *ClaudeAI) GetOrganizations() ([]providerapi.Organization, error) {
	body, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, err
	}

	var orgs []providerapi.Organization
	err = json.Unmarshal(body, &orgs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling organizations: %v", err)
	}

	return orgs, nil
}

func (c *ClaudeAI) GetProjects(organizationID string, includeArchived bool) ([]providerapi.Project, error) {
	url := fmt.Sprintf("/organizations/%s/projects", organizationID)
	body, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var projects []providerapi.Project
	err = json.Unmarshal(body, &projects)
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
	body, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var files []providerapi.File
	err = json.Unmarshal(body, &files)
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
	body, err := c.makeRequest("POST", url, data)
	if err != nil {
		return providerapi.Project{}, err
	}

	var project providerapi.Project
	err = json.Unmarshal(body, &project)
	if err != nil {
		return providerapi.Project{}, err
	}

	return project, nil
}

func (c *ClaudeAI) GetChatConversations(organizationID string) ([]providerapi.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations", organizationID)
	body, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var conversations []providerapi.ChatConversation
	err = json.Unmarshal(body, &conversations)
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

func (c *ClaudeAI) GetPublishedArtifacts(organizationID string) ([]providerapi.PublishedArtifact, error) {
	url := fmt.Sprintf("/organizations/%s/published_artifacts", organizationID)
	body, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var artifacts []providerapi.PublishedArtifact
	err = json.Unmarshal(body, &artifacts)
	if err != nil {
		return nil, err
	}

	return artifacts, nil
}

func (c *ClaudeAI) GetChatConversation(organizationID, conversationID string) (providerapi.ChatConversation, error) {
	url := fmt.Sprintf("/organizations/%s/chat_conversations/%s?rendering_mode=raw", organizationID, conversationID)
	body, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return providerapi.ChatConversation{}, err
	}

	var conversation providerapi.ChatConversation
	err = json.Unmarshal(body, &conversation)
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
	body, err := c.makeRequest("POST", url, data)
	if err != nil {
		return providerapi.ChatConversation{}, err
	}

	var conversation providerapi.ChatConversation
	err = json.Unmarshal(body, &conversation)
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
		sessionKey, err := c.getSessionKey()
		if err != nil {
			return
		}
		req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", sessionKey))
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

func (c *ClaudeAI) makeRequest(method, endpoint string, data interface{}) ([]byte, error) {
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

	sessionKey, err := c.getSessionKey()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", sessionKey))

	c.logger.WithFields(logrus.Fields{
		"method":  method,
		"url":     url,
		"headers": req.Header,
	}).Debug("Making request")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.logger.WithFields(logrus.Fields{
		"status":  resp.StatusCode,
		"headers": resp.Header,
	}).Debug("Received response")

	var reader io.Reader
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error creating gzip reader: %v", err)
		}
		defer gzReader.Close()
		reader = gzReader
	} else {
		reader = resp.Body
	}

	respBody, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return nil, c.handleHTTPError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

func (c *ClaudeAI) handleHTTPError(statusCode int, body []byte) error {
	contentStr := string(body)

	// Try to decode as UTF-8
	if !utf8.Valid(body) {
		// If UTF-8 decoding fails, try to decode as ISO-8859-1 (Latin-1)
		decoder := charmap.ISO8859_1.NewDecoder()
		decodedBody, err := decoder.Bytes(body)
		if err != nil {
			c.logger.WithError(err).Error("Failed to decode response body")
		} else {
			contentStr = string(decodedBody)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"status": statusCode,
		"body":   contentStr,
	}).Error("HTTP error")

	if statusCode == 403 {
		return fmt.Errorf("received a 403 Forbidden error. Please check your session key")
	} else if statusCode == 429 {
		var errorData map[string]interface{}
		if err := json.Unmarshal([]byte(contentStr), &errorData); err == nil {
			if errorMsg, ok := errorData["error"].(map[string]interface{})["message"].(string); ok {
				var resetsAt map[string]interface{}
				if err := json.Unmarshal([]byte(errorMsg), &resetsAt); err == nil {
					if resetsAtUnix, ok := resetsAt["resetsAt"].(float64); ok {
						resetsAtTime := time.Unix(int64(resetsAtUnix), 0).UTC()
						return fmt.Errorf("message limit exceeded. Try again after %s", resetsAtTime.Format(time.RFC1123))
					}
				}
			}
		}
		return fmt.Errorf("HTTP 429: Too Many Requests. Failed to parse error response")
	}

	return fmt.Errorf("API request failed with status code %d: %s", statusCode, contentStr)
}
