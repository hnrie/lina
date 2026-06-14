package crypt

import (
	"crypto/aes"
	goCipher "crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"strings"

	. "main/packages/onyx/logic/luau/Api"

	"golang.org/x/crypto/sha3"
)

type Crypt struct {
	Base64Encode  func(*LuaState) int `lua:"base64encode" alias:"crypt.base64encode,crypt.base64.encode,crypt.base64_encode,base64.encode,base64_encode"`
	Base64Decode  func(*LuaState) int `lua:"base64decode" alias:"crypt.base64decode,crypt.base64.decode,crypt.base64_decode,base64.decode,base64_decode"`
	Encrypt       func(*LuaState) int `lua:"encrypt" alias:"crypt.encrypt"`
	Decrypt       func(*LuaState) int `lua:"decrypt" alias:"crypt.decrypt"`
	GenerateBytes func(*LuaState) int `lua:"generatebytes" alias:"crypt.generatebytes,generatebytes"`
	GenerateKey   func(*LuaState) int `lua:"generatekey" alias:"crypt.generatekey,generatekey"`
	Hash          func(*LuaState) int `lua:"hash" alias:"crypt.hash"`
}

func Init(L *LuaState) {
	Register(L, Crypt{
		Base64Encode: func(ls *LuaState) int {
			if !ls.IsString(1) {
				ls.Error("base64encode: expected string argument")
				return 0
			}

			var Data string = ls.ToString(1)
			var Encoded string = base64.StdEncoding.EncodeToString([]byte(Data))
			ls.PushString(Encoded)
			return 1
		},

		Base64Decode: func(ls *LuaState) int {
			if !ls.IsString(1) {
				ls.Error("base64decode: expected string argument")
				return 0
			}

			var Data string = ls.ToString(1)
			Decoded, err := base64.StdEncoding.DecodeString(Data)
			if err != nil {
				ls.Error("base64decode: %v", err.Error())
				return 0
			}

			ls.PushString(string(Decoded))
			return 1
		},

		GenerateBytes: func(ls *LuaState) int {
			if !ls.IsNumber(1) {
				ls.Error("generatebytes: expected number argument")
				return 0
			}

			var Size int = ls.ToInteger(1)
			if Size <= 0 {
				ls.Error("generatebytes: size must be positive")
				return 0
			}

			var Bytes []byte = make([]byte, Size)
			_, err := rand.Read(Bytes)
			if err != nil {
				ls.Error("generatebytes: %v", err.Error())
				return 0
			}

			var Encoded string = base64.StdEncoding.EncodeToString(Bytes)
			ls.PushString(Encoded)
			return 1
		},

		GenerateKey: func(ls *LuaState) int {
			var Key []byte = make([]byte, 32)
			_, err := rand.Read(Key)
			if err != nil {
				ls.Error("generatekey: %v", err.Error())
				return 0
			}

			var Encoded string = base64.StdEncoding.EncodeToString(Key)
			ls.PushString(Encoded)
			return 1
		},

		Hash: func(ls *LuaState) int {
			if !ls.IsString(1) {
				ls.Error("hash: expected string argument at position 1")
				return 0
			}
			if !ls.IsString(2) {
				ls.Error("hash: expected string argument at position 2 (algorithm)")
				return 0
			}

			var Data string = ls.ToString(1)
			var Algorithm string = strings.ToLower(ls.ToString(2))

			var Hash []byte

			switch Algorithm {
			case "md5":
				var H [16]byte = md5.Sum([]byte(Data))
				Hash = H[:]

			case "sha1":
				var H [20]byte = sha1.Sum([]byte(Data))
				Hash = H[:]

			case "sha256":
				var H [32]byte = sha256.Sum256([]byte(Data))
				Hash = H[:]

			case "sha384":
				var H [48]byte = sha512.Sum384([]byte(Data))
				Hash = H[:]

			case "sha512":
				var H [64]byte = sha512.Sum512([]byte(Data))
				Hash = H[:]

			case "sha3-224":
				var H [28]byte = sha3.Sum224([]byte(Data))
				Hash = H[:]

			case "sha3-256":
				var H [32]byte = sha3.Sum256([]byte(Data))
				Hash = H[:]

			case "sha3-512":
				var H [64]byte = sha3.Sum512([]byte(Data))
				Hash = H[:]

			default:
				ls.Error("hash: unsupported algorithm '%s'", Algorithm)
				return 0
			}

			var Result string = strings.ToUpper(hex.EncodeToString(Hash))
			ls.PushString(Result)
			return 1
		},

		Encrypt: func(ls *LuaState) int {
			if !ls.IsString(1) {
				ls.Error("encrypt: expected string argument at position 1 (data)")
				return 0
			}
			if !ls.IsString(2) {
				ls.Error("encrypt: expected string argument at position 2 (key)")
				return 0
			}

			var Data string = ls.ToString(1)
			var KeyB64 string = ls.ToString(2)
			var IvB64 string = ls.OptString(3, "")
			var Mode string = strings.ToUpper(ls.OptString(4, "CBC"))

			Key, err := base64.StdEncoding.DecodeString(KeyB64)
			if err != nil {
				ls.Error("encrypt: invalid base64 key: %v", err.Error())
				return 0
			}

			if len(Key) != 32 {
				ls.Error("encrypt: key must be 256-bit (32 bytes), got %d bytes", len(Key))
				return 0
			}

			var Iv []byte
			if IvB64 != "" {
				Iv, err = base64.StdEncoding.DecodeString(IvB64)
				if err != nil {
					ls.Error("encrypt: invalid base64 IV: %v", err.Error())
					return 0
				}
			} else {
				Iv = make([]byte, 16)
				if _, err := rand.Read(Iv); err != nil {
					ls.Error("encrypt: failed to generate IV: %v", err.Error())
					return 0
				}
			}

			if len(Iv) != 16 {
				Iv = Iv[:16]
			}

			Block, err := aes.NewCipher(Key)
			if err != nil {
				ls.Error("encrypt: %v", err.Error())
				return 0
			}

			var Encrypted []byte

			switch Mode {
			case "CBC":
				var Padded []byte = pkcs7Pad([]byte(Data), Block.BlockSize())
				Encrypted = make([]byte, len(Padded))
				goCipher.NewCBCEncrypter(Block, Iv).CryptBlocks(Encrypted, Padded)

			case "ECB":
				var Padded []byte = pkcs7Pad([]byte(Data), Block.BlockSize())
				Encrypted = ecbEncrypt(Block, Padded)

			case "CTR":
				Encrypted = make([]byte, len(Data))
				goCipher.NewCTR(Block, Iv).XORKeyStream(Encrypted, []byte(Data))

			case "CFB":
				Encrypted = make([]byte, len(Data))
				goCipher.NewCFBEncrypter(Block, Iv).XORKeyStream(Encrypted, []byte(Data))

			case "OFB":
				Encrypted = make([]byte, len(Data))
				goCipher.NewOFB(Block, Iv).XORKeyStream(Encrypted, []byte(Data))

			case "GCM":
				Gcm, err := goCipher.NewGCM(Block)
				if err != nil {
					ls.Error("encrypt: GCM mode not available: %v", err.Error())
					return 0
				}
				Encrypted = Gcm.Seal(nil, Iv, []byte(Data), nil)

			default:
				ls.Error("encrypt: unsupported mode '%s'", Mode)
				return 0
			}

			ls.PushString(base64.StdEncoding.EncodeToString(Encrypted))
			ls.PushString(base64.StdEncoding.EncodeToString(Iv))
			return 2
		},

		Decrypt: func(ls *LuaState) int {
			if !ls.IsString(1) {
				ls.Error("decrypt: expected string argument at position 1 (data)")
				return 0
			}
			if !ls.IsString(2) {
				ls.Error("decrypt: expected string argument at position 2 (key)")
				return 0
			}
			if !ls.IsString(3) {
				ls.Error("decrypt: expected string argument at position 3 (iv)")
				return 0
			}

			var DataB64 string = ls.ToString(1)
			var KeyB64 string = ls.ToString(2)
			var IvB64 string = ls.ToString(3)
			var Mode string = strings.ToUpper(ls.OptString(4, "CBC"))

			Encrypted, err := base64.StdEncoding.DecodeString(DataB64)
			if err != nil {
				ls.Error("decrypt: invalid base64 data: %v", err.Error())
				return 0
			}

			Key, err := base64.StdEncoding.DecodeString(KeyB64)
			if err != nil {
				ls.Error("decrypt: invalid base64 key: %v", err.Error())
				return 0
			}

			Iv, err := base64.StdEncoding.DecodeString(IvB64)
			if err != nil {
				ls.Error("decrypt: invalid base64 IV: %v", err.Error())
				return 0
			}

			if len(Key) != 32 {
				ls.Error("decrypt: key must be 256-bit (32 bytes), got %d bytes", len(Key))
				return 0
			}

			if len(Iv) != 16 {
				Iv = Iv[:16]
			}

			Block, err := aes.NewCipher(Key)
			if err != nil {
				ls.Error("decrypt: %v", err.Error())
				return 0
			}

			var Decrypted []byte

			switch Mode {
			case "CBC":
				Decrypted = make([]byte, len(Encrypted))
				goCipher.NewCBCDecrypter(Block, Iv).CryptBlocks(Decrypted, Encrypted)
				Decrypted = pkcs7Unpad(Decrypted)

			case "ECB":
				Decrypted = pkcs7Unpad(ecbDecrypt(Block, Encrypted))

			case "CTR":
				Decrypted = make([]byte, len(Encrypted))
				goCipher.NewCTR(Block, Iv).XORKeyStream(Decrypted, Encrypted)

			case "CFB":
				Decrypted = make([]byte, len(Encrypted))
				goCipher.NewCFBDecrypter(Block, Iv).XORKeyStream(Decrypted, Encrypted)

			case "OFB":
				Decrypted = make([]byte, len(Encrypted))
				goCipher.NewOFB(Block, Iv).XORKeyStream(Decrypted, Encrypted)

			case "GCM":
				Gcm, err := goCipher.NewGCM(Block)
				if err != nil {
					ls.Error("decrypt: GCM mode not available: %v", err.Error())
					return 0
				}
				Decrypted, err = Gcm.Open(nil, Iv, Encrypted, nil)
				if err != nil {
					ls.Error("decrypt: GCM decryption failed: %v", err.Error())
					return 0
				}

			default:
				ls.Error("decrypt: unsupported mode '%s'", Mode)
				return 0
			}

			ls.PushString(string(Decrypted))
			return 1
		},
	})
}

