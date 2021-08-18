package server

import (
	"fmt"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Volume struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	Type     corev1.HostPathType `json:"type"`
	ReadOnly bool                `json:"readOnly"`
}

type VolumeAdmission struct {
	Volumes []Volume
}

func (admission *VolumeAdmission) HandleAdmission(review *admissionv1.AdmissionReview) {
	req := review.Request

	var pod corev1.Pod
	var err error

	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		logrus.Errorf("Could not unmarshal raw object: %v", err)
		review.Response = &admissionv1.AdmissionResponse{
			UID: review.Request.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}

		return
	}

	logrus.Debugf("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	response := &admissionv1.AdmissionResponse{
		UID: review.Request.UID,
	}

	var p []PatchOperation

	for _, volume := range admission.Volumes {
		patch := PatchOperation{
			Op:    "add",
			Path:  "/spec/volumes/-",
			Value: &corev1.Volume{
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: volume.Path,
						Type: &volume.Type,
					},
				},
				Name: volume.Name,
			},
		}
		p = append(p, patch)

		for i := range pod.Spec.Containers {
			patch := PatchOperation{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
				Value: &corev1.VolumeMount{
					MountPath: volume.Path,
					Name:      volume.Name,
					ReadOnly:  volume.ReadOnly,
				},
			}
			p = append(p, patch)
		}
	}

	// parse the []map into JSON
	response.Patch, err = json.Marshal(p)
	if err != nil {
		logrus.Errorf("Could not marshal patch object: %v", err)
		review.Response = &admissionv1.AdmissionResponse{
			UID: review.Request.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}

		return
	}

	patchType := admissionv1.PatchTypeJSONPatch
	response.PatchType = &patchType

	response.Allowed = true
	response.Result = &metav1.Status{
		Message: "Volumes mounted",
	}

	review.Response = response
	return
}
