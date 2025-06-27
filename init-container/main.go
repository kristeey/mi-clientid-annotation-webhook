package main

import (
	context "context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	certDir := "/etc/webhook/certs"
	os.MkdirAll(certDir, 0700)

	caCert, caKey := generateCA()
	serverCert, serverKey := generateServerCert(caCert, caKey)

	writePem(certDir+"/tls.crt", "CERTIFICATE", serverCert)
	writePem(certDir+"/tls.key", "RSA PRIVATE KEY", serverKey)
	writePem(certDir+"/ca.crt", "CERTIFICATE", caCert)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert})

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	webhookCfg := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "serviceaccount-label-mutator",
		},
		Webhooks: []admissionv1.MutatingWebhook{{
			Name:                    "webhook-server.webhook-demo.svc",
			AdmissionReviewVersions: []string{"v1"},
			SideEffects:             func() *admissionv1.SideEffectClass { s := admissionv1.SideEffectClassNone; return &s }(),
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      "webhook-server",
					Namespace: "webhook-demo",
					Path:      func() *string { s := "/mutate"; return &s }(),
					Port:      func() *int32 { p := int32(443); return &p }(),
				},
				CABundle: []byte(caPEM),
			},
			Rules: []admissionv1.RuleWithOperations{{
				Rule: admissionv1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"v1"},
					Resources:   []string{"serviceaccounts"},
					Scope:       func() *admissionv1.ScopeType { s := admissionv1.ScopeType("Namespaced"); return &s }(),
				},
				Operations: []admissionv1.OperationType{"CREATE", "UPDATE"},
			}},
			FailurePolicy:     func() *admissionv1.FailurePolicyType { f := admissionv1.Fail; return &f }(),
			MatchPolicy:       func() *admissionv1.MatchPolicyType { m := admissionv1.Equivalent; return &m }(),
			ObjectSelector:    &metav1.LabelSelector{},
			NamespaceSelector: &metav1.LabelSelector{},
		}},
	}

	_, err = clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), webhookCfg, metav1.CreateOptions{})
	if err != nil {
		// Try update if already exists
		if errors.IsAlreadyExists(err) {
			// Fetch the existing config to get the resourceVersion
			existing, getErr := clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), webhookCfg.Name, metav1.GetOptions{})
			if getErr != nil {
				log.Fatalf("Failed to get existing MutatingWebhookConfiguration: %v", getErr)
			}
			webhookCfg.ResourceVersion = existing.ResourceVersion
			_, err = clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), webhookCfg, metav1.UpdateOptions{})
			if err != nil {
				log.Fatalf("Failed to update MutatingWebhookConfiguration: %v", err)
			}
		} else {
			log.Fatalf("Failed to create or update MutatingWebhookConfiguration: %v", err)
		}
	}
	log.Println("MutatingWebhookConfiguration applied successfully.")
}

func generateCA() ([]byte, *rsa.PrivateKey) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	certTmpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "webhook-ca"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &certTmpl, &certTmpl, &key.PublicKey, key)
	return certDER, key
}

func generateServerCert(caCertDER []byte, caKey *rsa.PrivateKey) ([]byte, []byte) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	caCert, _ := x509.ParseCertificate(caCertDER)
	certTmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "webhook-server.webhook-demo.svc"},
		DNSNames:     []string{"webhook-server", "webhook-server.webhook-demo", "webhook-server.webhook-demo.svc"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, &key.PublicKey, caKey)
	return certDER, x509.MarshalPKCS1PrivateKey(key)
}

func writePem(filename, typ string, derBytes []byte) {
	f, _ := os.Create(filename)
	pem.Encode(f, &pem.Block{Type: typ, Bytes: derBytes})
	f.Close()
}
