package initca

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/bbandix/cfssl/config"
	"github.com/bbandix/cfssl/csr"
	"github.com/bbandix/cfssl/helpers"
	"github.com/bbandix/cfssl/signer"
	"github.com/bbandix/cfssl/signer/local"
)

var validKeyParams = []csr.BasicKeyRequest{
	{"rsa", 2048},
	{"rsa", 3072},
	{"rsa", 4096},
	{"ecdsa", 256},
	{"ecdsa", 384},
	{"ecdsa", 521},
}

var csrFiles = []string{
	"testdata/rsa2048.csr",
	"testdata/rsa3072.csr",
	"testdata/rsa4096.csr",
	"testdata/ecdsa256.csr",
	"testdata/ecdsa384.csr",
	"testdata/ecdsa521.csr",
}
var invalidCryptoParams = []csr.BasicKeyRequest{
	// Weak Key
	{"rsa", 1024},
	// Bad param
	{"rsaCrypto", 2048},
	{"ecdsa", 2000},
}

func TestInitCA(t *testing.T) {
	var req *csr.CertificateRequest
	hostname := "cloudflare.com"
	for _, param := range validKeyParams {
		req = &csr.CertificateRequest{
			Names: []csr.Name{
				{
					C:  "US",
					ST: "California",
					L:  "San Francisco",
					O:  "CloudFlare",
					OU: "Systems Engineering",
				},
			},
			CN:    hostname,
			Hosts: []string{hostname, "www." + hostname},
			KeyRequest: &param,
		}
		certBytes, _, keyBytes, err := New(req)
		if err != nil {
			t.Fatal("InitCA failed:", err)
		}
		key, err := helpers.ParsePrivateKeyPEM(keyBytes)
		if err != nil {
			t.Fatal("InitCA private key parsing failed:", err)
		}
		cert, err := helpers.ParseCertificatePEM(certBytes)
		if err != nil {
			t.Fatal("InitCA cert parsing failed:", err)
		}

		// Verify key parameters.
		switch req.KeyRequest.Algo() {
		case "rsa":
			if cert.PublicKey.(*rsa.PublicKey).N.BitLen() != param.Size() {
				t.Fatal("Cert key length mismatch.")
			}
			if key.(*rsa.PrivateKey).N.BitLen() != param.Size() {
				t.Fatal("Private key length mismatch.")
			}
		case "ecdsa":
			if cert.PublicKey.(*ecdsa.PublicKey).Curve.Params().BitSize != param.Size() {
				t.Fatal("Cert key length mismatch.")
			}
			if key.(*ecdsa.PrivateKey).Curve.Params().BitSize != param.Size() {
				t.Fatal("Private key length mismatch.")
			}
		}

		// Start a signer
		var CAPolicy = &config.Signing{
			Default: &config.SigningProfile{
				Usage:        []string{"cert sign", "crl sign"},
				ExpiryString: "300s",
				Expiry:       300 * time.Second,
				CA:           true,
			},
		}
		s, err := local.NewSigner(key, cert, signer.DefaultSigAlgo(key), nil)
		if err != nil {
			t.Fatal("Signer Creation error:", err)
		}
		s.SetPolicy(CAPolicy)

		// Sign RSA and ECDSA customer CSRs.
		for _, csrFile := range csrFiles {
			csrBytes, err := ioutil.ReadFile(csrFile)
			if err != nil {
				t.Fatal("CSR loading error:", err)
			}
			req := signer.SignRequest{
				Request: string(csrBytes),
				Hosts:   signer.SplitHosts(hostname),
				Profile: "",
				Label:   "",
			}

			bytes, err := s.Sign(req)
			if err != nil {
				t.Fatal(err)
			}
			customerCert, _ := helpers.ParseCertificatePEM(bytes)
			if customerCert.SignatureAlgorithm != s.SigAlgo() {
				t.Fatal("Signature Algorithm mismatch")
			}
			err = customerCert.CheckSignatureFrom(cert)
			if err != nil {
				t.Fatal("Signing CSR failed.", err)
			}
		}

	}
}

func TestInvalidCryptoParams(t *testing.T) {
	var req *csr.CertificateRequest
	hostname := "cloudflare.com"
	for _, invalidParam := range invalidCryptoParams {
		req = &csr.CertificateRequest{
			Names: []csr.Name{
				{
					C:  "US",
					ST: "California",
					L:  "San Francisco",
					O:  "CloudFlare",
					OU: "Systems Engineering",
				},
			},
			CN:    hostname,
			Hosts: []string{hostname, "www." + hostname},
			KeyRequest: &invalidParam,
		}
		_, _, _, err := New(req)
		if err == nil {
			t.Fatal("InitCA with bad params should fail:", err)
		}

		if !strings.Contains(err.Error(), `"code":2400`) {
			t.Fatal(err)
		}
	}
}

type validation struct {
	r *csr.CertificateRequest
	v bool
}

var testValidations = []validation{
	{&csr.CertificateRequest{}, false},
	{&csr.CertificateRequest{
		CN: "test CA",
	}, true},
	{&csr.CertificateRequest{
		Names: []csr.Name{csr.Name{}},
	}, false},
	{&csr.CertificateRequest{
		Names: []csr.Name{
			csr.Name{O: "Example CA"},
		},
	}, true},
}

func TestValidations(t *testing.T) {
	for i, tv := range testValidations {
		err := validator(tv.r)
		if tv.v && err != nil {
			t.Fatalf("%v", err)
		}

		if !tv.v && err == nil {
			t.Fatalf("%d: expected error, but no error was reported", i)
		}
	}
}
