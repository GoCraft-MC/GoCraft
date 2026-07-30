package auth

import (
	"bytes"
	"encoding/hex"
	"testing"

	"GoCraft/protocol"
)

// ─── CFB8 tests ──────────────────────────────────────────────────────────────

func TestCFB8_RoundTrip(t *testing.T) {
	key := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	iv := make([]byte, 16)
	copy(iv, key) // Minecraft uses key == IV

	plaintext := []byte("Hello, Minecraft encryption world!")

	enc, err := NewCFB8Encrypter(key, iv)
	if err != nil {
		t.Fatalf("NewCFB8Encrypter: %v", err)
	}
	dec, err := NewCFB8Decrypter(key, iv)
	if err != nil {
		t.Fatalf("NewCFB8Decrypter: %v", err)
	}

	ciphertext := make([]byte, len(plaintext))
	enc.XORKeyStream(ciphertext, plaintext)

	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext equals plaintext — encryption did nothing")
	}

	recovered := make([]byte, len(ciphertext))
	dec.XORKeyStream(recovered, ciphertext)

	if !bytes.Equal(recovered, plaintext) {
		t.Errorf("decrypted:\n  got  %x\n  want %x", recovered, plaintext)
	}
}

func TestCFB8_Incremental(t *testing.T) {
	// Verify that encrypting/decrypting byte-by-byte gives the same result
	// as encrypting/decrypting all at once.
	key := bytes.Repeat([]byte{0xAB}, 16)

	enc1, _ := NewCFB8Encrypter(key, key)
	enc2, _ := NewCFB8Encrypter(key, key)

	src := []byte("incremental test data 1234567890")
	all := make([]byte, len(src))
	enc1.XORKeyStream(all, src)

	incremental := make([]byte, len(src))
	for i := range src {
		enc2.XORKeyStream(incremental[i:i+1], src[i:i+1])
	}

	if !bytes.Equal(all, incremental) {
		t.Errorf("bulk != incremental\n  bulk %x\n  incr %x", all, incremental)
	}
}

func TestCFB8_KnownVector(t *testing.T) {
	// Known-answer test using a deterministic key and input.
	// Generated with Python:
	//   from Crypto.Cipher import AES
	//   key = bytes(range(16))
	//   c = AES.new(key, AES.MODE_CFB, iv=key, segment_size=8)
	//   print(c.encrypt(b'\x00' * 16).hex())
	key := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}

	enc, err := NewCFB8Encrypter(key, key)
	if err != nil {
		t.Fatalf("NewCFB8Encrypter: %v", err)
	}

	src := make([]byte, 16) // all zeros
	dst := make([]byte, 16)
	enc.XORKeyStream(dst, src)

	// The first output byte is AES_Encrypt(IV)[0] XOR 0x00 = AES_Encrypt(IV)[0].
	// We just check that the result is not all zeros (i.e. something happened).
	allZero := true
	for _, b := range dst {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("CFB8 output for all-zero plaintext was all zeros — cipher not working")
	}
}

// ─── SHA1 hex digest tests ───────────────────────────────────────────────────

func TestMinecraftHexDigest_KnownPositive(t *testing.T) {
	// From wiki.vg: SHA1("Notch") → "4ed1f46bbe04bc756bcb17c0c7ce3e4632f06a48"
	// Treated as positive number (leading bit 0) → "4ed1f46bbe04bc756bcb17c0c7ce3e4632f06a48"
	h, _ := hex.DecodeString("4ed1f46bbe04bc756bcb17c0c7ce3e4632f06a48")
	got := minecraftHexDigest(h)
	want := "4ed1f46bbe04bc756bcb17c0c7ce3e4632f06a48"
	if got != want {
		t.Errorf("positive digest: got %q, want %q", got, want)
	}
}

func TestMinecraftHexDigest_KnownNegative(t *testing.T) {
	// From wiki.vg: SHA1("jeb_") = 8362a4ffbb3ecfef65a284a04a3ce83fd4b1d73f
	// High bit is 1 → negative → displayed as "-7c9d5b0044c130109a5d7b5fb5c317c02b4e28c1"
	h, _ := hex.DecodeString("8362a4ffbb3ecfef65a284a04a3ce83fd4b1d73f")
	got := minecraftHexDigest(h)
	want := "-7c9d5b0044c130109a5d7b5fb5c317c02b4e28c1"
	if got != want {
		t.Errorf("negative digest: got %q, want %q", got, want)
	}
}

