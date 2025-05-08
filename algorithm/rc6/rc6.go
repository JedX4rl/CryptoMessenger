package rc6

import (
	"bytes"
	"fmt"
)

var (
	p32 uint = 0xb7e15163
	q32 uint = 0x9e3779b9
)

const (
	w         = 32
	wordBytes = w / 8
	r         = 20
	t         = 2 * (r + 2)
)

type RC6 struct {
	key []byte
	S   []uint
}

func (c *RC6) EncryptAsync(data []byte) (<-chan []byte, <-chan error) {
	//TODO implement me
	panic("implement me")
}

func (c *RC6) DecryptAsync(data []byte) (<-chan []byte, <-chan error) {
	//TODO implement me
	panic("implement me")
}

func NewRC6(key []byte) (*RC6, error) {
	b := len(key)
	if b != 16 && b != 24 && b != 32 {
		return nil, fmt.Errorf("invalid key size %d", b)
	}
	c := &RC6{key: key}
	if err := c.GenerateS(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *RC6) SetKey(key []byte) error {
	if !bytes.Equal(c.key, key) {
		c.key = key
		return c.GenerateS()
	}
	return nil
}

func (c *RC6) GenerateS() error {
	mask := uint((1 << w) - 1)
	S := make([]uint, t)
	S[0] = p32 & mask
	for i := 1; i < t; i++ {
		S[i] = (S[i-1] + q32) & mask
	}
	b := len(c.key)
	Lw := (b + wordBytes - 1) / wordBytes
	L := make([]uint, Lw)
	for i := b - 1; i >= 0; i-- {
		L[i/wordBytes] = ((L[i/wordBytes] << 8) + uint(c.key[i])) & mask
	}
	A, B := uint(0), uint(0)
	i, j := 0, 0
	for k := 0; k < 3*max(Lw, t); k++ {
		A = rotl((S[i]+A+B)&mask, 3)
		S[i] = A
		B = rotl((L[j]+A+B)&mask, (A+B)&(w-1))
		L[j] = B
		i = (i + 1) % t
		j = (j + 1) % Lw
	}
	c.S = S
	return nil
}

func (c *RC6) Encrypt(block []byte) ([]byte, error) {
	if len(block) != 4*wordBytes {
		return nil, fmt.Errorf("invalid block size %d", len(block))
	}

	mask := uint((1 << w) - 1)

	A := BytesToUint(block[0:4])
	B := BytesToUint(block[4:8])
	C := BytesToUint(block[8:12])
	D := BytesToUint(block[12:16])

	B = (B + c.S[0]) & mask
	D = (D + c.S[1]) & mask

	for i := 1; i <= r; i++ {
		tv := rotl((B*(2*B+1))&mask, 5)
		uv := rotl((D*(2*D+1))&mask, 5)

		A = (rotl(A^tv, uv&(w-1)) + c.S[2*i]) & mask
		C = (rotl(C^uv, tv&(w-1)) + c.S[2*i+1]) & mask

		A, B, C, D = B, C, D, A
	}
	A = (A + c.S[2*r+2]) & mask
	C = (C + c.S[2*r+3]) & mask

	out := make([]byte, 4*wordBytes)

	copy(out[0:4], UintToBytes(A))
	copy(out[4:8], UintToBytes(B))
	copy(out[8:12], UintToBytes(C))
	copy(out[12:16], UintToBytes(D))

	return out, nil
}

func (c *RC6) Decrypt(block []byte) ([]byte, error) {
	if len(block) != 4*wordBytes {
		return nil, fmt.Errorf("invalid block size %d", len(block))
	}

	mask := uint((1 << w) - 1)

	A := BytesToUint(block[0:4])
	B := BytesToUint(block[4:8])
	C := BytesToUint(block[8:12])
	D := BytesToUint(block[12:16])

	C = (C - c.S[2*r+3]) & mask
	A = (A - c.S[2*r+2]) & mask

	for i := r; i >= 1; i-- {
		A, B, C, D = D, A, B, C

		uv := rotl((D*(2*D+1))&mask, 5)
		tv := rotl((B*(2*B+1))&mask, 5)

		C = rotr((C-c.S[2*i+1])&mask, tv&(w-1)) ^ uv
		A = rotr((A-c.S[2*i])&mask, uv&(w-1)) ^ tv
	}
	D = (D - c.S[1]) & mask
	B = (B - c.S[0]) & mask

	out := make([]byte, 4*wordBytes)

	copy(out[0:4], UintToBytes(A))
	copy(out[4:8], UintToBytes(B))
	copy(out[8:12], UintToBytes(C))
	copy(out[12:16], UintToBytes(D))

	return out, nil
}

func BytesToUint(b []byte) uint {
	var v uint
	for i := 0; i < len(b); i++ {
		v |= uint(b[i]) << (8 * i)
	}
	return v
}

func UintToBytes(x uint) []byte {
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		b[i] = byte(x)
		x >>= 8
	}
	return b
}

func rotl(x, y uint) uint {
	s := y & (w - 1)
	return (x<<s | x>>(w-s)) & ((1 << w) - 1)
}

func rotr(x, y uint) uint {
	s := y & (w - 1)
	return (x>>s | x<<(w-s)) & ((1 << w) - 1)
}
