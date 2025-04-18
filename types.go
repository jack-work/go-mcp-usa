package main

import (
	"github.com/google/uuid"
)

type UsaMessage struct {
	ID      string
	Content string
	Role    string
}

type Message interface {
	GetID() string
	GetContent() string
	GetRole() string
}

func (u UsaMessage) GetID() string {
	return u.ID
}

func (u UsaMessage) GetContent() string {
	return u.Content
}

func (u UsaMessage) GetRole() string {
	return u.Role
}

func NewMessage(content string, role string) Message {
	return &UsaMessage{
		ID:      uuid.New().String(),
		Content: content,
		Role:    role,
	}
}
