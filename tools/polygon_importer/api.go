package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

const POLYGON_URL = "https://polygon.codeforces.com/api/"
const PACKAGES_METHOD = "problem.packages"
const PACKAGE_METHOD = "problem.package"

func ImportPackageApi(probIDInt int, outPath string) error {
	probID := strconv.Itoa(probIDInt)
	packageID, err := getPackages(probID)
	if err != nil {
		return fmt.Errorf("%s: %s", PACKAGES_METHOD, err.Error())
	}

	err = downloadPackage(probID, strconv.Itoa(packageID), outPath)
	if err != nil {
		return fmt.Errorf("%s: %s", PACKAGE_METHOD, err.Error())
	}
	fmt.Printf("Downloaded problem %d package\n", probIDInt)
	return nil
}

func downloadPackage(probID string, packageID string, outPath string) error {
	values := url.Values{}
	values.Set("problemId", probID)
	values.Set("packageId", packageID)
	values.Set("type", "linux")
	values = fixApiValues(PACKAGE_METHOD, values)

	client := resty.New()
	r := client.R()
	r.SetQueryParamsFromValues(values)
	r.SetOutput(outPath)
	resp, err := r.Get(POLYGON_URL + PACKAGE_METHOD)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("exit code %d, body: %s", resp.StatusCode(), resp.Body())
	}
	return nil
}

type PPackagesList struct {
	Status  string            `json:"status"`
	Comment string            `json:"comment"`
	Result  []*PSinglePackage `json:"result"`
}

type PSinglePackage struct {
	ID       int    `json:"id"`
	Revision int    `json:"revision"`
	Type     string `json:"type"`
}

func getPackages(probID string) (int, error) {
	values := url.Values{}
	values.Set("problemId", probID)
	values = fixApiValues(PACKAGES_METHOD, values)

	client := resty.New()
	r := client.R()
	r.SetQueryParamsFromValues(values)
	res, err := r.Get(POLYGON_URL + PACKAGES_METHOD)
	if err != nil {
		return 0, err
	}
	if res.StatusCode() != http.StatusOK {
		return 0, fmt.Errorf("exit code %d, body: %s", res.StatusCode(), res.String())
	}

	packages := &PPackagesList{}
	err = json.Unmarshal(res.Body(), packages)
	if err != nil {
		return 0, fmt.Errorf("can not parse json response, error: %s", err.Error())
	}

	if packages.Status != "OK" {
		return 0, fmt.Errorf("responded with %s, comment: %s", packages.Status, packages.Comment)
	}

	var bestPackage *PSinglePackage
	for _, pack := range packages.Result {
		if pack.Type != "linux" {
			continue
		}

		if bestPackage == nil || pack.Revision > bestPackage.Revision {
			bestPackage = pack
		}
	}

	if bestPackage == nil {
		return 0, fmt.Errorf("no package created for problem %s", probID)
	}

	return bestPackage.ID, nil
}

func fixApiValues(method string, values url.Values) url.Values {
	tm := time.Now().Unix()
	values.Set("time", strconv.FormatInt(tm, 10))
	values.Set("apiKey", polygonApiKey)

	builder := bytes.Buffer{}
	rand := "000000"
	builder.WriteString(rand)
	builder.WriteRune('/')
	builder.WriteString(method)
	builder.WriteRune('?')
	builder.WriteString(values.Encode())
	builder.WriteRune('#')
	builder.WriteString(polygonApiSecret)

	endoded := sha512.Sum512(builder.Bytes())

	values.Set("apiSig", rand+hex.EncodeToString(endoded[:]))
	return values
}

const kRetryCount = 10
