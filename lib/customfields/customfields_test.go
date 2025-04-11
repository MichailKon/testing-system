package customfields

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestTime(t *testing.T) {
	var resTime Time
	var unmarshalledTime Time
	t.Run("no error + serialization", func(t *testing.T) {
		var timetests = []struct {
			in       string
			out      Time
			jsonTime string
			yamlTime string
			strTime  string
		}{
			{"1", 1, `"1ns"`, "1ns\n", "1ns"},
			{"1ns", 1, `"1ns"`, "1ns\n", "1ns"},
			{"5ns", 5, `"5ns"`, "5ns\n", "5ns"},
			{"1us", 1_000, `"1us"`, "1us\n", "1us"},
			{"5us", 5_000, `"5us"`, "5us\n", "5us"},
			{"1ms", 1_000_000, `"1ms"`, "1ms\n", "1ms"},
			{"5ms", 5_000_000, `"5ms"`, "5ms\n", "5ms"},
			{"1s", 1_000_000_000, `"1s"`, "1s\n", "1s"},
			{"5s", 5_000_000_000, `"5s"`, "5s\n", "5s"},
		}
		for _, tt := range timetests {
			t.Run(tt.in, func(t *testing.T) {
				require.Nil(t, resTime.FromStr(tt.in))
				require.Equal(t, tt.out, resTime)
				// json
				jsonT, err := json.Marshal(resTime)
				require.Nil(t, err)
				require.Equal(t, tt.jsonTime, string(jsonT))
				require.Nil(t, json.Unmarshal(jsonT, &unmarshalledTime))
				require.Equal(t, tt.out, unmarshalledTime)
				// yaml
				yamlT, err := yaml.Marshal(resTime)
				require.Nil(t, err)
				require.Equal(t, tt.yamlTime, string(yamlT))
				require.Nil(t, yaml.Unmarshal(yamlT, &unmarshalledTime))
				require.Equal(t, tt.out, unmarshalledTime)
				// toString
				require.Equal(t, tt.strTime, unmarshalledTime.String())
			})
		}
	})
	t.Run("error", func(t *testing.T) {
		var timetests = []struct {
			in string
		}{
			{"5ts"},
			{"aboba"},
			{"5.5s"},
			{"5.5"},
		}
		for _, tt := range timetests {
			t.Run(tt.in, func(t *testing.T) {
				require.NotNil(t, resTime.FromStr(tt.in))
			})
		}
	})
}

func TestMemory(t *testing.T) {
	var resMemory Memory
	var unmarshalledMemory Memory
	t.Run("no error + serialization", func(t *testing.T) {
		var memorytests = []struct {
			in       string
			out      Memory
			jsonTime string
			yamlTime string
			strTime  string
		}{
			{"1", 1, `"1b"`, "1b\n", "1b"},
			{"1b", 1, `"1b"`, "1b\n", "1b"},
			{"5b", 5, `"5b"`, "5b\n", "5b"},
			{"1k", 1 << 10, `"1k"`, "1k\n", "1k"},
			{"5k", 5 * (1 << 10), `"5k"`, "5k\n", "5k"},
			{"1m", 1 << 20, `"1m"`, "1m\n", "1m"},
			{"5m", 5 * (1 << 20), `"5m"`, "5m\n", "5m"},
			{"1g", 1 << 30, `"1g"`, "1g\n", "1g"},
			{"5g", 5 * (1 << 30), `"5g"`, "5g\n", "5g"},
		}
		for _, tt := range memorytests {
			t.Run(tt.in, func(t *testing.T) {
				require.Nil(t, resMemory.FromStr(tt.in))
				require.Equal(t, tt.out, resMemory)
				// json
				jsonT, err := json.Marshal(resMemory)
				require.Nil(t, err)
				require.Equal(t, tt.jsonTime, string(jsonT))
				require.Nil(t, json.Unmarshal(jsonT, &unmarshalledMemory))
				require.Equal(t, tt.out, unmarshalledMemory)
				// yaml
				yamlT, err := yaml.Marshal(resMemory)
				require.Nil(t, err)
				require.Equal(t, tt.yamlTime, string(yamlT))
				require.Nil(t, yaml.Unmarshal(yamlT, &unmarshalledMemory))
				require.Equal(t, tt.out, unmarshalledMemory)
				// toString
				require.Equal(t, tt.strTime, unmarshalledMemory.String())
			})
		}
	})
	t.Run("error", func(t *testing.T) {
		var memorytests = []struct {
			in string
		}{
			{"5t"},
			{"aboba"},
			{"5.5k"},
			{"5.5"},
		}
		for _, tt := range memorytests {
			t.Run(tt.in, func(t *testing.T) {
				require.NotNil(t, resMemory.FromStr(tt.in))
			})
		}
	})
}
