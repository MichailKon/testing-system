package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing_system/common/config"
	db2 "testing_system/common/db"
)

var polygonApiKey string
var polygonApiSecret string

var archive *zip.ReadCloser

var probTmpPath string

var probXML XProblemXML

func main() {
	configPath := os.Args[1]
	cfg := config.ReadConfig(configPath)
	db, err := db2.NewDB(cfg.DB)
	if err != nil {
		panic(err)
	}
	polygonApiKey = os.Args[3]
	polygonApiSecret = os.Args[4]
	probID, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}

	workdir, err := os.MkdirTemp("", "")
	if err != nil {
		panic(err)
	}
	packagePath := filepath.Join(workdir, "package.zip")
	if err = ImportPackageApi(probID, packagePath); err != nil {
		panic(err)
	}

	archive, err = zip.OpenReader(packagePath)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	probTmpPath = filepath.Join(workdir, "problem")
	if err = os.MkdirAll(probTmpPath, 0755); err != nil {
		panic(err)
	}

	if err = extractAllFiles("tests", "tests"); err != nil {
		panic(err)
	}

	if err = extractAllFiles("files", "sources"); err != nil {
		panic(err)
	}

	if err = moveFile("problem.xml", "", "problem.xml"); err != nil {
		panic(err)
	}

	probXMLData, err := os.ReadFile(filepath.Join(probTmpPath, "problem.xml"))
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal(probXMLData, &probXML)
	if err != nil {
		panic(err)
	}

	checkerCPP := filepath.Base(probXML.Assets.Checker.Source.Path)
	cmd := exec.Command("g++", checkerCPP, "-std=c++20", "-o", "checker")
	cmd.Dir = filepath.Join(probTmpPath, "sources")
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	if err = os.MkdirAll(filepath.Join(probTmpPath, "checker"), 0755); err != nil {
		panic(err)
	}

	err = exec.Command(
		"mv",
		filepath.Join(probTmpPath, "sources", "checker"),
		filepath.Join(probTmpPath, "checker", "checker"),
	).Run() // If go has no single line solution for this, I will use cmd!
	if err != nil {
		panic(err)
	}

	prob := buildProblemModel()

	err = db.Create(prob).Error
	if err != nil {
		panic(err)
	}

	out, err := exec.Command("mv", probTmpPath, filepath.Join(
		cfg.Storage.StoragePath,
		"Problem",
		strconv.FormatUint(uint64(prob.ID), 10),
	)).CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}
	fmt.Printf("Created problem %d", prob.ID)
}
