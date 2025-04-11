package customfields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

// Time is set by number and size suffix. Possible suffixes are:
// * s: means seconds
// * ms: means milliseconds
// * us: means microseconds (not recommended)
// * ns: means nanoseconds (definitely not recommended)
// Suffix can be in uppercase or lowercase.
// E.g. "10s" means 10 seconds (the value will be 10^10), "5ms" means 5 milliseconds (the value will be 5 * 10^6)

type Time uint64

func (t *Time) Val() uint64 {
	return uint64(*t)
}

func (t Time) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}

func (t *Time) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	return t.FromStr(s)
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return t.FromStr(s)
}

func (t *Time) Scan(value interface{}) error {
	val, ok := value.(int64)
	if !ok {
		return fmt.Errorf("Time must be a int64")
	}
	*t = Time(val)
	return nil
}

func (t *Time) Value() (driver.Value, error) {
	return int64(*t), nil
}

func (t *Time) GormDataType() string {
	return "int64" // uint64 not supported by goorm
}

func (t *Time) FromStr(s string) error {
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
	*t = Time(num)
	return nil
}

func (t *Time) String() string {
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
