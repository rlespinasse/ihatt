package forge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Options controls one forge sync.
type Options struct {
	Full     bool // ignore the watermark and fetch everything
	Comments bool // also fetch comment bodies (one extra request per changed item)
}

// DecodePages reads the JSON arrays that paginated CLI output prints back to back
// (gh api --paginate and glab api --paginate) and returns their items as one slice.
func DecodePages[T any](data []byte) ([]T, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var all []T
	for {
		var page []T
		err := dec.Decode(&page)
		if errors.Is(err, io.EOF) {
			return all, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parse page: %w", err)
		}
		all = append(all, page...)
	}
}
