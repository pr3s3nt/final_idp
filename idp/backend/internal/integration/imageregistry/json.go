package imageregistry

import (
	"encoding/json"
	"net/http"
)

func decodeJSON(resp *http.Response, v any) error {
	return json.NewDecoder(resp.Body).Decode(v)
}
