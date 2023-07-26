package utils

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/blake2b"

	"github.com/bnb-chain/greenfield-sp-standard-test/core/log"
)

const (
	sizeFr = fr.Bytes
)

func GetNonce(userAddress string, endpoint string) (string, error) {
	header := make(map[string]string)
	header["X-Gnfd-User-Address"] = userAddress
	header["X-Gnfd-App-Domain"] = "https://greenfield.bnbchain.org/"
	response, err := HttpGetWithHeader(endpoint+"/auth/request_nonce", header)
	return response, err
}

func UpdateAccountKey(SpAddress, endpoint string) (string, error) {
	privateKeyNew, _ := crypto.GenerateKey()
	//privateKeyNew := GetEddsaCompressedPublicKey(seed)

	addressNew := crypto.PubkeyToAddress(privateKeyNew.PublicKey)
	log.Infof("address is: %s", addressNew.Hex())
	requestNonce := "1"
	unSignedContent := "Sign this message to access SP 0x123453333331111" + addressNew.Hex() //
	unSignedContentHash := accounts.TextHash([]byte(unSignedContent))                        // personal sign
	edcsaSig, _ := crypto.Sign(unSignedContentHash, privateKeyNew)
	userEddsaPublicKeyStr := GetEddsaCompressedPublicKey(string(edcsaSig))
	log.Infof("userEddsaPublicKeyStr is %s", userEddsaPublicKeyStr)
	domainNew := "https://greenfield.bnbchain.org/"

	PublicKeyString := userEddsaPublicKeyStr
	ExpiryDate := time.Now().Add(time.Hour * 24).Format(time.RFC3339)
	log.Infof("ExpiryDate: %v", ExpiryDate)
	User := addressNew.Hex()

	unSignedContent333 := fmt.Sprintf("%v wants you to sign in with your BNB Greenfield account:%v\\nRegister your identity public key %v\\nURI: %v\\nVersion: 1\\nChain ID: 9000\\nIssued At: 2023-06-21T10:15:38Z\\nExpiration Time: %v\\nResources:\\n- SP %v (name: sp4) with nonce: %v", domainNew, User, PublicKeyString, domainNew, ExpiryDate, SpAddress, requestNonce)
	unSignedContent444 := fmt.Sprintf("%v wants you to sign in with your BNB Greenfield account:%v\nRegister your identity public key %v\nURI: %v\nVersion: 1\nChain ID: 9000\nIssued At: 2023-06-21T10:15:38Z\nExpiration Time: %v\nResources:\n- SP %v (name: sp4) with nonce: %v", domainNew, User, PublicKeyString, domainNew, ExpiryDate, SpAddress, requestNonce)

	unSignedContentHash33 := accounts.TextHash([]byte(unSignedContent444))
	signature333, _ := crypto.Sign(unSignedContentHash33, privateKeyNew)
	SignString := ConvertToString(signature333)
	AuthString := fmt.Sprintf("PersonalSign ECDSA-secp256k1,SignedMsg=%v,Signature=%v", unSignedContent333, SignString)

	headers := make(map[string]string)
	headers["x-gnfd-app-domain"] = domainNew
	headers["x-gnfd-app-reg-nonce"] = requestNonce
	headers["x-gnfd-app-reg-public-key"] = PublicKeyString
	headers["X-Gnfd-Expiry-Timestamp"] = ExpiryDate
	headers["authorization"] = AuthString
	headers["origin"] = domainNew
	headers["x-gnfd-user-address"] = User
	return HttpPostWithHeader(endpoint+"/auth/update_key", "{}", headers)
}

func GetEddsaCompressedPublicKey(seed string) string {
	buf := make([]byte, 32)
	copy(buf, seed)
	reader := bytes.NewReader(buf)
	sk, err := GenerateKey(reader)
	if err != nil {
		return err.Error()
	}
	var buf2 bytes.Buffer
	buf2.Write(sk.PublicKey.Bytes())
	return hex.EncodeToString(buf2.Bytes())
}

func ConvertToString(targetStrung []byte) string {
	var buf bytes.Buffer
	buf.Write(targetStrung)
	return "0x" + hex.EncodeToString(buf.Bytes())
}

