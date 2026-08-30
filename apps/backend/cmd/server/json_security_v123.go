package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// decodeStrictJSON decodes exactly one JSON object/value from the request body.
// Unknown object fields and trailing JSON/garbage are rejected so security-
// sensitive handlers do not silently accept ambiguous payloads.
func decodeStrictJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}
