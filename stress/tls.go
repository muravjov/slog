package stress

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type TLSConfig struct {
	CA         string `toml:"ca"`
	Cert       string `toml:"cert"`
	Key        string `toml:"key"`
	SkipVerify bool   `toml:"skip_verify"`
	ServerName string `toml:"server_name"`
}

func makeTLSConfig(ca, cert, key string, isCLient bool) (tlsConfig *tls.Config) {
	if ca != "" && cert != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			log.Fatal(err)
		}
		certificates := []tls.Certificate{cert}

		caCertPool := x509.NewCertPool()
		// Load CA cert
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			log.Fatal(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)

		if isCLient {
			tlsConfig = &tls.Config{
				Certificates: certificates,
				RootCAs:      caCertPool,
			}
		} else {
			tlsConfig = &tls.Config{
				Certificates: certificates,
				ClientCAs:    caCertPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			}
		}
	}
	return
}

func makeClientTLSConfig(t TLSConfig) (tlsConfig *tls.Config) {
	tlsConfig = makeTLSConfig(t.CA, t.Cert, t.Key, true)
	if tlsConfig != nil {
		if t.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}

		if t.ServerName != "" {
			tlsConfig.ServerName = t.ServerName
		}
	}
	return
}

func readToml(tlsFname string, tls interface{}) {
	contents, err := ioutil.ReadFile(tlsFname)
	if err != nil {
		log.Fatalf("Error reading TLS config: %s", tlsFname)
	}

	if _, err := toml.Decode(string(contents), tls); err != nil {
		log.Fatalf("Error parsing toml: %s", tlsFname)
	}
}

func NewTLSConfig(tlsFname string) *tls.Config {
	var tls TLSConfig
	readToml(tlsFname, &tls)
	return makeClientTLSConfig(tls)
}

type ServerTLSConfig struct {
	CA   string `toml:"ca"`
	Cert string `toml:"cert"`
	Key  string `toml:"key"`
}

func NewServerTLSConfig(fname string) *tls.Config {
	var tls ServerTLSConfig
	readToml(fname, &tls)

	return makeTLSConfig(tls.CA, tls.Cert, tls.Key, false)
}
