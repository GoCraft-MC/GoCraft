package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// cfb8Stream implements cipher.Stream for AES in CFB8 mode.
//
// Go's standard library provides CFB with 128-bit segment size (CFB128).
// Minecraft uses CFB with 8-bit segment size (CFB8), so we implement it here.
//
// CFB8 algorithm for each byte i:
//
//	keystream = AES_Encrypt(shiftRegister)
//	if encrypting: ciphertext[i] = plaintext[i] ^ keystream[0]
//	               shiftRegister = (shiftRegister << 8) | ciphertext[i]
//	if decrypting: plaintext[i]  = ciphertext[i] ^ keystream[0]
//	               shiftRegister = (shiftRegister << 8) | ciphertext[i]
//
// In both modes the shift register is updated with the ciphertext byte.
type cfb8Stream struct {
	block    cipher.Block
	sr       []byte // shift register (one AES block = 16 bytes)
	ks       []byte // keystream scratch buffer (one AES block)
	encrypts bool
}

// NewCFB8Encrypter returns a cipher.Stream that encrypts in AES-CFB8 mode.
// key must be 16 bytes (AES-128). iv must be 16 bytes and equals key in Minecraft.
func NewCFB8Encrypter(key, iv []byte) (cipher.Stream, error) {
	return newCFB8(key, iv, true)
}

// NewCFB8Decrypter returns a cipher.Stream that decrypts in AES-CFB8 mode.
// key must be 16 bytes (AES-128). iv must be 16 bytes and equals key in Minecraft.
func NewCFB8Decrypter(key, iv []byte) (cipher.Stream, error) {
	return newCFB8(key, iv, false)
}

func newCFB8(key, iv []byte, encrypts bool) (cipher.Stream, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("auth: creating AES cipher: %w", err)
	}
	if len(iv) != block.BlockSize() {
		return nil, fmt.Errorf("auth: IV length %d != block size %d", len(iv), block.BlockSize())
	}
	sr := make([]byte, block.BlockSize())
	copy(sr, iv)
	return &cfb8Stream{
		block:    block,
		sr:       sr,
		ks:       make([]byte, block.BlockSize()),
		encrypts: encrypts,
	}, nil
}

func (s *cfb8Stream) XORKeyStream(dst, src []byte) {
	if len(dst) < len(src) {
		panic("auth: CFB8 output buffer smaller than input")
	}
	for i := range src {
		// Encrypt the shift register to produce one block of keystream.
		s.block.Encrypt(s.ks, s.sr)

		// XOR the first keystream byte with the source byte.
		ct := src[i] // the ciphertext byte (src[i] when decrypting, dst[i] when encrypting)
		dst[i] = src[i] ^ s.ks[0]
		if s.encrypts {
			ct = dst[i] // for encryption the ciphertext is what we just produced
		}

		// Shift the register left by one byte and append the ciphertext byte.
		copy(s.sr, s.sr[1:])
		s.sr[len(s.sr)-1] = ct
	}
}
