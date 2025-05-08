package rc5

import (
	"bytes"
	"fmt"
)

var (
	p16 uint = 0xb7e1
	q16 uint = 0x9e37

	p32 uint = 0xb7e15163
	q32 uint = 0x9e3779b9

	p64 uint = 0xb7e151628aed2a6b
	q64 uint = 0x9e3779b97f4a7c15
)

type RC5 struct {
	key      []byte
	s        []uint
	wordSize uint
	rounds   uint
	b        uint
}

func (r *RC5) EncryptAsync(data []byte) (<-chan []byte, <-chan error) {
	//TODO implement me
	panic("implement me")
}

func (r *RC5) DecryptAsync(data []byte) (<-chan []byte, <-chan error) {
	//TODO implement me
	panic("implement me")
}

func NewRC5(w, r, b uint, key []byte) (*RC5, error) {
	if w != 16 && w != 32 && w != 64 {
		return nil, fmt.Errorf("w must be either 16 or 32 or 64")
	}
	if r > 255 {
		return nil, fmt.Errorf("r must be between 0 and 255")
	}
	if b > 255 {
		return nil, fmt.Errorf("b must be between 0 and 255")
	}
	if len(key) != int(b) {
		return nil, fmt.Errorf("key must be equal to length of key")
	}

	c := &RC5{
		wordSize: w,
		rounds:   r,
		b:        b,
		key:      key,
	}

	if err := c.GenerateS(key); err != nil {
		return nil, fmt.Errorf("failed to generate rc5 key: %w", err)
	}

	return c, nil
}

func (r *RC5) SetKey(key []byte) error {
	if !bytes.Equal(r.key, key) {
		r.key = key
		return r.GenerateS(key)
	}
	return nil
}

func (r *RC5) GenerateS(key []byte) error {
	if len(key) > 255 {
		return fmt.Errorf("key must be between 0 and 255")
	}
	s := r.MyNewKeyTable()
	l := r.MyBytesToWords(key)
	r.s = r.MyExpTable(s, l)
	return nil
}

func (r *RC5) MyNewKeyTable() []uint {
	t := 2 * (r.rounds + 1)
	S := make([]uint, t)
	mask := ^uint(0)
	if r.wordSize < 64 {
		mask = (1 << r.wordSize) - 1
	}

	p, q := r.selectMagicConstants()

	S[0] = p & mask
	for i := uint(1); i < t; i++ {
		S[i] = (S[i-1] + q) & mask
	}

	return S
}

func (r *RC5) MyBytesToWords(key []byte) []uint {

	u := r.wordSize / 8
	c := ((max(r.b, 1)) + u - 1) / u
	L := make([]uint, c)
	mask := ^uint(0)
	if r.wordSize < 64 {
		mask = (1 << r.wordSize) - 1
	}

	for i := int(r.b) - 1; i >= 0; i-- {
		L[uint(i)/u] = (r.Rotl(L[uint(i)/u], 8) + uint(key[i])) & mask
	}

	return L
}

func (r *RC5) MyExpTable(S []uint, L []uint) []uint {
	T := 2 * (r.rounds + 1)
	u := r.wordSize / 8
	c := max(r.b, 1) / u
	k := 3 * T
	if c > T {
		k = 3 * c
	}

	mask := uint((1 << r.wordSize) - 1)

	A, B := uint(0), uint(0)
	i, j := uint(0), uint(0)

	for ; k > 0; k-- {
		A = r.Rotl((S[i]+A+B)&mask, 3)
		S[i] = A

		sumL := (L[j] + A + B) & mask
		shift := (A + B) & mask
		B = r.Rotl(sumL, shift)
		L[j] = B

		i = (i + 1) % T
		j = (j + 1) % c
	}

	return S
}

func (r *RC5) selectMagicConstants() (uint, uint) {
	switch r.wordSize {
	case 16:
		return p16, q16
	case 32:
		return p32, q32
	case 64:
		return p64, q64
	default:
		return p16, q16
	}
}

func (r *RC5) Rotl(k uint, c uint) uint {
	c &= r.wordSize - 1
	return (k << c) | (k >> (r.wordSize - c))
}

func (r *RC5) Encrypt(block []byte) ([]byte, error) {
	wordBytes := int(r.wordSize / 8)
	if len(block) != 2*wordBytes {
		return nil, fmt.Errorf("block must be %d bytes", 2*wordBytes)
	}

	mask := uint((1 << r.wordSize) - 1)

	A := MyBytesToUint(block[:wordBytes])
	B := MyBytesToUint(block[wordBytes:])

	A = (A + r.s[0]) & mask
	B = (B + r.s[1]) & mask

	for i := uint(1); i <= r.rounds; i++ {
		A = (r.Rotl(A^B, B) + r.s[2*i]) & mask
		B = (r.Rotl(B^A, A) + r.s[2*i+1]) & mask
	}

	out := make([]byte, 0, 2*wordBytes)
	out = append(out, MyUintToBytes(A, wordBytes)...)
	out = append(out, MyUintToBytes(B, wordBytes)...)

	return out, nil
}

func (r *RC5) Decrypt(block []byte) ([]byte, error) {
	wordBytes := int(r.wordSize / 8)
	if len(block) != 2*wordBytes {
		return nil, fmt.Errorf("block must be %d bytes", 2*wordBytes)
	}

	mask := uint((1 << r.wordSize) - 1)

	A := MyBytesToUint(block[:wordBytes])
	B := MyBytesToUint(block[wordBytes:])

	for i := r.rounds; i >= 1; i-- {
		B = r.Rotr((B-r.s[2*i+1])&mask, A) ^ A
		A = r.Rotr((A-r.s[2*i])&mask, B) ^ B
	}

	B = (B - r.s[1]) & mask
	A = (A - r.s[0]) & mask

	out := make([]byte, 0, 2*wordBytes)
	out = append(out, MyUintToBytes(A, wordBytes)...)
	out = append(out, MyUintToBytes(B, wordBytes)...)

	return out, nil
}

func MyBytesToUint(b []byte) uint {
	var val uint
	for i := len(b) - 1; i >= 0; i-- {
		val = (val << 8) | uint(b[i])
	}
	return val
}

func MyUintToBytes(x uint, size int) []byte {
	b := make([]byte, size)
	for i := 0; i < size; i++ {
		b[i] = byte(x & 0xff)
		x >>= 8
	}
	return b
}

func (r *RC5) wordMask() uint {
	return (1 << r.wordSize) - 1
}

func (r *RC5) Rotr(x, c uint) uint {
	c %= r.wordSize
	return (x >> c) | (x << (r.wordSize - c))
}
