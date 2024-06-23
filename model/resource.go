package model

import (
	"encoding/json"
	"time"
)

type State string

const (
	Free       = ""
	Assigning  = "assigning"
	Initiating = "initiating"
	Running    = "running"
	Deleting   = "deleting"
	Deleted    = "deleted"
	Failed     = "failed"
)

type Timestamp time.Time

// MarshalJSON for Timestamp to serialize it as a Unix timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	ts := time.Time(t).Unix()
	return json.Marshal(ts)
}

// UnmarshalJSON for Timestamp to deserialize a Unix timestamp into the Timestamp type.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var ts int64
	if err := json.Unmarshal(data, &ts); err != nil {
		return err
	}
	*t = Timestamp(time.Unix(ts, 0))
	return nil
}

func (t Timestamp) Equal(u Timestamp) bool {
	return time.Time(t).Equal(time.Time(u))
}

type NamespacedName struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Metadata struct {
	NamespacedName
	UUID string `json:"uuid"`
}

type Resource struct {
	Metadata Metadata               `json:"metadata"`
	Kind     string                 `json:"kind"`
	Spec     map[string]interface{} `json:"spec"`

	State            State   `json:"state"`
	Version          int64   `json:"version"`
	Message          *string `json:"message,omitempty"`
	LastUpdateTime   int64   `json:"lastUpdateTime"`
	ExecutorEndpoint *string `json:"executorEndpoint,omitempty"`
}