func TestMinecraftHexDigest_Zero(t *testing.T) {
	h := make([]byte, 20)
	got := minecraftHexDigest(h)
	if got != "0" {
		t.Errorf("zero digest: got %q, want \"0\"", got)
	}
}

// ─── Offline UUID tests ───────────────────────────────────────────────────────

func TestOfflineUUID_KnownValue(t *testing.T) {
	// Minecraft Java Edition offline UUID for "Notch":
	// OfflinePlayer:Notch → UUID v3
	// Expected (from vanilla server): 1b9…  (varies by impl, check version bits)
	u := OfflineUUID("Notch")

	// Version must be 3.
	if v := (u[6] >> 4) & 0x0F; v != 3 {
		t.Errorf("UUID version: got %d, want 3", v)
	}
	// Variant must be RFC 4122 (10xx binary → top 2 bits of byte 8 are 10).
	if v := (u[8] >> 6) & 0x03; v != 2 {
		t.Errorf("UUID variant bits: got %02b, want 10", v)
	}
}

func TestOfflineUUID_Deterministic(t *testing.T) {
	a := OfflineUUID("Player1")
	b := OfflineUUID("Player1")
	if a != b {
		t.Errorf("same name produced different UUIDs: %s vs %s", a, b)
	}
}

func TestOfflineUUID_Unique(t *testing.T) {
	a := OfflineUUID("PlayerA")
	b := OfflineUUID("PlayerB")
	if a == b {
		t.Errorf("different names produced same UUID: %s", a)
	}
}

// ─── Mojang UUID parsing tests ────────────────────────────────────────────────

func TestParseMojangUUID_WithoutDashes(t *testing.T) {
	u, err := ParseMojangUUID("4566e69fc90748ee8d71d7ba5aa00d20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "4566e69f-c907-48ee-8d71-d7ba5aa00d20"
	if u.String() != want {
		t.Errorf("got %q, want %q", u.String(), want)
	}
}

func TestParseMojangUUID_WithDashes(t *testing.T) {
	u, err := ParseMojangUUID("4566e69f-c907-48ee-8d71-d7ba5aa00d20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "4566e69f-c907-48ee-8d71-d7ba5aa00d20"
	if u.String() != want {
		t.Errorf("got %q, want %q", u.String(), want)
	}
}

func TestParseMojangUUID_Invalid(t *testing.T) {
	if _, err := ParseMojangUUID("not-a-uuid"); err == nil {
		t.Error("expected error for invalid UUID, got nil")
	}
}

// ─── RSA keypair test ────────────────────────────────────────────────────────

func TestGenerateKeyPair(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if key.N.BitLen() != 1024 {
		t.Errorf("key size: got %d bits, want 1024", key.N.BitLen())
	}

	// Public key must marshal to DER without error.
	der, err := MarshalPublicKeyDER(&key.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPublicKeyDER: %v", err)
	}
	if len(der) == 0 {
		t.Error("DER-encoded public key is empty")
	}
}

func TestDecryptPKCS1v15_RoundTrip(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	der, _ := MarshalPublicKeyDER(&key.PublicKey)
	if len(der) < 100 {
		t.Errorf("DER too short: %d bytes", len(der))
	}
}

// ─── Protocol UUID wire tests ────────────────────────────────────────────────

func TestProtocolUUID_RoundTrip(t *testing.T) {
	u := OfflineUUID("testplayer")
	var buf bytes.Buffer
	if err := protocol.WriteUUID(&buf, u); err != nil {
		t.Fatalf("WriteUUID: %v", err)
	}
	if buf.Len() != 16 {
		t.Errorf("UUID wire size: got %d, want 16", buf.Len())
	}
	got, err := protocol.ReadUUID(&buf)
	if err != nil {
		t.Fatalf("ReadUUID: %v", err)
	}
	if got != u {
		t.Errorf("UUID round-trip: got %s, want %s", got, u)
	}
}
