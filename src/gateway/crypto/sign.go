package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	_ "crypto/sha512"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"

	_ "golang.org/x/crypto/sha3"
)

type ecdsaSequence struct {
	R, S *big.Int
}

type EcdsaSignature struct {
	R         *big.Int `json:"r"`
	S         *big.Int `json:"s"`
	Signature string   `json:"signature"`
}

type RsaSignature struct {
	Signature string `json:"signature"`
}

// Sign signs the data using the privKey and algorithm.
func Sign(data []byte, privKey interface{}, algorithmName string, padding string) (interface{}, error) {
	hash, err := GetSupportedAlgorithm(algorithmName)

	if err != nil {
		return nil, err
	}

	a := hash.New()
	a.Write(data)
	hashed := a.Sum(nil)

	switch privKey.(type) {
	case *rsa.PrivateKey:
		switch strings.ToLower(padding) {
		case "pkcs1v15":
			r, err := rsa.SignPKCS1v15(rand.Reader, privKey.(*rsa.PrivateKey), hash, hashed[:])
			if err != nil {
				return nil, err
			}

			sig := &RsaSignature{base64.StdEncoding.EncodeToString(r)}
			return sig, nil
		case "pss":
			r, err := rsa.SignPSS(rand.Reader, privKey.(*rsa.PrivateKey), hash, hashed[:], nil)
			if err != nil {
				return nil, err
			}

			sig := &RsaSignature{base64.StdEncoding.EncodeToString(r)}
			return sig, nil
		default:
			return nil, errors.New("invalid padding scheme")
		}
	case *ecdsa.PrivateKey:
		r, s, err := ecdsa.Sign(rand.Reader, privKey.(*ecdsa.PrivateKey), hashed[:])
		if err != nil {
			return nil, err
		}

		seq := ecdsaSequence{r, s}
		derSig, err := asn1.Marshal(seq)
		if err != nil {
			return nil, err
		}

		sig := &EcdsaSignature{R: r, S: s, Signature: base64.StdEncoding.EncodeToString(derSig)}
		return sig, nil
	default:
		return nil, errors.New(fmt.Sprintf("invalid or unsupported private key type: %T", privKey))
	}
}

func Verify(data []byte, signature string, publicKey interface{}, algorithmName string, padding string) (bool, error) {
	hash, err := GetSupportedAlgorithm(algorithmName)

	if err != nil {
		return false, err
	}

	a := hash.New()
	a.Write(data)
	hashed := a.Sum(nil)

	switch publicKey.(type) {
	case *rsa.PublicKey:
		switch strings.ToLower(padding) {
		case "pkcs1v15":
			decodedSignature, err := base64.StdEncoding.DecodeString(signature)

			if err != nil {
				return false, err
			}

			err = rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), hash, hashed[:], decodedSignature)
			return err == nil, err
		case "pss":
			decodedSignature, err := base64.StdEncoding.DecodeString(signature)

			if err != nil {
				return false, err
			}

			err = rsa.VerifyPSS(publicKey.(*rsa.PublicKey), hash, hashed[:], decodedSignature, nil)
			return err == nil, err
		default:
			return false, errors.New("invalid or unsupported padding scheme")
		}
	case *ecdsa.PublicKey:
		// Break the signature into R and S ints.
		decodedSignature, err := base64.StdEncoding.DecodeString(signature)
		if err != nil {
			return false, err
		}
		seq := &ecdsaSequence{}
		asn1.Unmarshal(decodedSignature, seq)

		valid := ecdsa.Verify(publicKey.(*ecdsa.PublicKey), hashed, seq.R, seq.S)
		return valid, nil
	default:
		return false, errors.New(fmt.Sprintf("invalid or unsupported public key type: %T", publicKey))
	}
}
