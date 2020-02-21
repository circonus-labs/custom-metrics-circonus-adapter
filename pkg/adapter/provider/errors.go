// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package provider

import (
	"fmt"
	"net/http"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewNoSuchMetricError returns a StatusError indicating that the given metric could not be found.
// It is similar to NewNotFound, but more specialized.
func NewNoSuchMetricError(metricName string, err error) *apierr.StatusError {
	return newMetricNotFoundWithMessageError(fmt.Sprintf("the server could not find the descriptor for metric %s: %s", metricName, err))
}

// NewMetricNotFoundError returns a StatusError indicating that the given metric could not be found.
// It is similar to NewNotFound, but more specialized.
func NewMetricNotFoundError(resource schema.GroupResource, metricName string) *apierr.StatusError {
	return newMetricNotFoundWithMessageError(fmt.Sprintf("the server could not find the metric %s for %s", metricName, resource.String()))
}

// NewMetricNotFoundForError returns a StatusError indicating that the given metric could not be
// found for the given named object. It is similar to NewNotFound, but more specialized.
func NewMetricNotFoundForError(resource schema.GroupResource, metricName string, resourceName string) *apierr.StatusError {
	return newMetricNotFoundWithMessageError(fmt.Sprintf("the server could not find the metric %s for %s %s", metricName, resource.String(), resourceName))
}

// NewExternalMetricNotFoundError returns a status error indicating that the given metric could
// not be found. It is similar to NewNotFound, but more specialized.
func NewExternalMetricNotFoundError(metricName string) *apierr.StatusError {
	return newMetricNotFoundWithMessageError(fmt.Sprintf("the server could not find the metric %s for provided labels", metricName))
}

// NewLabelNotAllowedError returns a status error indicating that the given label is forbidden.
func NewLabelNotAllowedError(label string) *apierr.StatusError {
	return apierr.NewBadRequest(fmt.Sprintf("Metric label: %q is not allowed", label))
}

// NewOperationNotSupportedError returns a StatusError indicating that the invoked API call is not
// supported.
func NewOperationNotSupportedError(operation string) *apierr.StatusError {
	return &apierr.StatusError{
		ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    int32(http.StatusNotImplemented),
			Reason:  metav1.StatusReasonBadRequest,
			Message: fmt.Sprintf("Operation: %q is not implemented", operation),
		},
	}
}

func newMetricNotFoundWithMessageError(message string) *apierr.StatusError {
	return &apierr.StatusError{
		ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Code:    int32(http.StatusNotFound),
			Reason:  metav1.StatusReasonNotFound,
			Message: message,
		},
	}
}
