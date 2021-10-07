package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	volumes = []Volume{
		{
			Name:     "home",
			Path:     "/data/project",
			ReadOnly: false,
		},
		{
			Name:     "etc-ldap",
			Path:     "/etc/ldap",
			ReadOnly: true,
		},
	}
)

func decodeResponse(body io.ReadCloser) (*admissionv1.AdmissionReview, error) {
	response, _ := ioutil.ReadAll(body)
	review := &admissionv1.AdmissionReview{}
	decoder := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	_, _, err := decoder.Decode(response, nil, review)
	return review, err
}

func encodeRequest(review *admissionv1.AdmissionReview) []byte {
	ret, err := json.Marshal(review)
	if err != nil {
		logrus.Errorln(err)
	}
	return ret
}

func getResponse(request admissionv1.AdmissionReview) (*admissionv1.AdmissionReview, error) {
	admission := &VolumeAdmission{
		Volumes: volumes,
	}

	server := httptest.NewServer(GetAdmissionControllerServerNoSsl(admission, ":8080").Handler)
	requestString := string(encodeRequest(&request))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	return decodeResponse(r.Body)
}

func TestServerIgnoresNonToolPods(t *testing.T) {
	review, err := getResponse(admissionv1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: v1.GroupVersionKind{
				Group: "", Version: "v1", Kind: "pod",
			},
			Operation: "CREATE",
			Namespace: "maintain-kubeusers",
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"kind": "Pod",
					"apiVersion": "v1",
					"metadata": {
						"name": "maintain-kubeusers-123123123",
						"namespace": "maintain-kubeusers",
						"uid": "4b54be10-8d3c-11e9-8b7a-080027f5f85c",
						"creationTimestamp": "2019-06-12T18:02:51Z"
					},
					"spec": {
						"containers": [
							{
								"name": "maintain-kubeusers",
								"image": "docker-registry.tools.wmflabs.org/maintain-kubeusers:latest",
								"command": ["/app/venv/bin/python"],
								"args": ["/app/maintain_kubeusers.py"]
							}
						]
					}
				}`),
			},
		},
	})

	t.Log(review.Response)

	if err != nil {
		t.Error(err)
	}

	if review.Response.Allowed {
		t.Error("Should not allow non-tools")
	}

	if review.Response.Patch != nil {
		t.Error("Should not contain patch when not allowed")
	}
}

func TestServerMountsAllVolumesWhenNoneExist(t *testing.T) {
	review, err := getResponse(admissionv1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: v1.GroupVersionKind{
				Group: "", Version: "v1", Kind: "pod",
			},
			Operation: "CREATE",
			Namespace: "tool-test",
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"kind": "Pod",
					"apiVersion": "v1",
					"metadata": {
						"name": "test-123123123",
						"namespace": "tool-test",
						"uid": "4b54be10-8d3c-11e9-8b7a-080027f5f85c",
						"creationTimestamp": "2019-06-12T18:02:51Z"
					},
					"spec": {
						"containers": [
							{
								"name": "test",
								"image": "docker-registry.tools.wmflabs.org/toolforge-python39-web:latest",
								"command": ["/usr/bin/webservice-runner"],
								"args": ["python39"]
							}
						]
					}
				}`),
			},
		},
	})

	t.Log(review.Response)

	if err != nil {
		t.Error(err)
	}

	if !review.Response.Allowed {
		t.Error("Should allow tools")
	}

	if *review.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Error("Wrong patch type found")
	}

	var p []PatchOperation
	err = json.Unmarshal(review.Response.Patch, &p)
	if err != nil {
		t.Error(err)
	}

	if len(p) != 8 {
		t.Errorf("Patch length %d does not match expected value 8", len(p))
	}
}

func TestServerMountsAllVolumesIfSomeExist(t *testing.T) {
	review, err := getResponse(admissionv1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: v1.GroupVersionKind{
				Group: "", Version: "v1", Kind: "pod",
			},
			Operation: "CREATE",
			Namespace: "tool-test",
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"kind": "Pod",
					"apiVersion": "v1",
					"metadata": {
						"name": "test-123123123",
						"namespace": "tool-test",
						"uid": "4b54be10-8d3c-11e9-8b7a-080027f5f85c",
						"creationTimestamp": "2019-06-12T18:02:51Z"
					},
					"spec": {
						"containers": [
							{
								"name": "test",
								"image": "docker-registry.tools.wmflabs.org/toolforge-python39-web:latest",
								"command": ["/usr/bin/webservice-runner"],
								"args": ["python39"],
								"volumeMounts": [
									{
										"mountPath": "/var/run/secrets/kubernetes.io/serviceaccount",
										"name": "default-token-abcde",
										"readOnly": true
									}
								]
							}
						],
						    "volumes": [
							{
								"name": "default-token-abcde",
								"secret": {
									"defaultMode": 420,
									"secretName": "default-token-abcde"
								}
							}
						]
					}
				}`),
			},
		},
	})

	t.Log(review.Response)

	if err != nil {
		t.Error(err)
	}

	if !review.Response.Allowed {
		t.Error("Should allow tools")
	}

	if *review.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Error("Wrong patch type found")
	}

	var p []PatchOperation
	err = json.Unmarshal(review.Response.Patch, &p)
	if err != nil {
		t.Error(err)
	}

	if len(p) != 6 {
		t.Errorf("Patch length %d does not match expected value of 6", len(p))
	}
}

func TestServerMountsNeededVolumes(t *testing.T) {
	review, err := getResponse(admissionv1.AdmissionReview{
		TypeMeta: v1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: v1.GroupVersionKind{
				Group: "", Version: "v1", Kind: "pod",
			},
			Operation: "CREATE",
			Namespace: "tool-test",
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"kind": "Pod",
					"apiVersion": "v1",
					"metadata": {
						"name": "maintain-kubeusers-123123123",
						"namespace": "maintain-kubeusers",
						"uid": "4b54be10-8d3c-11e9-8b7a-080027f5f85c",
						"creationTimestamp": "2019-06-12T18:02:51Z"
					},
					"spec": {
						"containers": [
							{
								"name": "maintain-kubeusers",
								"image": "docker-registry.tools.wmflabs.org/maintain-kubeusers:latest",
								"command": ["/app/venv/bin/python"],
								"args": ["/app/maintain_kubeusers.py"],
								"env": [
									{
										"name": "HOME",
										"value": "/foobar"
									}
								],
								"volumeMounts": [
									{
										"mountPath": "/data/project",
										"name": "home"
									}
								]
							}
						],
						"volumes": [
							{
								"name": "home",
								"hostPath": {
									"path": "/data/project",
									"type": "Directory"
								}
							}
						]
					}
				}`),
			},
		},
	})

	t.Log(review.Response)

	if err != nil {
		t.Error(err)
	}

	if !review.Response.Allowed {
		t.Error("Should allow tools")
	}

	if *review.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Error("Wrong patch type found")
	}

	var p []PatchOperation
	err = json.Unmarshal(review.Response.Patch, &p)
	if err != nil {
		t.Error(err)
	}

	if len(p) != 2 {
		t.Error("Patch length does not match expected value")
	}
}
