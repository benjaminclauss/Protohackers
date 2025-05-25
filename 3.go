package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"strings"
	"sync"
	"unicode"
)

const DefaultWelcomeMessage = "Welcome to budgetchat! What shall I call you?"

// BudgetChat is a simple TCP-based chat room protocol.
type BudgetChat struct {
	namePromptMessage string
	users             map[string]net.Conn
	mu                sync.Mutex
}

func NewBudgetChat(namePromptMessage string) *BudgetChat {
	return &BudgetChat{
		namePromptMessage: namePromptMessage,
		users:             make(map[string]net.Conn),
	}
}

func (b *BudgetChat) Handle(conn net.Conn) error {
	defer CloseOrLog(conn)

	if _, err := fmt.Fprintln(conn, b.namePromptMessage); err != nil {
		return err
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		// TODO: Handle
		return fmt.Errorf("budgetchat: no user names found")
	}

	name := scanner.Text()
	if err := b.validateAndAddUser(name, conn); err != nil {
		_, err = fmt.Fprintln(conn, "Error:", err)
		if err != nil {
			LogWriteError(err)
		}
		return err
	}
	defer b.disconnect(name)

	b.announcePresence(name)
	if err := b.listAllPresentUserNames(name, conn); err != nil {
		return err
	}

	for scanner.Scan() {
		b.relay(name, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		// TODO: Clean.
		LogReadError(err)
	}
	return nil
}

func (b *BudgetChat) validateAndAddUser(name string, conn net.Conn) error {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return fmt.Errorf("username must be alphanumeric")
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.users[name]; exists {
		return fmt.Errorf("username already taken")
	}

	b.users[name] = conn
	slog.Info("added user", "name", name, "remote_addr", conn.RemoteAddr())
	return nil
}

func (b *BudgetChat) announcePresence(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for otherUser, conn := range b.users {
		if otherUser != name {
			_, err := fmt.Fprintln(conn, "* "+name+" has entered the room")
			if err != nil {
				slog.Warn("error announcing new user", "username", otherUser, "err", err)
			}
		}
	}
}

func (b *BudgetChat) disconnect(user string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for otherUser, conn := range b.users {
		if otherUser != user {
			_, err := fmt.Fprintln(conn, "* "+user+" has left the room")
			if err != nil {
				slog.Warn("error sending disconnect message", "username", otherUser, "err", err)
			}
		}
	}
	delete(b.users, user)
}

func (b *BudgetChat) listAllPresentUserNames(name string, conn net.Conn) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var usernames []string
	for username := range maps.Keys(b.users) {
		if username != name {
			usernames = append(usernames, username)
		}
	}
	_, err := fmt.Fprintln(conn, "* The room contains: "+strings.Join(usernames, ", "))
	return err
}

func (b *BudgetChat) relay(sender string, msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for otherUser, conn := range b.users {
		if otherUser != sender {
			_, err := fmt.Fprintln(conn, "["+sender+"] "+msg)
			if err != nil {
				slog.Warn("error relaying message", "username", otherUser, "err", err)
			}
		}
	}
}
