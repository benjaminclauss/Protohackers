package linereversal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessage(t *testing.T) {
	tests := map[string]struct {
		m        string
		expected Message
	}{
		"connect": {
			m:        "/connect/1234567/",
			expected: &ConnectMessage{Session: 1234567},
		},
		"hello": {
			m:        "/data/1234567/0/hello/",
			expected: &DataMessage{1234567, 0, "hello"},
		},
		"ack": {
			m:        "/ack/1234567/1024/",
			expected: &AckMessage{1234567, 1024},
		},
		"close": {
			m:        "/close/1234567/",
			expected: &CloseMessage{1234567},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			message, err := ParseMessage(test.m)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, message)
		})
	}
}
