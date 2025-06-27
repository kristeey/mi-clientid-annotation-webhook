package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

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
	body, err := ioutil.ReadAll(r.Body)
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

	if val, ok := labels["hei"]; ok && val == "hallo" {
		labels["added-hei"] = "added-hallo"
		patch := []map[string]interface{}{
			{
				"op":    "add",
				"path":  "/metadata/labels/added-hei",
				"value": "added-hallo",
			},
		}
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
