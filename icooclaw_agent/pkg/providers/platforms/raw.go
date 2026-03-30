package platforms

import "encoding/json"

func mustMarshalAdapterRaw(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}
