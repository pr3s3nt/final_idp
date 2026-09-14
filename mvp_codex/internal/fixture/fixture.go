package fixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
)

type Data struct {
	Snapshot    domain.DeploymentInputSnapshot `json:"snapshot"`
	Definitions []domain.ResourceDefinition    `json:"resourceDefinitions"`
	Instances   []domain.ResourceInstance      `json:"resourceInstances,omitempty"`
}

func Load(path string) (Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Data{}, fmt.Errorf("read fixture: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	var data Data
	if err := decoder.Decode(&data); err != nil {
		return Data{}, fmt.Errorf("decode fixture: %w", err)
	}
	return data, nil
}
