package model

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestResourceSerialization(t *testing.T) {
	message := "Resource is active"
	executorEndpoint := "http://localhost:8080"
	resource := Resource{
		Metadata: Metadata{
			NamespacedName: NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			},
			UUID: "1234-5678-91011",
		},
		State:            Running,
		Kind:             "Pod",
		Spec:             map[string]interface{}{"cpu": "2", "memory": "4Gi"},
		Version:          1,
		Message:          &message,
		LastUpdateTime:   time.Now().Unix(),
		ExecutorEndpoint: &executorEndpoint,
	}

	// Serialize the resource
	data, err := json.Marshal(resource)
	assert.NoError(t, err, "Should not have an error while marshaling")

	// Deserialize the data into a new Resource struct
	var newResource Resource
	err = json.Unmarshal(data, &newResource)
	assert.NoError(t, err, "Should not have an error while unmarshaling")

	// Check if the deserialized data matches the original
	assert.Equal(t, resource, newResource, "The original and deserialized resources should match")
}

// TestOptionalFields checks the handling of optional fields during serialization and deserialization.
func TestOptionalFields(t *testing.T) {
	// Resource with nil optional fields
	resource := Resource{
		Metadata: Metadata{
			NamespacedName: NamespacedName{
				Name:      "test-resource",
				Namespace: "default",
			},
			UUID: "1234-5678-91011",
		},
		State:          Running,
		Kind:           "Pod",
		Spec:           map[string]interface{}{"cpu": "2", "memory": "4Gi"},
		Version:        1,
		LastUpdateTime: time.Now().Unix(),
	}

	// Serialize the resource
	data, err := json.Marshal(resource)
	assert.NoError(t, err, "Should not have an error while marshaling with nil optional fields")

	// Deserialize the data into a new Resource struct
	var newResource Resource
	err = json.Unmarshal(data, &newResource)
	assert.NoError(t, err, "Should not have an error while unmarshaling with nil optional fields")

	// Check if the deserialized data matches the original
	assert.Nil(t, newResource.Message, "Message should be nil")
	assert.Nil(t, newResource.ExecutorEndpoint, "ExecutorEndpoint should be nil")
}
