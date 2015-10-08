// +build !nopkcs11

package pkcs11

import (
	"io/ioutil"

	"github.com/bbandix/cfssl/config"
	"github.com/bbandix/cfssl/crypto/pkcs11key"
	"github.com/bbandix/cfssl/errors"
	"github.com/bbandix/cfssl/helpers"
	"github.com/bbandix/cfssl/log"
	"github.com/bbandix/cfssl/signer"
	"github.com/bbandix/cfssl/signer/local"
)

// Enabled is set to true if PKCS #11 support is present.
const Enabled = true

// New returns a new PKCS #11 signer.
func New(caCertFile string, policy *config.Signing, cfg *pkcs11key.Config) (signer.Signer, error) {
	if cfg == nil {
		return nil, errors.New(errors.PrivateKeyError, errors.ReadFailed)
	}

	log.Debugf("Loading PKCS #11 module %s", cfg.Module)
	certData, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, errors.New(errors.PrivateKeyError, errors.ReadFailed)
	}

	cert, err := helpers.ParseCertificatePEM(certData)
	if err != nil {
		return nil, err
	}

	priv, err := pkcs11key.New(cfg.Module, cfg.TokenLabel, cfg.PIN, cfg.PrivateKeyLabel)
	if err != nil {
		return nil, errors.New(errors.PrivateKeyError, errors.ReadFailed)
	}
	sigAlgo := signer.DefaultSigAlgo(priv)

	return local.NewSigner(priv, cert, sigAlgo, policy)
}
