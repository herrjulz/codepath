package reports

import (
	"crypto/rand"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewID returns a 26-character ULID-style identifier.
func NewID() (string, error) {
	var raw [16]byte
	ms := uint64(time.Now().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	if _, err := rand.Read(raw[6:]); err != nil {
		return "", err
	}
	return encodeID(raw), nil
}

func encodeID(id [16]byte) string {
	dst := [26]byte{
		crockford[(id[0]&224)>>5],
		crockford[id[0]&31],
		crockford[(id[1]&248)>>3],
		crockford[((id[1]&7)<<2)|((id[2]&192)>>6)],
		crockford[(id[2]&62)>>1],
		crockford[((id[2]&1)<<4)|((id[3]&240)>>4)],
		crockford[((id[3]&15)<<1)|((id[4]&128)>>7)],
		crockford[(id[4]&124)>>2],
		crockford[((id[4]&3)<<3)|((id[5]&224)>>5)],
		crockford[id[5]&31],
		crockford[(id[6]&248)>>3],
		crockford[((id[6]&7)<<2)|((id[7]&192)>>6)],
		crockford[(id[7]&62)>>1],
		crockford[((id[7]&1)<<4)|((id[8]&240)>>4)],
		crockford[((id[8]&15)<<1)|((id[9]&128)>>7)],
		crockford[(id[9]&124)>>2],
		crockford[((id[9]&3)<<3)|((id[10]&224)>>5)],
		crockford[id[10]&31],
		crockford[(id[11]&248)>>3],
		crockford[((id[11]&7)<<2)|((id[12]&192)>>6)],
		crockford[(id[12]&62)>>1],
		crockford[((id[12]&1)<<4)|((id[13]&240)>>4)],
		crockford[((id[13]&15)<<1)|((id[14]&128)>>7)],
		crockford[(id[14]&124)>>2],
		crockford[((id[14]&3)<<3)|((id[15]&224)>>5)],
		crockford[id[15]&31],
	}
	return string(dst[:])
}

// ValidID reports whether id is a server-generated report id.
func ValidID(id string) bool {
	if len(id) != 26 {
		return false
	}
	for _, r := range id {
		if !isCrockford(r) {
			return false
		}
	}
	return true
}

func isCrockford(r rune) bool {
	switch {
	case r >= '0' && r <= '9':
		return true
	case r >= 'A' && r <= 'H':
		return true
	case r >= 'J' && r <= 'K':
		return true
	case r >= 'M' && r <= 'N':
		return true
	case r >= 'P' && r <= 'T':
		return true
	case r >= 'V' && r <= 'Z':
		return true
	default:
		return false
	}
}
