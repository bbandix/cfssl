// Package ocspsign implements the ocspsign command.
package ocspsign

import (
	"io/ioutil"
	"time"

	"github.com/bbandix/cfssl/cli"
	"github.com/bbandix/cfssl/helpers"
	"github.com/bbandix/cfssl/log"
	"github.com/bbandix/cfssl/ocsp"
)

// Usage text of 'cfssl ocspsign'
var ocspSignerUsageText = `cfssl ocspsign -- signs an OCSP response for a given CA, cert, and status.
Returns a base64-encoded OCSP response.

Usage of ocspsign:
        cfssl ocspsign -ca cert -responder cert -responder-key key -cert cert [-status status] [-reason code] [-revoked-at YYYY-MM-DD]

Flags:
`

// Flags of 'cfssl ocspsign'
var ocspSignerFlags = []string{"ca", "responder", "responder-key", "reason", "status", "revoked-at", "interval"}

// ocspSignerMain is the main CLI of OCSP signer functionality.
func ocspSignerMain(args []string, c cli.Config) (err error) {
	// Read the cert to be revoked from file
	certBytes, err := ioutil.ReadFile(c.CertFile)
	if err != nil {
		log.Critical("Unable to read certificate: ", err)
		return
	}
	cert, err := helpers.ParseCertificatePEM(certBytes)
	if err != nil {
		log.Critical("Unable to parse certificate: ", err)
		return
	}

	req := ocsp.SignRequest{
		Certificate: cert,
		Status:      c.Status,
	}

	if c.Status == "revoked" {
		req.Reason = c.Reason

		req.RevokedAt = time.Now()
		if c.RevokedAt != "now" {
			req.RevokedAt, err = time.Parse("2006-01-02", c.RevokedAt)
			if err != nil {
				log.Critical("Malformed revocation time: ", c.RevokedAt)
				return
			}
		}
	}

	s, err := SignerFromConfig(c)
	if err != nil {
		log.Critical("Unable to create OCSP signer: ", err)
		return
	}

	resp, err := s.Sign(req)
	if err != nil {
		log.Critical("Unable to sign OCSP response: ", err)
		return
	}

	cli.PrintOCSPResponse(resp)
	return
}

// SignerFromConfig creates a signer from a cli.Config as a helper for cli and serve
func SignerFromConfig(c cli.Config) (ocsp.Signer, error) {
	//if this is called from serve then we need to use the specific responder key file
	//fallback to key for backwards-compatibility
	k := c.ResponderKeyFile
	if k == "" {
		k = c.KeyFile
	}
	return ocsp.NewSignerFromFile(c.CAFile, c.ResponderFile, k, time.Duration(c.Interval))
}

// CLISigner assembles the definition of Command 'sign'
var Command = &cli.Command{UsageText: ocspSignerUsageText, Flags: ocspSignerFlags, Main: ocspSignerMain}
