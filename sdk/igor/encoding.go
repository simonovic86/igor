package igor

import (
	"encoding/binary"
	"errors"
)

// errShortRead is returned by Decoder when the input data is too short.
var errShortRead = errors.New("igor: short read during decode")

// Encoder helps serialize agent state for checkpointing.
// Uses little-endian byte order, matching Igor checkpoint format.
// Methods are chainable for concise usage:
//
//	igor.NewEncoder(28).Uint64(count).Int64(ts).Uint32(flags).Finish()
type Encoder struct {
	buf []byte
}

// NewEncoder creates a new encoder with pre-allocated capacity.
func NewEncoder(capacity int) *Encoder {
	return &Encoder{buf: make([]byte, 0, capacity)}
}

// Uint64 appends a uint64 value (8 bytes, little-endian).
func (e *Encoder) Uint64(v uint64) *Encoder {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	e.buf = append(e.buf, b[:]...)
	return e
}

// Int64 appends an int64 value (8 bytes, little-endian).
func (e *Encoder) Int64(v int64) *Encoder {
	return e.Uint64(uint64(v))
}

// Uint32 appends a uint32 value (4 bytes, little-endian).
func (e *Encoder) Uint32(v uint32) *Encoder {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	e.buf = append(e.buf, b[:]...)
	return e
}

// Int32 appends an int32 value (4 bytes, little-endian).
func (e *Encoder) Int32(v int32) *Encoder {
	return e.Uint32(uint32(v))
}

// Bytes appends a length-prefixed byte slice (4-byte length + data).
func (e *Encoder) Bytes(v []byte) *Encoder {
	e.Uint32(uint32(len(v)))
	e.buf = append(e.buf, v...)
	return e
}

// String appends a length-prefixed string (4-byte length + UTF-8 data).
func (e *Encoder) String(v string) *Encoder {
	return e.Bytes([]byte(v))
}

// Bool appends a boolean as a single byte (0 or 1).
func (e *Encoder) Bool(v bool) *Encoder {
	if v {
		e.buf = append(e.buf, 1)
	} else {
		e.buf = append(e.buf, 0)
	}
	return e
}

// Finish returns the encoded bytes.
func (e *Encoder) Finish() []byte {
	return e.buf
}

// Decoder helps deserialize agent state during resume.
// Methods return zero values on short reads; check Err() after decoding.
type Decoder struct {
	data []byte
	pos  int
	err  error
}

// NewDecoder creates a decoder over the given bytes.
func NewDecoder(data []byte) *Decoder {
	return &Decoder{data: data}
}

// Uint64 reads a uint64 value (8 bytes, little-endian).
func (d *Decoder) Uint64() uint64 {
	if d.err != nil || d.pos+8 > len(d.data) {
		d.err = errShortRead
		return 0
	}
	v := binary.LittleEndian.Uint64(d.data[d.pos : d.pos+8])
	d.pos += 8
	return v
}

// Int64 reads an int64 value (8 bytes, little-endian).
func (d *Decoder) Int64() int64 {
	return int64(d.Uint64())
}

// Uint32 reads a uint32 value (4 bytes, little-endian).
func (d *Decoder) Uint32() uint32 {
	if d.err != nil || d.pos+4 > len(d.data) {
		d.err = errShortRead
		return 0
	}
	v := binary.LittleEndian.Uint32(d.data[d.pos : d.pos+4])
	d.pos += 4
	return v
}

// Int32 reads an int32 value (4 bytes, little-endian).
func (d *Decoder) Int32() int32 {
	return int32(d.Uint32())
}

// Bytes reads a length-prefixed byte slice.
func (d *Decoder) Bytes() []byte {
	length := d.Uint32()
	if d.err != nil {
		return nil
	}
	if d.pos+int(length) > len(d.data) {
		d.err = errShortRead
		return nil
	}
	v := make([]byte, length)
	copy(v, d.data[d.pos:d.pos+int(length)])
	d.pos += int(length)
	return v
}

// String reads a length-prefixed string.
func (d *Decoder) String() string {
	b := d.Bytes()
	if d.err != nil {
		return ""
	}
	return string(b)
}

// Bool reads a boolean (single byte, 0 = false).
func (d *Decoder) Bool() bool {
	if d.err != nil || d.pos >= len(d.data) {
		d.err = errShortRead
		return false
	}
	v := d.data[d.pos] != 0
	d.pos++
	return v
}

// Err returns the first error encountered during decoding, or nil.
func (d *Decoder) Err() error {
	return d.err
}
