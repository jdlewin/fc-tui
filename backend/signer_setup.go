// One-time signer setup: create -> EIP-712 sign with developer mnemonic ->
// register -> approval URL -> poll -> save SIGNER_UUID to ../.env.
// Usage: FARCASTER_DEVELOPER_MNEMONIC="word ..." go run . -signer-setup
// The mnemonic is your Farcaster recovery phrase (Warpcast Settings).
// It never leaves this laptop: it only signs locally, Neynar only sees the signature.
package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// EIP-712 domain for Farcaster SignedKeyRequestValidator (matches Neynar docs
// and @farcaster/hub-nodejs ViemLocalEip712Signer).
const (
	skrDomainName    = "Farcaster SignedKeyRequestValidator"
	skrDomainVersion = "1"
	skrChainID       = 10
	skrContract      = "0x00000000Fc700472606ED4FA22623Acf62c60553"
	skrTypeString    = "SignedKeyRequest(uint256 requestFid,bytes key,uint256 deadline)"
)

func abiUint256(v *big.Int) []byte {
	b := v.Bytes()
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func abiAddress(hexAddr string) []byte {
	b, _ := hex.DecodeString(strings.TrimPrefix(hexAddr, "0x"))
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func eip712DomainSeparator() []byte {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	enc := bytes.Join([][]byte{
		domainTypeHash,
		crypto.Keccak256([]byte(skrDomainName)),
		crypto.Keccak256([]byte(skrDomainVersion)),
		abiUint256(big.NewInt(skrChainID)),
		abiAddress(skrContract),
	}, nil)
	return crypto.Keccak256(enc)
}

func keyRequestDigest(requestFid int64, keyBytes []byte, deadline int64) []byte {
	typeHash := crypto.Keccak256([]byte(skrTypeString))
	structHash := crypto.Keccak256(bytes.Join([][]byte{
		typeHash,
		abiUint256(big.NewInt(requestFid)),
		crypto.Keccak256(keyBytes),
		abiUint256(big.NewInt(deadline)),
	}, nil))
	return crypto.Keccak256(append([]byte{0x19, 0x01}, append(eip712DomainSeparator(), structHash...)...))
}

// signKeyRequest signs (requestFid, keyBytes, deadline) per EIP-712.
// Returns 0x hex signature with v=27/28 like viem's signTypedData.
func signKeyRequest(privHex string, requestFid int64, keyBytes []byte, deadline int64) (string, error) {
	key, err := crypto.ToECDSA(mustHex(privHex))
	if err != nil {
		return "", err
	}
	digest := keyRequestDigest(requestFid, keyBytes, deadline)
	sig, err := crypto.Sign(digest, key)
	if err != nil {
		return "", err
	}
	sig[64] += 27
	return "0x" + hex.EncodeToString(sig), nil
}

func mustHex(s string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		panic(err)
	}
	return b
}

// deriveKey turns a BIP-39 mnemonic into the default Ethereum account (m/44'/60'/0'/0/0).
func deriveKey(mnemonic string) (privHex string, address string, err error) {
	seed := bip39.NewSeed(strings.TrimSpace(mnemonic), "")
	master, err := bip32.NewMasterKey(seed)
	if err != nil {
		return "", "", err
	}
	// path m/44'/60'/0'/0/0
	path := []uint32{
		44 + bip32.FirstHardenedChild,
		60 + bip32.FirstHardenedChild,
		0 + bip32.FirstHardenedChild,
		0,
		0,
	}
	k := master
	for _, i := range path {
		k, err = k.NewChildKey(i)
		if err != nil {
			return "", "", err
		}
	}
	priv := k.Key
	ecdsaKey, err := crypto.ToECDSA(priv)
	if err != nil {
		return "", "", err
	}
	addr := crypto.PubkeyToAddress(ecdsaKey.PublicKey)
	return hex.EncodeToString(priv), addr.Hex(), nil
}

func neynarGet(key, path string) ([]byte, int, error) {
	req, _ := http.NewRequest("GET", "https://api.neynar.com"+path, nil)
	req.Header.Set("x-api-key", key)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func neynarPost(key, path string, payload any) ([]byte, int, error) {
	raw, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.neynar.com"+path, bytes.NewReader(raw))
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

func lookupFidByCustody(key, custody string) (int64, error) {
	body, code, err := neynarGet(key, "/v2/farcaster/user/custody-address/?custody_address="+custody)
	if err != nil {
		return 0, err
	}
	if code != 200 {
		return 0, fmt.Errorf("neynar %d: %s", code, string(body))
	}
	var out struct {
		User struct {
			Fid int64 `json:"fid"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	if out.User.Fid == 0 {
		return 0, fmt.Errorf("no fid for custody address %s", custody)
	}
	return out.User.Fid, nil
}

func saveEnvValue(envPath, k, v string) error {
	lines := []string{}
	if data, err := os.ReadFile(envPath); err == nil {
		for _, ln := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(ln), k+"=") {
				continue
			}
			lines = append(lines, ln)
		}
	}
	// trim trailing empties, then append
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	lines = append(lines, k+"="+v, "")
	return os.WriteFile(envPath, []byte(strings.Join(lines, "\n")), 0600)
}

func signerSetup(key string) {
	if v := os.Getenv("SIGNER_UUID"); v != "" {
		body, code, err := neynarGet(key, "/v2/farcaster/signer/?signer_uuid="+v)
		if err == nil && code == 200 && strings.Contains(string(body), `"status":"approved"`) {
			fmt.Println("SIGNER_UUID already approved — nothing to do")
			return
		}
		fmt.Println("existing SIGNER_UUID not approved — making a fresh one")
	}

	mnemonic := os.Getenv("FARCASTER_DEVELOPER_MNEMONIC")
	if mnemonic == "" {
		fmt.Fprintln(os.Stderr, "missing FARCASTER_DEVELOPER_MNEMONIC.")
		fmt.Fprintln(os.Stderr, "Use your Farcaster recovery phrase from Warpcast Settings.")
		fmt.Fprintln(os.Stderr, "It stays on this laptop (chmod 600 .env) and only signs locally.")
		os.Exit(1)
	}
	privHex, addr, err := deriveKey(mnemonic)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad mnemonic:", err)
		os.Exit(1)
	}
	fmt.Println("developer address:", addr)

	appFid := int64(0)
	if s := os.Getenv("APP_FID"); s != "" {
		appFid, _ = strconv.ParseInt(s, 10, 64)
	} else if s := os.Getenv("FEED_FID"); s != "" {
		appFid, _ = strconv.ParseInt(s, 10, 64) // personal setup: app == you
	}
	if appFid == 0 {
		appFid, err = lookupFidByCustody(key, addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot find fid for developer address:", err)
			fmt.Fprintln(os.Stderr, "Set APP_FID in .env to your app account fid and re-run.")
			os.Exit(1)
		}
	}
	fmt.Println("app fid:", appFid)

	// 1. create signer
	body, code, err := neynarPost(key, "/v2/farcaster/signer/", map[string]string{})
	if err != nil || code != 200 {
		fmt.Fprintf(os.Stderr, "create signer failed: %v %d %s\n", err, code, string(body))
		os.Exit(1)
	}
	var created struct {
		SignerUUID string `json:"signer_uuid"`
		PublicKey  string `json:"public_key"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(body, &created); err != nil || created.SignerUUID == "" {
		fmt.Fprintln(os.Stderr, "create signer bad response:", string(body))
		os.Exit(1)
	}
	fmt.Println("signer:", created.SignerUUID, "status:", created.Status)

	// 2. sign key request with developer mnemonic (24h expiry)
	deadline := time.Now().Unix() + 86400
	pubBytes, err := hex.DecodeString(strings.TrimPrefix(created.PublicKey, "0x"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bad public key:", err)
		os.Exit(1)
	}
	sig, err := signKeyRequest(privHex, appFid, pubBytes, deadline)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign failed:", err)
		os.Exit(1)
	}

	// 3. register (Neynar sponsors gas, charges credits)
	regBody, code, err := neynarPost(key, "/v2/farcaster/signer/signed_key/", map[string]any{
		"signer_uuid": created.SignerUUID,
		"app_fid":     appFid,
		"deadline":    deadline,
		"signature":   sig,
		"sponsor":     map[string]any{"sponsored_by_neynar": true},
	})
	if err != nil || code != 200 {
		fmt.Fprintf(os.Stderr, "register failed: %v %d %s\n", err, code, string(regBody))
		os.Exit(1)
	}
	var reg struct {
		Status           string `json:"status"`
		SignerApprovalURL string `json:"signer_approval_url"`
	}
	_ = json.Unmarshal(regBody, &reg)
	fmt.Println("status:", reg.Status)
	if reg.SignerApprovalURL == "" {
		fmt.Fprintln(os.Stderr, "no approval url in response:", string(regBody))
		os.Exit(1)
	}
	fmt.Println()
	fmt.Println("Open this on your phone and approve:")
	fmt.Println(reg.SignerApprovalURL)
	fmt.Println("Then press ENTER here.")
	bufio.NewReader(os.Stdin).ReadString('\n')

	for i := 1; i <= 6; i++ {
		stBody, code, err := neynarGet(key, "/v2/farcaster/signer/?signer_uuid="+created.SignerUUID)
		fmt.Printf("poll %d (http %d): %.120s\n", i, code, string(stBody))
		if err == nil && code == 200 && strings.Contains(string(stBody), `"status":"approved"`) {
			envPath := "../.env"
			if _, err := os.Stat(envPath); err != nil {
				envPath = ".env"
			}
			if err := saveEnvValue(envPath, "SIGNER_UUID", created.SignerUUID); err != nil {
				fmt.Fprintln(os.Stderr, "save failed:", err)
				os.Exit(1)
			}
			fmt.Println("saved SIGNER_UUID to", envPath, "(mode 0600)")
			fmt.Println("try: ./post.sh \"hello from my bar\"")
			return
		}
		time.Sleep(10 * time.Second)
	}
	fmt.Fprintln(os.Stderr, "not approved yet — approve on phone, then re-run (it will reuse checks). UUID:", created.SignerUUID)
}
