package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// Well-known vector: Hardhat/Foundry test mnemonic account #0.
func TestDeriveTestMnemonic(t *testing.T) {
	_, addr, err := deriveKey("test test test test test test test test test test test junk")
	if err != nil {
		t.Fatal(err)
	}
	want := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	if !strings.EqualFold(addr, want) {
		t.Fatalf("derive mismatch: got %s want %s", addr, want)
	}
}

// Domain must match Neynar SignedKeyRequestValidator or register 400s.
func TestSignerDomainMatchesValidator(t *testing.T) {
	if skrDomainName != "Farcaster SignedKeyRequestValidator" {
		t.Fatalf("domain name drift: got %q", skrDomainName)
	}
	if !strings.EqualFold(skrContract, "0x00000000Fc700472606ED4FA22623Acf62c60553") {
		t.Fatalf("contract drift: got %s", skrContract)
	}
}
// Roundtrip: signature must recover to the signing address.
func TestSignKeyRequestRecovers(t *testing.T) {
	privHex, addr, err := deriveKey("test test test test test test test test test test test junk")
	if err != nil {
		t.Fatal(err)
	}
	sigHex, err := signKeyRequest(privHex, 10695, make([]byte, 32), 9999999999)
	if err != nil {
		t.Fatal(err)
	}
	sig := mustHex(sigHex)
	sig[64] -= 27 // back to 0/1 for recovery
	// rebuild digest the same way signKeyRequest does
	digest := keyRequestDigest(10695, make([]byte, 32), 9999999999)
	pub, err := crypto.SigToPub(digest, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got := crypto.PubkeyToAddress(*pub).Hex(); !strings.EqualFold(got, addr) {
		t.Fatalf("recover mismatch: got %s want %s", got, addr)
	}
}
