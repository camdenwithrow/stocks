package common

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var log = NewLogger()

const (
	VersionSize   = 1
	TypeSize      = 1
	LengthSize    = 2
	TimestampSize = 8
	HeaderSize    = VersionSize + TypeSize + LengthSize + TimestampSize
)

const VERSION = 1

const (
	MESSAGE_PACKET = iota + 1
	I
)

type Packet struct {
	Version   byte
	Type      byte
	Length    uint16
	Timestamp uint64
	Data      []byte
}

func (p Packet) Serialize() ([]byte, error) {
	buf := make([]byte, HeaderSize+p.Length)
	buf[0] = p.Version
	buf[1] = p.Type
	binary.BigEndian.PutUint16(buf[2:4], p.Length)
	binary.BigEndian.PutUint64(buf[4:12], p.Timestamp)
	n := copy(buf[12:], p.Data)
	if n != int(p.Length) {
		return nil, errors.New("Failed to copy all of data into packet")
	}
	return buf, nil
}

func Deserialize(buf []byte, n int) (*Packet, error) {
	version := buf[0]
	if version != VERSION {
		return nil, errors.New(fmt.Sprintf("Error reading packet. Incorrect version. Expected: %d Got: %d", VERSION, version))
	}

	length := binary.BigEndian.Uint16(buf[2:4])
	if int(length) != (n - HeaderSize) {
		return nil, errors.New(fmt.Sprintf("Incorrect length. Expected: %d Got: %d", n-HeaderSize, length))
	}

	packet := &Packet{
		Version:   version,
		Type:      buf[1],
		Length:    length,
		Timestamp: binary.BigEndian.Uint64(buf[4:12]),
		Data:      buf[HeaderSize:n],
	}
	return packet, nil
}
