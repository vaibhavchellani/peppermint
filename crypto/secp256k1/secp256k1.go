package secp256k1

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"

	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

//-------------------------------------
const (
	Secp256k1PrivKeyAminoRoute = "tendermint/PrivKeySecp256k1"
	Secp256k1PubKeyAminoRoute  = "tendermint/PubKeySecp256k1"
)

var cdc = amino.NewCodec()

func init() {
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(PubKeySecp256k1{},
		Secp256k1PubKeyAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(PrivKeySecp256k1{},
		Secp256k1PrivKeyAminoRoute, nil)
}

//-------------------------------------

var _ crypto.PrivKey = PrivKeySecp256k1{}

// PrivKeySecp256k1 implements PrivKey.
type PrivKeySecp256k1 [32]byte

// Bytes marshalls the private key using amino encoding.
func (privKey PrivKeySecp256k1) Bytes() []byte {
	return cdc.MustMarshalBinaryBare(privKey)
}

// Sign creates an ECDSA signature on curve Secp256k1, using SHA256 on the msg.
func (privKey PrivKeySecp256k1) Sign(msg []byte) ([]byte, error) {
	privateObject, err := ethCrypto.ToECDSA(privKey[:])
	if err != nil {
		return nil, err
	}

	return ethCrypto.Sign(ethCrypto.Keccak256(msg), privateObject)
}

// PubKey performs the point-scalar multiplication from the privKey on the
// generator point to get the pubkey.
func (privKey PrivKeySecp256k1) PubKey() crypto.PubKey {
	privateObject, err := ethCrypto.ToECDSA(privKey[:])
	if err != nil {
		panic(err)
	}

	var pubkeyBytes PubKeySecp256k1
	copy(pubkeyBytes[:], ethCrypto.FromECDSAPub(&privateObject.PublicKey))
	return pubkeyBytes
}

// Equals - you probably don't need to use this.
// Runs in constant time based on length of the keys.
func (privKey PrivKeySecp256k1) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySecp256k1); ok {
		return subtle.ConstantTimeCompare(privKey[:], otherSecp[:]) == 1
	}
	return false
}

// GenPrivKey generates a new ECDSA private key on curve secp256k1 private key.
// It uses OS randomness in conjunction with the current global random seed
// in tendermint/libs/common to generate the private key.
func GenPrivKey() PrivKeySecp256k1 {
	return genPrivKey(crypto.CReader())
}

// genPrivKey generates a new secp256k1 private key using the provided reader.
func genPrivKey(rand io.Reader) PrivKeySecp256k1 {
	privKeyBytes := [32]byte{}
	_, err := io.ReadFull(rand, privKeyBytes[:])
	if err != nil {
		panic(err)
	}
	// crypto.CRandBytes is guaranteed to be 32 bytes long, so it can be
	// casted to PrivKeySecp256k1.
	return PrivKeySecp256k1(privKeyBytes)
}

// GenPrivKeySecp256k1 hashes the secret with SHA2, and uses
// that 32 byte output to create the private key.
// NOTE: secret should be the output of a KDF like bcrypt,
// if it's derived from user input.
func GenPrivKeySecp256k1(secret []byte) PrivKeySecp256k1 {
	privKey32 := sha256.Sum256(secret)
	// sha256.Sum256() is guaranteed to be 32 bytes long, so it can be
	// casted to PrivKeySecp256k1.
	return PrivKeySecp256k1(privKey32)
}

//-------------------------------------

var _ crypto.PubKey = PubKeySecp256k1{}

// PubKeySecp256k1Size is comprised of 32 bytes for one field element
// (the x-coordinate), plus one byte for the parity of the y-coordinate.
const PubKeySecp256k1Size = 65

// PubKeySecp256k1 implements crypto.PubKey.
// It is the compressed form of the pubkey. The first byte depends is a 0x02 byte
// if the y-coordinate is the lexicographically largest of the two associated with
// the x-coordinate. Otherwise the first byte is a 0x03.
// This prefix is followed with the x-coordinate.
type PubKeySecp256k1 [PubKeySecp256k1Size]byte

// Address returns a Bitcoin style addresses: RIPEMD160(SHA256(pubkey))
func (pubKey PubKeySecp256k1) Address() crypto.Address {
	// ETH compliant
	return crypto.Address(ethCrypto.Keccak256(pubKey[1:])[12:])
}

// Bytes returns the pubkey marshalled with amino encoding.
func (pubKey PubKeySecp256k1) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(pubKey)
	if err != nil {
		panic(err)
	}
	return bz
}

func (pubKey PubKeySecp256k1) VerifyBytes(msg []byte, sig []byte) bool {
	hash := ethCrypto.Keccak256(msg)
	return ethCrypto.VerifySignature(pubKey[:], hash, sig[:64])
}

func (pubKey PubKeySecp256k1) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

func (pubKey PubKeySecp256k1) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false
}
