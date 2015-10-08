// +build !nopkcs11

// Package pkcs11 in the ocsp directory provides a way to construct a
// PKCS#11-based OCSP signer.
package pkcs11

import (
	"io/ioutil"
	"github.com/bbandix/cfssl/crypto/pkcs11key"
	"github.com/bbandix/cfssl/errors"
	"github.com/bbandix/cfssl/helpers"
	"github.com/bbandix/cfssl/log"
	"github.com/bbandix/cfssl/ocsp"
	ocspConfig "github.com/bbandix/cfssl/ocsp/config"
)

// Enabled is set to true if PKCS #11 support is present.
const Enabled = true

// NewPKCS11Signer returns a new PKCS #11 signer.
func NewPKCS11Signer(cfg ocspConfig.Config) (ocsp.Signer, error) {
	log.Debugf("Loading PKCS #11 module %s", cfg.PKCS11.Module)
	certData, err := ioutil.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, errors.New(errors.CertificateError, errors.ReadFailed)
	}

	cert, err := helpers.ParseCertificatePEM(certData)
	if err != nil {
		return nil, err
	}

	PKCS11 := cfg.PKCS11
	priv, err := pkcs11key.New(
		PKCS11.Module,
		PKCS11.TokenLabel,
		PKCS11.PIN,
		PKCS11.PrivateKeyLabel)
	if err != nil {
		return nil, errors.New(errors.PrivateKeyError, errors.ReadFailed)
	}

	return ocsp.NewSigner(cert, cert, priv, cfg.Interval)
}
