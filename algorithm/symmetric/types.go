package symmetric

type CipherMode int

const (
	ECB CipherMode = iota
	CBC
	PCBC
	CFB
	OFB
	CTR
	RandomDelta
)

type PaddingMode int

const (
	Zeros PaddingMode = iota
	AnsiX923
	PKCS7
	Iso10126
)

type RoundKey interface {
	GenerateKeys(inputKey []byte) ([][]byte, error)
}

type BlockCipher interface {
	Encryption(rightHalf, roundKey []byte) ([]byte, error)
	Decryption(rightHalf, roundKey []byte) ([]byte, error)
}

type CipherScheme interface {
	SetKey(key []byte) error
	Encrypt(block []byte) ([]byte, error)
	Decrypt(block []byte) ([]byte, error)
	EncryptAsync(data []byte) (<-chan []byte, <-chan error)
	DecryptAsync(data []byte) (<-chan []byte, <-chan error)
}
