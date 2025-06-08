package linereversal

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := map[string]struct {
		m        string
		expected Message
	}{
		"connect": {
			m:        "/connect/1234567/",
			expected: &ConnectMessage{SessionToken: 1234567},
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
