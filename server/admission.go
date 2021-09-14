package server

import (
	"fmt"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"strings"
)

// PatchOperation describes an operation done to modify a Kubernetes
// resource
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}


// Volume contains details about one specific volume mounted to
// Toolforge Kubernetes containers
type Volume struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	Type     corev1.HostPathType `json:"type"`
	ReadOnly bool                `json:"readOnly"`
}

// VolumeAdmission type is what does all the magic
type VolumeAdmission struct {
	Volumes []Volume
}

// HandleAdmission has all the webhook logic to possibly mount volumes
// to containers if needed
func (admission *VolumeAdmission) HandleAdmission(review *admissionv1.AdmissionReview) {
	req := review.Request

	var pod corev1.Pod
	err := json.Unmarshal(req.Object.Raw, &pod)
	if err != nil {
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

	if !strings.HasPrefix(req.Namespace, "tool-") {
		logrus.Warningf("Skipping non-tool namespace %v", req.Namespace)

		review.Response = &admissionv1.AdmissionResponse{
			UID: review.Request.UID,
			Result: &metav1.Status{
				Message: "Only tools can have tool volumes mounted to them",
			},
		}

		return
	}

	// TODO: remove after PodPreset migration is done
	if _, exists := pod.Annotations["podpreset.admission.kubernetes.io/podpreset-mount-toolforge-vols"]; exists {
		review.Response = &admissionv1.AdmissionResponse{
			UID:     review.Request.UID,
			Allowed: true,
			Result: &metav1.Status{
				Message: "Volumes already mounted from a pod preset",
			},
		}

		return
	}

	toolName := strings.Replace(req.Namespace, "tool-", "", 1)

	var p []PatchOperation

	for _, volume := range admission.Volumes {
		var volumeType = volume.Type
		patch := PatchOperation{
			Op:   "add",
			Path: "/spec/volumes/-",
			Value: &corev1.Volume{
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: volume.Path,
						Type: &volumeType,
					},
				},
				Name: volume.Name,
			},
		}
		p = append(p, patch)

		for i := range pod.Spec.Containers {
			patch := PatchOperation{
				Op:   "add",
				Path: fmt.Sprintf("/spec/containers/%d/volumeMounts/-", i),
				Value: &corev1.VolumeMount{
					MountPath: volume.Path,
					Name:      volume.Name,
					ReadOnly:  volume.ReadOnly,
				},
			}
			p = append(p, patch)
		}
	}

	for i := range pod.Spec.Containers {
		// I have no clue why this is not required for volumes or volume mounts,
		// but if there are no env vars set the array does not exist and if it's not
		// set the add new array entry patch would fail. (Don't ask why I know.)
		if pod.Spec.Containers[i].Env == nil {
			patch := PatchOperation{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/env", i),
				Value: []corev1.EnvVar{},
			}
			p = append(p, patch)
		}

		patch := PatchOperation{
			Op:   "add",
			Path: fmt.Sprintf("/spec/containers/%d/env/-", i),
			Value: &corev1.EnvVar{
				Name:  "HOME",
				Value: fmt.Sprintf("/data/project/%v", toolName),
			},
		}
		p = append(p, patch)
	}

	patchType := admissionv1.PatchTypeJSONPatch

	response := &admissionv1.AdmissionResponse{
		UID:       review.Request.UID,
		PatchType: &patchType,
		Allowed:   true,
		Result: &metav1.Status{
			Message: "Volumes mounted",
		},
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

	review.Response = response
	return
}
