package scan

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/cloudflare/cf-tls/tls"
	"github.com/bbandix/cfssl/helpers"
)

// Sentinel for failures in sayHello. Should always be caught.
var errHelloFailed = errors.New("Handshake failed in sayHello")

// TLSHandshake contains scanners testing host cipher suite negotiation
var TLSHandshake = &Family{
	Description: "Scans for host's SSL/TLS version and cipher suite negotiation",
	Scanners: map[string]*Scanner{
		"CipherSuite": {
			"Determines host's cipher suites accepted and prefered order",
			cipherSuiteScan,
		},
		"SigAlgs": {
			"Determines host's accepted signature and hash algorithms",
			sigAlgsScan,
		},
		"CertsBySigAlgs": {
			"Determines host's certificate signature algorithm matching client's accepted signature and hash algorithms",
			certSigAlgsScan,
		},
		"CertsByCiphers": {
			"Determines host's certificate signature algorithm matching client's accepted ciphers",
			certSigAlgsScanByCipher,
		},
	},
}

func getCipherIndex(ciphers []uint16, serverCipher uint16) (cipherIndex int, err error) {
	//func getCipherIndex(ciphers []uint16, serverCipher uint16) (cipherIndex int, err error) {
	//	fmt.Println(serverCipher, ciphers)
	var cipherID uint16
	for cipherIndex, cipherID = range ciphers {
		if serverCipher == cipherID {
			return
		}
	}
	err = fmt.Errorf("server negotiated ciphersuite we didn't send: %s", tls.CipherSuites[serverCipher])
	return
}

func getCurveIndex(curves []tls.CurveID, serverCurve tls.CurveID) (curveIndex int, err error) {
	var curveID tls.CurveID
	for curveIndex, curveID = range curves {
		if serverCurve == curveID {
			return
		}
	}
	err = fmt.Errorf("server negotiated elliptic curve we didn't send: %s", tls.Curves[serverCurve])
	return
}

func sayHello(addr, hostname string, ciphers []uint16, curves []tls.CurveID, vers uint16, sigAlgs []tls.SignatureAndHash) (cipherIndex, curveIndex int, certs [][]byte, err error) {
	tcpConn, err := net.Dial(Network, addr)
	if err != nil {
		return
	}
	config := defaultTLSConfig(hostname)
	config.MinVersion = vers
	config.MaxVersion = vers
	if ciphers == nil {
		ciphers = allCiphersIDs()
	}
	config.CipherSuites = ciphers

	if curves == nil {
		curves = allCurvesIDs()
	}
	config.CurvePreferences = curves

	if sigAlgs == nil {
		sigAlgs = tls.AllSignatureAndHashAlgorithms
	}
	tls.SetSupportedSKXSignatureAlgorithms(sigAlgs)
	defer tls.ResetSupportedSKXSignatureAlgorithms()

	conn := tls.Client(tcpConn, config)
	serverCipher, serverCurveType, serverCurve, serverVersion, certificates, err := conn.SayHello()
	certs = certificates
	conn.Close()
	if err != nil {
		err = errHelloFailed
		return
	}

	if serverVersion != vers {
		err = fmt.Errorf("server negotiated protocol version we didn't send: %s", tls.Versions[serverVersion])
		return
	}

	cipherIndex, err = getCipherIndex(ciphers, serverCipher)

	if tls.CipherSuites[serverCipher].EllipticCurve {
		if curves == nil {
			curves = allCurvesIDs()
		}
		if serverCurveType != 3 {
			err = fmt.Errorf("server negotiated non-named ECDH parameters; we didn't analyze them. Server curve type: %d", serverCurveType)
		}
		curveIndex, err = getCurveIndex(curves, serverCurve)
	}

	return
}

func allCiphersIDs() []uint16 {
	ciphers := make([]uint16, 0, len(tls.CipherSuites))
	for cipherID := range tls.CipherSuites {
		ciphers = append(ciphers, cipherID)
	}
	return ciphers
}

