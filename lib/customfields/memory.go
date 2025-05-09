package customfields

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Memory is set by number and size suffix. Possible suffixes are:
// * g: means gigibytes
// * m: means mebibytes
// * k: means kibibytes
// * b: means bytes
// Suffix can be in uppercase or lowercase.
// E.g. "10g" means 10 gigibyte (the value will be 10 * 2^30), "5ms" means 5 milliseconds (the value will be 5 * 2^20)

type Memory uint64

func (m *Memory) Val() uint64 {
	return uint64(*m)
}

func (m Memory) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

func (m *Memory) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m Memory) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Memory) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return m.FromStr(s)
}

func (m *Memory) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return errors.New("Memory must be int64")
	}
	*m = Memory(val)
	return nil
}

func (m Memory) Value() (driver.Value, error) {
	return int64(m), nil
}

func (m Memory) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "int64"
	case "postgres":
		return "bigint"
	}
	return ""
}

func (m *Memory) FromStr(s string) error {
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
	*m = Memory(num)
	return nil
}

func NewMemory(s string) (*Memory, error) {
	var m Memory
	if err := m.FromStr(s); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m Memory) String() string {
	v := m.Val()
	suf := "b"
	if v%1024 == 0 {
		suf = "k"
		v /= 1024
		if v%1024 == 0 {
			suf = "m"
			v /= 1024
			if v%1024 == 0 {
				suf = "g"
				v /= 1024
			}
		}
	}
	return fmt.Sprintf("%d%s", v, suf)
}
