// Package v1alpha1 contains API Schema definitions for the harbor v1alpha1 API group.
// +kubebuilder:object:generate=true
// +groupName=harbor.harbor-operator.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type harborSchemeBuilder struct {
	GroupVersion schema.GroupVersion
	runtime.SchemeBuilder
}

func (b *harborSchemeBuilder) Register(objects ...runtime.Object) {
	b.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(b.GroupVersion, objects...)
		metav1.AddToGroupVersion(scheme, b.GroupVersion)
		return nil
	})
}

var (
	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: "harbor.harbor-operator.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &harborSchemeBuilder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
