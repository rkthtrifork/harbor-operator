package v1alpha1

// ObjectRef is a generic reference to a Kubernetes object in a specific namespace.
type ObjectRef struct {
	// Name of the object.
	Name string `json:"name"`
	// Namespace where the object is located.
	Namespace string `json:"namespace"`
}
