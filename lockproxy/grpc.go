package lockproxy

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"

	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func GrpcDialTransportSecurity(
	certPath string,
	keyPath string,
	caPath string,
) (grpc.DialOption, error) {
	if certPath == "" && keyPath == "" && caPath == "" {
		return grpc.WithInsecure(), nil
	}

	clientCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, xerrors.Errorf("failed to load X509 key pair: %s, %s: %w", certPath, keyPath, err)
	}

	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, xerrors.Errorf("failed to load CA cert: %s: %w", caPath, err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, xerrors.Errorf("credentials: failed to append certificates")
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
	})

	return grpc.WithTransportCredentials(creds), nil
}
