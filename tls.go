package chancon

import (
	"crypto/tls"
	"fmt"
)

type tlsManager struct {
	tlsConfig *tls.Config
}

func newTlsManager() *tlsManager {
	return &tlsManager{}
}

func (s *tlsManager) SetSslConfig(config *tls.Config) {
	s.tlsConfig = config
}

func (s *tlsManager) loadTlsConfig() (*tls.Config, error) {
	if s.tlsConfig == nil {
		return nil, ErrNoSslConfig
	}

	return s.tlsConfig, nil
}

func CreateTlsConfigFromCertificate(certificatePath string, privateKeyPath string) *tls.Config {
	cer, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
	if err != nil {
		panic(fmt.Sprintf("Failed creating tls config: %s", err.Error()))
	}

	return &tls.Config{Certificates: []tls.Certificate{cer}}

}
