package provider

import (
	"time"
)

type Provider interface {
	Login() (string, time.Time, error)
	GetOrganizations() ([]Organization, error)
	GetProjects(organizationID string, includeArchived bool) ([]Project, error)
	ListFiles(organizationID, projectID string) ([]File, error)
	UploadFile(organizationID, projectID, fileName, content string) error
	DeleteFile(organizationID, projectID, fileUUID string) error
	ArchiveProject(organizationID, projectID string) error
	CreateProject(organizationID, name, description string) (Project, error)
	GetChatConversations(organizationID string) ([]ChatConversation, error)
	GetPublishedArtifacts(organizationID string) ([]PublishedArtifact, error)
	GetChatConversation(organizationID, conversationID string) (ChatConversation, error)
	GetArtifactContent(organizationID, artifactUUID string) (string, error)
	DeleteChat(organizationID string, conversationUUIDs []string) error
	CreateChat(organizationID, chatName, projectUUID string) (ChatConversation, error)
	SendMessage(organizationID, chatID, prompt, timezone string) (<-chan MessageEvent, error)
}

type Organization struct {
	ID   string
	Name string
}

type Project struct {
	ID         string
	Name       string
	ArchivedAt *time.Time
}

type File struct {
	UUID      string
	FileName  string
	Content   string
	CreatedAt time.Time
}

type ChatConversation struct {
	UUID      string
	Name      string
	ProjectID string
	Messages  []ChatMessage
}

type ChatMessage struct {
	UUID    string
	Content string
	Sender  string
}

type PublishedArtifact struct {
	UUID    string
	Content string
}

type MessageEvent struct {
	Completion string
	Error      string
	Done       bool
}
