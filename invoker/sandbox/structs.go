package sandbox

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing_system/common/constants/verdict"
	"unicode"
)

type RunConfig struct {
	// TL is set with TLstr by number and size suffix. Possible suffixes are:
	// * s: means seconds
	// * ms: means milliseconds
	// * us: means microseconds (not recommended)
	// * ns: means nanoseconds (definitely not recommended)
	// Suffix can be in uppercase or lowercase.
	// E.g. "10s" means 10 seconds (the value of TL will be 10^10), "5ms" means 5 milliseconds (the value of TL will be 5 * 10^6)
	TL    uint64 `yaml:"-" json:"-"`
	TLstr string `yaml:"TL" json:"TL"`

	// ML is set with MLstr by number and size suffix. Possible suffixes are:
	// * B: means bytes
	// * K: means kibibytes or 2^10
	// * M: means mebibytes or 2^20
	// * G: means gigibytes or 2^30
	// Suffix can be in uppercase or lowercase.
	// E.g. "1G" means 1 gigibyte (the value of TL will be  2^30), "128M" means 128 mebibytes (the value of ML will be 128 * 2^20)
	ML    uint64 `yaml:"-" json:"-"`
	MLstr string `yaml:"ML" json:"ML"`

	// WL means wall time limit, it is set up same way as TL
	WL    uint64 `yaml:"-" json:"-"`
	WLstr string `yaml:"WL" json:"WL"`
}

// FillIn parses string written values into numeric
func (r *RunConfig) FillIn() error {
	if err := parseTime(r.TLstr, &r.TL); err != nil {
		return fmt.Errorf("can not parse TL %s", err.Error())
	}
	if err := parseSize(r.MLstr, &r.ML); err != nil {
		return fmt.Errorf("can not parse ML %s", err.Error())
	}
	if err := parseTime(r.WLstr, &r.WL); err != nil {
		return fmt.Errorf("can not parse WL %s", err.Error())
	}
	return nil
}

func parseTime(str string, val *uint64) error {
	numStr, suf := sepStr(str)
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return err
	}
	switch suf {
	case "", "ns":
		break
	case "us":
		num *= 1000
		fallthrough
	case "ms":
		num *= 1000
		fallthrough
	case "s":
		num *= 1000
	default:
		return fmt.Errorf("unknown time suffix %s", suf)
	}
	*val = num
	return nil
}

func parseSize(str string, val *uint64) error {
	numStr, suf := sepStr(str)
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return err
	}
	switch suf {
	case "", "b":
		break
	case "k":
		num *= 1024
		fallthrough
	case "m":
		num *= 1024
		fallthrough
	case "g":
		num *= 1024
	default:
		return fmt.Errorf("unknown size suffix %s", suf)
	}
	*val = num
	return nil
}

func sepStr(str string) (string, string) {
	str = strings.ToLower(str)
	pos := 0
	for ; pos < len(str); pos++ {
		if !unicode.IsDigit(rune(str[pos])) {
			break
		}
	}
	return str[:pos], str[pos:]
}

type RunResult struct {
	Err error

	Verdict verdict.Verdict

	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}