func allCurvesIDs() []tls.CurveID {
	curves := make([]tls.CurveID, 0, len(tls.Curves))
	for curveID := range tls.Curves {
		// No unassigned or explicit curves in the scan, per http://tools.ietf.org/html/rfc4492#section-5.4
		if curveID == 0 || curveID == 65281 || curveID == 65282 {
			continue
		} else {
			curves = append(curves, curveID)
		}
	}
	return curves
}

type cipherDatum struct {
	versionID uint16
	curves    []tls.CurveID
}

// cipherVersions contains lists of host's supported cipher suites based on SSL/TLS Version.
// If a cipher suite uses ECC, also contains a list of supported curves by SSL/TLS Version.
type cipherVersions struct {
	cipherID uint16
	data     []cipherDatum
}

type cipherVersionList []cipherVersions

func (cvList cipherVersionList) String() string {
	cvStrings := make([]string, len(cvList))
	for i, c := range cvList {
		versStrings := make([]string, len(c.data))
		for j, d := range c.data {
			curveStrings := make([]string, len(d.curves))
			for k, c := range d.curves {
				curveStrings[k] = tls.Curves[c]
			}
			versStrings[j] = fmt.Sprintf("%s: [ %s ]", tls.Versions[d.versionID], strings.Join(curveStrings, ","))
		}
		cvStrings[i] = fmt.Sprintf("%s\t%s", tls.CipherSuites[c.cipherID], strings.Join(versStrings, ","))
	}
	return strings.Join(cvStrings, "\n")
}

func (cvList cipherVersionList) MarshalJSON() ([]byte, error) {
	b := new(bytes.Buffer)
	cvStrs := make([]string, len(cvList))
	for i, cv := range cvList {
		versStrings := make([]string, len(cv.data))
		for j, d := range cv.data {
			curveStrings := make([]string, len(d.curves))
			if len(d.curves) > 0 {
				for k, c := range d.curves {
					curveStrings[k] = fmt.Sprintf("\"%s\"", tls.Curves[c])
				}
				versStrings[j] = fmt.Sprintf("{\"%s\":[%s]}", tls.Versions[d.versionID], strings.Join(curveStrings, ","))
			} else {
				versStrings[j] = fmt.Sprintf("\"%s\"", tls.Versions[d.versionID])
			}
		}
		cvStrs[i] = fmt.Sprintf("{\"%s\":[%s]}", tls.CipherSuites[cv.cipherID].String(), strings.Join(versStrings, ","))
	}
	fmt.Fprintf(b, "[%s]", strings.Join(cvStrs, ","))
	return b.Bytes(), nil
}

func doCurveScan(addr, hostname string, vers, cipherID uint16, ciphers []uint16) (supportedCurves []tls.CurveID, err error) {
	allCurves := allCurvesIDs()
	curves := make([]tls.CurveID, len(allCurves))
	copy(curves, allCurves)
	for len(curves) > 0 {
		var curveIndex int
		_, curveIndex, _, err = sayHello(addr, hostname, []uint16{cipherID}, curves, vers, nil)
		if err != nil {
			// This case is expected, because eventually we ask only for curves the server doesn't support
			if err == errHelloFailed {
				err = nil
				break
			}
			return
		}
		curveID := curves[curveIndex]
		supportedCurves = append(supportedCurves, curveID)
		curves = append(curves[:curveIndex], curves[curveIndex+1:]...)
	}
	return
}

