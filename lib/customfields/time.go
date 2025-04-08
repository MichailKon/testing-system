package customfields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

// TimeLimit is set by number and size suffix. Possible suffixes are:
// * s: means seconds
// * ms: means milliseconds
// * us: means microseconds (not recommended)
// * ns: means nanoseconds (definitely not recommended)
// Suffix can be in uppercase or lowercase.
// E.g. "10s" means 10 seconds (the value will be 10^10), "5ms" means 5 milliseconds (the value will be 5 * 10^6)

type TimeLimit uint64

func (t *TimeLimit) Val() uint64 {
	return uint64(*t)
}

func (t *TimeLimit) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *TimeLimit) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return t.FromStr(s)
}

func (t *TimeLimit) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *TimeLimit) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return t.FromStr(s)
}

func (t *TimeLimit) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return fmt.Errorf("TimeLimit must be a int64")
	}
	*t = TimeLimit(val)
	return nil
}

func (t *TimeLimit) Value() (driver.Value, error) {
	return int64(*t), nil
}

func (t *TimeLimit) GormDataType() string {
	return "int64" // uint64 not supported by goorm
}

func (t *TimeLimit) FromStr(s string) error {
	num, suf, err := separateStr(s)
	if err != nil {
		return err
	}
	switch suf {
	case "", "ns":
		break
	case "s":
		num *= 1000
		fallthrough
	case "ms":
		num *= 1000
		fallthrough
	case "us":
		num *= 1000
	default:
		return fmt.Errorf("unknown time suffix %s", suf)
	}
	*t = TimeLimit(num)
	return nil
}

func (t *TimeLimit) String() string {
	if t == nil {
		return "<nil>"
	}
	v := t.Val()
	suf := "ns"
	if v%1000 == 0 {
		suf = "us"
		v /= 1000
		if v%1000 == 0 {
			suf = "ms"
			v /= 1000
			if v%1000 == 0 {
				suf = "s"
				v /= 1000
			}
		}
	}
	return fmt.Sprintf("%d%s", v, suf)
}
