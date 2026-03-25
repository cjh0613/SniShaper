package cert

import (
	"crypto/sha1"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CertManager struct {
	caCert     *x509.Certificate
	caKey      *rsa.PrivateKey
	certCache  map[string]*tls.Certificate
	certMu     sync.RWMutex
	caPath     string
	certPath   string
	keyPath    string
}

func NewCertManager(caPath, certPath, keyPath string) *CertManager {
	return &CertManager{
		certCache: make(map[string]*tls.Certificate),
		caPath:    caPath,
		certPath:  certPath,
		keyPath:   keyPath,
	}
}

func (cm *CertManager) GenerateCA() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SniShaper"},
			CommonName:   "SniShaper CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return err
	}

	cm.caCert = cert
	cm.caKey = privateKey

	if err := cm.saveCA(); err != nil {
		return err
	}

	fmt.Println("[Cert] CA certificate generated successfully")
	return nil
}

func (cm *CertManager) saveCA() error {
	caFile, err := os.Create(cm.caPath)
	if err != nil {
		return err
	}
	defer caFile.Close()

	if err := pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: cm.caCert.Raw}); err != nil {
		return err
	}

	keyFile, err := os.Create(cm.keyPath)
	if err != nil {
		return err
	}
	defer keyFile.Close()

	keyBytes := x509.MarshalPKCS1PrivateKey(cm.caKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return err
	}

	return nil
}

func (cm *CertManager) LoadCA() error {
	caData, err := os.ReadFile(cm.caPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cm.GenerateCA()
		}
		return err
	}

	block, _ := pem.Decode(caData)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}

	keyData, err := os.ReadFile(cm.keyPath)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(keyData)
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	cm.caCert = cert
	cm.caKey = key

	return nil
}

func (cm *CertManager) GenerateDomainCert(domains []string) error {
	if cm.caCert == nil || cm.caKey == nil {
		if err := cm.LoadCA(); err != nil {
			return err
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	serial := big.NewInt(time.Now().UnixNano())
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domains,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, cm.caCert, &privateKey.PublicKey, cm.caKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	_, err = x509.ParseCertificate(certDER)
	if err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}

	cm.certMu.Lock()
	cm.certCache["default"] = &tlsCert
	cm.certMu.Unlock()

	certFile, err := os.Create(cm.certPath)
	if err != nil {
		return err
	}
	defer certFile.Close()

	certFile.Write(certPEM)
	certFile.Write(keyPEM)

	fmt.Printf("[Cert] Domain certificate generated for: %v\n", domains)
	return nil
}

func (cm *CertManager) GetCACertPath() string {
	return cm.caPath
}

func (cm *CertManager) GetCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	if cm.caCert != nil {
		pool.AddCert(cm.caCert)
	}
	return pool
}

func (cm *CertManager) GetDomainCert() (*tls.Certificate, error) {
	cm.certMu.RLock()
	defer cm.certMu.RUnlock()

	if cert, ok := cm.certCache["default"]; ok {
		return cert, nil
	}

	return nil, fmt.Errorf("no certificate loaded")
}

func (cm *CertManager) GetCA() *x509.Certificate {
	return cm.caCert
}

func (cm *CertManager) GetCACert() *x509.Certificate {
	return cm.caCert
}

func (cm *CertManager) GetCAKey() interface{} {
	return cm.caKey
}

func (cm *CertManager) IsCAInstalled() bool {
	status := cm.GetCAInstallStatus()
	return status.Installed
}

type CAInstallStatus struct {
	Installed   bool
	Platform    string
	CertPath    string
	InstallHelp string
}

func (cm *CertManager) GetCAInstallStatus() CAInstallStatus {
	status := CAInstallStatus{
		CertPath:    cm.caPath,
		Platform:    "windows",
		InstallHelp: "双击 CA 证书文件 -> 安装证书 -> 导入到\"受信任的根证书颁发机构\"（当前用户或本地计算机）",
	}

	if cm.caCert == nil {
		if err := cm.LoadCA(); err != nil {
			return status
		}
	}
	if cm.caCert == nil {
		return status
	}

	sum := sha1.Sum(cm.caCert.Raw)
	thumb := strings.ToUpper(hex.EncodeToString(sum[:]))

	psScript := fmt.Sprintf(`
		Add-Type -AssemblyName System.Security
		$thumb = '%s'
		$stores = @('Root', 'CA')
		$locations = @('CurrentUser', 'LocalMachine')
		foreach ($loc in $locations) {
			foreach ($name in $stores) {
				$store = New-Object System.Security.Cryptography.X509Certificates.X509Store($name, $loc)
				$store.Open('ReadOnly')
				$found = $store.Certificates.Find([System.Security.Cryptography.X509Certificates.X509FindType]::FindByThumbprint, $thumb, $false)
				$store.Close()
				if ($found.Count -gt 0) {
					Write-Output 'FOUND'
					exit 0
				}
			}
		}
	`, thumb)
	output, _ := outputHiddenCommand("powershell", "-NoProfile", "-Command", psScript)
	status.Installed = strings.Contains(strings.ToUpper(string(output)), "FOUND")
	return status
}

func (cm *CertManager) InstallCA() error {
	if cm.caPath == "" {
		return fmt.Errorf("CA certificate path is empty")
	}

	// Use certutil to install to CurrentUser Root store. 
	// This will pop up a standard Windows security warning.
	err := runHiddenCommand("certutil", "-user", "-addstore", "root", cm.caPath)
	if err != nil {
		return fmt.Errorf("failed to install CA certificate: %w", err)
	}

	fmt.Println("[Cert] CA certificate installed successfully to CurrentUser Root store")
	return nil
}
func (cm *CertManager) OpenCAFile() error {
	return startHiddenCommand("cmd", "/c", "start", "", cm.caPath)
}

func (cm *CertManager) GetCACertPEM() string {
	if cm.caCert == nil {
		return ""
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cm.caCert.Raw}))
}

func (cm *CertManager) RegenerateCA() error {
	cm.certMu.Lock()
	defer cm.certMu.Unlock()

	if err := cm.GenerateCA(); err != nil {
		return err
	}

	cm.certCache = make(map[string]*tls.Certificate)

	fmt.Println("[Cert] CA certificate regenerated successfully")
	return nil
}

func (cm *CertManager) ExportCert() ([]byte, error) {
	if cm.caCert == nil {
		return nil, fmt.Errorf("no CA certificate available")
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cm.caCert.Raw}), nil
}

func InitCertManager(certDir string) (*CertManager, error) {
	os.MkdirAll(certDir, 0755)

	cm := NewCertManager(
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "domain.crt"),
		filepath.Join(certDir, "domain.key"),
	)

	if err := cm.LoadCA(); err != nil {
		return nil, err
	}

	defaultDomains := []string{
		"google.com",
		"youtube.com",
		"github.com",
		"gstatic.com",
	}

	if err := cm.GenerateDomainCert(defaultDomains); err != nil {
		fmt.Printf("[Cert] Warning: failed to generate default cert: %v\n", err)
	}

	return cm, nil
}