// cipherSuiteScan returns, by TLS Version, the sort list of cipher suites
// supported by the host
func cipherSuiteScan(addr, hostname string) (grade Grade, output Output, err error) {
	var cvList cipherVersionList
	allCiphers := allCiphersIDs()

	var vers uint16
	for vers = tls.VersionTLS12; vers >= tls.VersionSSL30; vers-- {
		ciphers := make([]uint16, len(allCiphers))
		copy(ciphers, allCiphers)
		for len(ciphers) > 0 {
			var cipherIndex int
			cipherIndex, _, _, err = sayHello(addr, hostname, ciphers, nil, vers, nil)
			if err != nil {
				if err == errHelloFailed {
					err = nil
					break
				}
				return
			}
			if vers == tls.VersionSSL30 {
				grade = Warning
			}
			cipherID := ciphers[cipherIndex]

			// If this is an EC cipher suite, do a second scan for curve support
			var supportedCurves []tls.CurveID
			if tls.CipherSuites[cipherID].EllipticCurve {
				supportedCurves, err = doCurveScan(addr, hostname, vers, cipherID, ciphers)
				if len(supportedCurves) == 0 {
					err = errors.New("couldn't negotiate any curves")
				}
			}
			for i, c := range cvList {
				if cipherID == c.cipherID {
					cvList[i].data = append(c.data, cipherDatum{vers, supportedCurves})
					goto exists
				}
			}
			cvList = append(cvList, cipherVersions{cipherID, []cipherDatum{cipherDatum{vers, supportedCurves}}})
		exists:
			ciphers = append(ciphers[:cipherIndex], ciphers[cipherIndex+1:]...)
		}
	}

	if len(cvList) == 0 {
		err = errors.New("couldn't negotiate any cipher suites")
		return
	}

	if grade != Warning {
		grade = Good
	}

	output = cvList
	return
}

// sigAlgsScan returns the accepted signature and hash algorithms of the host
func sigAlgsScan(addr, hostname string) (grade Grade, output Output, err error) {
	var supportedSigAlgs []tls.SignatureAndHash
	for _, sigAlg := range tls.AllSignatureAndHashAlgorithms {
		_, _, _, e := sayHello(addr, hostname, nil, nil, tls.VersionTLS12, []tls.SignatureAndHash{sigAlg})
		if e == nil {
			supportedSigAlgs = append(supportedSigAlgs, sigAlg)
		}
	}

	if len(supportedSigAlgs) > 0 {
		grade = Good
		output = supportedSigAlgs
	} else {
		err = errors.New("no SigAlgs supported")
	}
	return
}

// certSigAlgScan returns the server certificate with various sigature and hash algorithms in the ClientHello
func certSigAlgsScan(addr, hostname string) (grade Grade, output Output, err error) {
	var certSigAlgs = make(map[string]string)
	for _, sigAlg := range tls.AllSignatureAndHashAlgorithms {
		_, _, derCerts, e := sayHello(addr, hostname, nil, nil, tls.VersionTLS12, []tls.SignatureAndHash{sigAlg})
		if e == nil {
			if len(derCerts) == 0 {
				return Bad, nil, errors.New("no certs returned")
			}
			certs, _, err := helpers.ParseCertificatesDER(derCerts[0], "")
			if err != nil {
				return Bad, nil, err
			}

			certSigAlgs[sigAlg.String()] = helpers.SignatureString(certs[0].SignatureAlgorithm)
			//certSigAlgs = append(certSigAlgs, certs[0].SignatureAlgorithm)
		}
	}

	if len(certSigAlgs) > 0 {
		grade = Good
		output = certSigAlgs
	} else {
		err = errors.New("no SigAlgs supported")
	}
	return

}

// certSigAlgScan returns the server certificate with various ciphers in the ClientHello
func certSigAlgsScanByCipher(addr, hostname string) (grade Grade, output Output, err error) {
	var certSigAlgs = make(map[string]string)
	for cipherID := range tls.CipherSuites {
		_, _, derCerts, e := sayHello(addr, hostname, []uint16{cipherID}, nil, tls.VersionTLS12, []tls.SignatureAndHash{})
		if e == nil {
			if len(derCerts) == 0 {
				return Bad, nil, errors.New("no certs returned")
			}
			certs, _, err := helpers.ParseCertificatesDER(derCerts[0], "")
			if err != nil {
				return Bad, nil, err
			}

			certSigAlgs[tls.CipherSuites[cipherID].Name] = helpers.SignatureString(certs[0].SignatureAlgorithm)
			//certSigAlgs = append(certSigAlgs, certs[0].SignatureAlgorithm)
		}
	}

	if len(certSigAlgs) > 0 {
		grade = Good
		output = certSigAlgs
	} else {
		err = errors.New("no cipher supported")
	}
	return
}