func pkcs7Pad(Data []byte, BlockSize int) []byte {
	var Padding int = BlockSize - (len(Data) % BlockSize)
	var Padded []byte = make([]byte, len(Data)+Padding)
	copy(Padded, Data)
	for i := len(Data); i < len(Padded); i++ {
		Padded[i] = byte(Padding)
	}
	return Padded
}

func pkcs7Unpad(Data []byte) []byte {
	if len(Data) == 0 {
		return Data
	}
	var Padding int = int(Data[len(Data)-1])
	if Padding > len(Data) || Padding == 0 {
		return Data
	}
	for i := 0; i < Padding; i++ {
		if Data[len(Data)-1-i] != byte(Padding) {
			return Data
		}
	}
	return Data[:len(Data)-Padding]
}

func ecbEncrypt(Block goCipher.Block, Data []byte) []byte {
	var Encrypted []byte = make([]byte, len(Data))
	var Size int = Block.BlockSize()
	for i := 0; i < len(Data); i += Size {
		Block.Encrypt(Encrypted[i:i+Size], Data[i:i+Size])
	}
	return Encrypted
}

func ecbDecrypt(Block goCipher.Block, Data []byte) []byte {
	var Decrypted []byte = make([]byte, len(Data))
	var Size int = Block.BlockSize()
	for i := 0; i < len(Data); i += Size {
		Block.Decrypt(Decrypted[i:i+Size], Data[i:i+Size])
	}
	return Decrypted
}
