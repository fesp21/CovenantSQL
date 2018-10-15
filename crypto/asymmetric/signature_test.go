/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package asymmetric

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/ed25519"
)

var (
	priv *PrivateKey
	pub  *PublicKey
)

func init() {
	rand.Seed(time.Now().UnixNano())

	var err error
	priv, pub, err = GenSecp256k1KeyPair()
	if err != nil {
		panic(err)
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{
			name: "Test curve",
			key: []byte{
				0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
				0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
				0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
				0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
			},
		},
	}

	for _, test := range tests {
		priv, pub := PrivKeyFromBytes(test.key)
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		sig, err := priv.Sign(hash)

		if err != nil {
			t.Errorf("%s could not sign: %v", test.name, err)
			continue
		}

		if !sig.Verify(hash, pub) {
			t.Errorf("%s could not verify: %v", test.name, err)
			continue
		}

		serializedKey := priv.Serialize()
		if !bytes.Equal(serializedKey, test.key) {
			t.Errorf("%s unexpected serialized bytes - got: %x, want: %x", test.name,
				serializedKey, test.key)
		}

		serializedSig := sig.Serialize()
		targetSig, err := ParseDERSignature(serializedSig, btcec.S256())
		if err != nil {
			t.Errorf("%s could not serialized: %v", test.name, err)
			continue
		}
		if !reflect.DeepEqual(sig, targetSig) {
			t.Errorf("%s unexpected serialized bytes - got: %x, want: %x", test.name,
				targetSig, sig)
		}
	}
}

func TestSignature_MarshalBinary(t *testing.T) {
	Convey("marshal unmarshal signature", t, func() {
		// generate
		privateKey, publicKey, _ := GenSecp256k1KeyPair()

		// random data
		buf := make([]byte, 32)
		rand.Read(buf)

		// sign
		var sign *Signature
		sign, _ = privateKey.Sign(buf)

		var err error
		keyBytes, err := sign.MarshalBinary()
		So(err, ShouldBeNil)

		sign2 := new(Signature)
		err = sign2.UnmarshalBinary(keyBytes)
		So(err, ShouldBeNil)

		So(sign2.Verify(buf, publicKey), ShouldBeTrue)
	})

	Convey("test marshal unmarshal nil value", t, func() {
		var sig *Signature
		var err error

		// nil value marshal
		_, err = sig.MarshalBinary()
		So(err, ShouldNotBeNil)

		// nil value unmarshal
		err = sig.UnmarshalBinary([]byte{})
		So(err, ShouldNotBeNil)
	})
}

func BenchmarkGenKey(b *testing.B) {
	b.Log(b.Name())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := GenSecp256k1KeyPair(); err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}
func generateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(secp256k1.S256(), crand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)

	return pubkey, privkey
}

func BenchmarkSign(b *testing.B) {
	b.Run("Secp256k1", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := priv.Sign(hash[:])
			if err != nil {
				b.Fatalf("Error occurred: %v", err)
			}
		}
	})

	b.Run("C-Secp256k1", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		_, privP := generateKeyPair()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := secp256k1.Sign(hash, privP)
			if err != nil {
				b.Fatalf("Error occurred: %v", err)
			}
		}
	})

	b.Run("P224", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, _ := ecdsa.GenerateKey(elliptic.P224(), crand.Reader)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Sign(crand.Reader, privP, hash)
		}
	})

	b.Run("P256", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, _ := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Sign(crand.Reader, privP, hash)
		}
	})

	b.Run("P384", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, _ := ecdsa.GenerateKey(elliptic.P384(), crand.Reader)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Sign(crand.Reader, privP, hash)
		}
	})

	b.Run("P521", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, _ := ecdsa.GenerateKey(elliptic.P521(), crand.Reader)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Sign(crand.Reader, privP, hash)
		}
	})

	b.Run("Curve25519", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		_, privP, _ := ed25519.GenerateKey(crand.Reader)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ed25519.Sign(privP, hash)
		}
	})
}

func BenchmarkVerify(b *testing.B) {
	b.Run("Secp256k1", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		sig, err := priv.Sign(hash[:])
		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sig.Verify(hash[:], pub)
		}
	})

	b.Run("C-Secp256k1", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		pubP, privP := generateKeyPair()

		s, err := secp256k1.Sign(hash, privP)
		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			secp256k1.VerifySignature(pubP, hash, s[:64])
		}
	})

	b.Run("P224", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, err := ecdsa.GenerateKey(elliptic.P224(), crand.Reader)

		if err != nil {
			panic(err)
		}
		pubP := privP.PublicKey

		r, s, _ := ecdsa.Sign(crand.Reader, privP, hash)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Verify(&pubP, hash, r, s)
		}
	})

	b.Run("P256", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)

		if err != nil {
			panic(err)
		}
		pubP := privP.PublicKey

		r, s, _ := ecdsa.Sign(crand.Reader, privP, hash)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Verify(&pubP, hash, r, s)
		}

	})

	b.Run("P384", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, err := ecdsa.GenerateKey(elliptic.P384(), crand.Reader)

		if err != nil {
			panic(err)
		}
		pubP := privP.PublicKey

		r, s, _ := ecdsa.Sign(crand.Reader, privP, hash)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Verify(&pubP, hash, r, s)
		}
	})

	b.Run("P521", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		privP, err := ecdsa.GenerateKey(elliptic.P521(), crand.Reader)

		if err != nil {
			panic(err)
		}
		pubP := privP.PublicKey

		r, s, _ := ecdsa.Sign(crand.Reader, privP, hash)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ecdsa.Verify(&pubP, hash, r, s)
		}
	})

	b.Run("ed25519", func(b *testing.B) {
		b.Log(b.Name())
		hash := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		pubP, privP, err := ed25519.GenerateKey(crand.Reader)

		if err != nil {
			panic(err)
		}

		s := ed25519.Sign(privP, hash)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if !ed25519.Verify(pubP, hash, s) {
				b.Fatal(b.Name())
			}
		}
	})
}

func BenchmarkPublicKeySerialization(b *testing.B) {
	b.Log(b.Name())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pub.Serialize()
	}
}

func BenchmarkParsePublicKey(b *testing.B) {
	b.Log(b.Name())
	buffer := pub.Serialize()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParsePubKey(buffer)

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}

func BenchmarkSignatureSerialization(b *testing.B) {
	b.Log(b.Name())
	var hash [32]byte
	rand.Read(hash[:])
	sig, err := priv.Sign(hash[:])

	if err != nil {
		b.Fatalf("Error occurred: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sig.Serialize()
	}
}

func BenchmarkParseSignature(b *testing.B) {
	b.Log(b.Name())
	var hash [32]byte
	rand.Read(hash[:])
	sig, err := priv.Sign(hash[:])
	buffer := sig.Serialize()

	if err != nil {
		b.Fatalf("Error occurred: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseSignature(buffer)

		if err != nil {
			b.Fatalf("Error occurred: %v", err)
		}
	}
}
