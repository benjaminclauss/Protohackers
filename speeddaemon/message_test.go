package speeddaemon

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessage_MarshalBinary(t *testing.T) {
	tests := []struct {
		given    *ErrorMessage
		expected []byte
	}{
		{
			given:    &ErrorMessage{Msg: "bad"},
			expected: []byte{0x10, 0x03, 0x62, 0x61, 0x64},
		},
		{
			given:    &ErrorMessage{Msg: "illegal msg"},
			expected: []byte{0x10, 0x0b, 0x69, 0x6c, 0x6c, 0x65, 0x67, 0x61, 0x6c, 0x20, 0x6d, 0x73, 0x67},
		},
	}

	for _, test := range tests {
		data, err := test.given.MarshalBinary()
		assert.NoError(t, err)
		assert.Equal(t, test.expected, data)
	}
}

func Test_readPlateMessage(t *testing.T) {
	tests := []struct {
		given    []byte
		expected *PlateMessage
	}{
		{
			given:    []byte{0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x00, 0x03, 0xe8},
			expected: &PlateMessage{Plate: "UN1X", Timestamp: 1000},
		},
		{
			given:    []byte{0x07, 0x52, 0x45, 0x30, 0x35, 0x42, 0x4b, 0x47, 0x00, 0x01, 0xe2, 0x40},
			expected: &PlateMessage{Plate: "RE05BKG", Timestamp: 123456},
		},
	}

	for _, test := range tests {
		msg, err := readPlateMessage(bytes.NewReader(test.given))
		assert.NoError(t, err)
		assert.Equal(t, test.expected, msg)
	}
}

func TestTicketMessage_MarshalBinary(t *testing.T) {
	tests := []struct {
		given    *TicketMessage
		expected []byte
	}{
		{
			given: &TicketMessage{
				Plate:      "UN1X",
				Road:       66,
				Mile1:      100,
				Timestamp1: 123456,
				Mile2:      110,
				Timestamp2: 123816,
				Speed:      10000,
			},
			expected: []byte{0x21, 0x04, 0x55, 0x4e, 0x31, 0x58, 0x00, 0x42, 0x00, 0x64, 0x00, 0x01, 0xe2, 0x40, 0x00, 0x6e, 0x00, 0x01, 0xe3, 0xa8, 0x27, 0x10},
		},
		{
			given: &TicketMessage{
				Plate:      "RE05BKG",
				Road:       368,
				Mile1:      1234,
				Timestamp1: 1000000,
				Mile2:      1235,
				Timestamp2: 1000060,
				Speed:      6000,
			},
			expected: []byte{0x21, 0x07, 0x52, 0x45, 0x30, 0x35, 0x42, 0x4b, 0x47, 0x01, 0x70, 0x04, 0xd2, 0x00, 0x0f, 0x42, 0x40, 0x04, 0xd3, 0x00, 0x0f, 0x42, 0x7c, 0x17, 0x70},
		},
	}

	for _, test := range tests {
		data, err := test.given.MarshalBinary()
		assert.NoError(t, err)
		assert.Equal(t, test.expected, data)
	}
}

func Test_readWantHeartbeatMessage(t *testing.T) {
	tests := []struct {
		given    []byte
		expected *WantHeartbeatMessage
	}{
		{
			given:    []byte{0x00, 0x00, 0x00, 0x0a},
			expected: &WantHeartbeatMessage{Interval: 10}},
		{
			given:    []byte{0x00, 0x00, 0x04, 0xdb},
			expected: &WantHeartbeatMessage{Interval: 1243},
		},
	}

	for _, test := range tests {
		msg, err := readWantHeartbeatMessage(bytes.NewReader(test.given))
		assert.NoError(t, err)
		assert.Equal(t, test.expected, msg)
	}
}

func Test_HeartbeatMessage_MarshalBinary(t *testing.T) {
	msg := &HeartbeatMessage{}
	data, err := msg.MarshalBinary()
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x41}, data)
}

func Test_readIAmCameraMessage(t *testing.T) {
	tests := []struct {
		given    []byte
		expected *IAmCameraMessage
	}{
		{
			given:    []byte{0x00, 0x42, 0x00, 0x64, 0x00, 0x3c},
			expected: &IAmCameraMessage{Road: 66, Mile: 100, Limit: 60},
		},
		{
			given:    []byte{0x01, 0x70, 0x04, 0xd2, 0x00, 0x28},
			expected: &IAmCameraMessage{Road: 368, Mile: 1234, Limit: 40},
		},
	}

	for _, test := range tests {
		msg, err := readIAmCameraMessage(bytes.NewReader(test.given))
		assert.NoError(t, err)
		assert.Equal(t, test.expected, msg)
	}
}

func Test_readIAmDispatcherMessage(t *testing.T) {
	tests := []struct {
		given    []byte
		expected *IAmDispatcherMessage
	}{
		{
			given:    []byte{0x01, 0x00, 0x42},
			expected: &IAmDispatcherMessage{Roads: []uint16{66}},
		},
		{
			given:    []byte{0x03, 0x00, 0x42, 0x01, 0x70, 0x13, 0x88},
			expected: &IAmDispatcherMessage{Roads: []uint16{66, 368, 5000}},
		},
	}

	for _, test := range tests {
		msg, err := readIAmDispatcherMessage(bytes.NewReader(test.given))
		assert.NoError(t, err)
		assert.Equal(t, test.expected, msg)
	}
}