func GenerateKey(r io.Reader) (*eddsa.PrivateKey, error) {
	c := twistededwards.GetEdwardsCurve()
	var (
		randSrc = make([]byte, 32)
		scalar  = make([]byte, 32)
		pub     eddsa.PublicKey
	)

	// hash(h) = private_key || random_source, on 32 bytes each
	seed := make([]byte, 32)
	_, err := r.Read(seed)
	if err != nil {
		return nil, err
	}
	h := blake2b.Sum512(seed[:])
	for i := 0; i < 32; i++ {
		randSrc[i] = h[i+32]
	}

	// prune the key
	// https://tools.ietf.org/html/rfc8032#section-5.1.5, key generation

	h[0] &= 0xF8
	h[31] &= 0x7F
	h[31] |= 0x40

	// 0xFC = 1111 1100
	// convert 256 bits to 254 bits supporting bn254 curve

	h[31] &= 0xFC

	// reverse first bytes because setBytes interpret stream as big endian
	// but in eddsa specs s is the first 32 bytes in little endian
	for i, j := 0, sizeFr-1; i < sizeFr; i, j = i+1, j-1 {
		scalar[i] = h[j]
	}

	a := new(big.Int).SetBytes(scalar[:])
	for i := 253; i < 256; i++ {
		a.SetBit(a, i, 0)
	}

	copy(scalar[:], a.FillBytes(make([]byte, 32)))

	var bscalar big.Int
	bscalar.SetBytes(scalar[:])
	pub.A.ScalarMul(&c.Base, &bscalar)
	var res [sizeFr * 3]byte
	pubkBin := pub.A.Bytes()
	subtle.ConstantTimeCopy(1, res[:sizeFr], pubkBin[:])
	subtle.ConstantTimeCopy(1, res[sizeFr:2*sizeFr], scalar[:])
	subtle.ConstantTimeCopy(1, res[2*sizeFr:], randSrc[:])

	var sk = &eddsa.PrivateKey{}
	// make sure sk is not nil
	_, err = sk.SetBytes(res[:])

	return sk, err
}

func RegisterEDDSAPublicKey(appDomain, endpoint, eddsaSeed, SpAddress, requestNonce string, addressNew common.Address, privateKeyNew *ecdsa.PrivateKey) (string, error) {
	userEddsaPublicKeyStr := GetEddsaCompressedPublicKey(eddsaSeed)
	log.Debugf("userEddsaPublicKeyStr is %s", userEddsaPublicKeyStr)
	publicKeyString := userEddsaPublicKeyStr
	// ExpiryDate := "2023-06-27T06:35:24Z"
	ExpiryDate := time.Now().Add(time.Hour * 24).Format(time.RFC3339)
	log.Debug("ExpiryDate:", ExpiryDate)
	User := addressNew.Hex()
	unSignedContent1 := fmt.Sprintf("%v wants you to sign in with your BNB Greenfield account:%v\\nRegister your identity public key %v\\nURI: %v\\nVersion: 1\\nChain ID: 9000\\nIssued At: 2023-06-21T10:15:38Z\\nExpiration Time: %v\\nResources:\\n- SP %v (name: sp4) with nonce: %v", appDomain, User, publicKeyString, appDomain, ExpiryDate, SpAddress, requestNonce)
	unSignedContent2 := fmt.Sprintf("%v wants you to sign in with your BNB Greenfield account:%v\nRegister your identity public key %v\nURI: %v\nVersion: 1\nChain ID: 9000\nIssued At: 2023-06-21T10:15:38Z\nExpiration Time: %v\nResources:\n- SP %v (name: sp4) with nonce: %v", appDomain, User, publicKeyString, appDomain, ExpiryDate, SpAddress, requestNonce)

	unSignedContentHash33 := accounts.TextHash([]byte(unSignedContent2))
	signature, _ := crypto.Sign(unSignedContentHash33, privateKeyNew)
	SignString := ConvertToString(signature)
	AuthString := fmt.Sprintf("PersonalSign ECDSA-secp256k1,SignedMsg=%v,Signature=%v", unSignedContent1, SignString)

	headers := make(map[string]string)
	headers["x-gnfd-app-domain"] = appDomain
	headers["x-gnfd-app-reg-nonce"] = requestNonce
	headers["x-gnfd-app-reg-public-key"] = publicKeyString
	headers["X-Gnfd-Expiry-Timestamp"] = ExpiryDate
	headers["authorization"] = AuthString
	headers["origin"] = "https://greenfield.bnbchain.org/"
	headers["x-gnfd-user-address"] = User
	jsonResult, error1 := HttpPostWithHeader(endpoint+"/auth/update_key", "{}", headers)

	return jsonResult, error1
}

func GenerateEddsaPrivateKey(seed string) (sk *eddsa.PrivateKey, err error) {
	buf := make([]byte, 32)
	copy(buf, seed)
	reader := bytes.NewReader(buf)
	sk, err = GenerateKey(reader)
	return sk, err
}
