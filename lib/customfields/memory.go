package customfields

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

type MemoryLimit uint64

func (m *MemoryLimit) Val() uint64 {
	return uint64(*m)
}

func (m *MemoryLimit) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("MemoryLimit does not support marshalling")
}

func (m *MemoryLimit) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m *MemoryLimit) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("MemoryLimit does not support marshalling")
}

func (m *MemoryLimit) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m *MemoryLimit) FromStr(s string) error {
	num, suf, err := sepStr(s)
	if err != nil {
		return err
	}
	switch suf {
	case "", "b":
		break
	case "g":
		num *= 1024
		fallthrough
	case "m":
		num *= 1024
		fallthrough
	case "k":
		num *= 1024
	default:
		return fmt.Errorf("unknown size suffix %s", suf)
	}
	*m = MemoryLimit(num)
	return nil
}

func (m *MemoryLimit) String() string {
	if m == nil {
		return "<nil>"
	}
	v := m.Val()
	suf := "b"
	if v%1024 == 0 {
		suf = "k"
		v /= 1024
		if v%1024 != 0 {
			suf = "m"
			v /= 1024
			if v%1024 != 0 {
				suf = "g"
				v /= 1024
			}
		}
	}
	return fmt.Sprintf("%d%s", v, suf)
}
