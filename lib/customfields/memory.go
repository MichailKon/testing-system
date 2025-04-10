package customfields

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
)

type MemoryLimit uint64

func (m *MemoryLimit) Val() uint64 {
	return uint64(*m)
}

func (m *MemoryLimit) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

func (m *MemoryLimit) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m *MemoryLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *MemoryLimit) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m *MemoryLimit) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("MemoryLimit must be int64")
	}
	*m = MemoryLimit(val)
	return nil
}

func (m *MemoryLimit) Value() (driver.Value, error) {
	return int64(*m), nil
}

func (m *MemoryLimit) GormDataType() string {
	return "int64" // uint64 not supported by goorm
}

func (m *MemoryLimit) FromStr(s string) error {
	num, suf, err := separateStr(s)
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
