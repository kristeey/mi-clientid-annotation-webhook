package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	http.HandleFunc("/mutate", handleMutate)
	log.Println("Starting webhook server on :8443...")
	certFile := "/etc/webhook/certs/tls.crt"
	keyFile := "/etc/webhook/certs/tls.key"
	if err := http.ListenAndServeTLS(":8443", certFile, keyFile, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read request", http.StatusBadRequest)
		return
	}

	var review admissionv1.AdmissionReview
	if err := json.Unmarshal(body, &review); err != nil {
		http.Error(w, "could not parse admission review", http.StatusBadRequest)
		return
	}

	if review.Request == nil || review.Request.Kind.Kind != "ServiceAccount" {
		writeResponse(w, admissionv1.AdmissionResponse{
			UID:     review.Request.UID,
			Allowed: true,
		})
		return
	}

	var sa corev1.ServiceAccount
	if err := json.Unmarshal(review.Request.Object.Raw, &sa); err != nil {
		http.Error(w, "could not parse service account", http.StatusBadRequest)
		return
	}

	labels := sa.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	// Azure MI label logic
	const miLabel = "mi.clientid.webhook/azure-mi-client-name"
	const clientIDLabel = "azure.workload.identity/client-id"
	if miName, ok := labels[miLabel]; ok && miName != "" {
		subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			http.Error(w, "failed to get Azure credential", http.StatusInternalServerError)
			return
		}
		msiClient, err := armmsi.NewUserAssignedIdentitiesClient(subID, cred, nil)
		if err != nil {
			http.Error(w, "failed to create Azure MSI client", http.StatusInternalServerError)
			return
		}
		// List all identities in the subscription and find by name
		pager := msiClient.NewListBySubscriptionPager(nil)
		var clientID string
		for pager.More() {
			resp, err := pager.NextPage(context.Background())
			if err != nil {
				http.Error(w, "failed to list managed identities", http.StatusInternalServerError)
				return
			}
			for _, id := range resp.Value {
				if id.Name != nil && *id.Name == miName && id.Properties != nil && id.Properties.ClientID != nil {
					clientID = *id.Properties.ClientID
					break
				}
			}
			if clientID != "" {
				break
			}
		}
		if clientID == "" {
			http.Error(w, "managed identity not found", http.StatusNotFound)
			return
		}
		patch := make([]map[string]interface{}, 0)
		// If the ServiceAccount has no labels, add the labels map first
		if sa.Labels == nil || len(sa.Labels) == 0 {
			patch = append(patch, map[string]interface{}{
				"op":    "add",
				"path":  "/metadata/labels",
				"value": map[string]string{},
			})
		}
		// Patch path must escape '/' as '~1' per JSON Patch spec
		patch = append(patch, map[string]interface{}{
			"op":    "add",
			"path":  "/metadata/labels/azure.workload.identity~1client-id",
			"value": clientID,
		})
		patchBytes, _ := json.Marshal(patch)
		writeResponse(w, admissionv1.AdmissionResponse{
			UID:       review.Request.UID,
			Allowed:   true,
			Patch:     patchBytes,
			PatchType: func() *admissionv1.PatchType { pt := admissionv1.PatchTypeJSONPatch; return &pt }(),
		})
		return
	}

	writeResponse(w, admissionv1.AdmissionResponse{
		UID:     review.Request.UID,
		Allowed: true,
	})
}

func writeResponse(w http.ResponseWriter, resp admissionv1.AdmissionResponse) {
	review := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &resp,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}
