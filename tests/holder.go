package tests

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"github.com/xorcare/pointer"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"testing_system/common"
	"testing_system/common/config"
	"testing_system/common/db/models"
	"testing_system/invoker"
	"testing_system/lib/logger"
	"testing_system/master"
	"testing_system/storage"
)

const defaultLogLevel = logger.LogLevelInfo

type TSHolder struct {
	ts *common.TestingSystem
	t  *testing.T

	dir        string
	storageDir string
	submitsDir string

	client *resty.Client

	finishWait sync.WaitGroup

	submits []*submitTest
}

func initTS(t *testing.T, sandbox string) *TSHolder {
	h := &TSHolder{
		t:   t,
		dir: t.TempDir(),
	}
	h.storageDir = filepath.Join(h.dir, "storage")
	require.NoError(t, os.CopyFS(h.storageDir, os.DirFS("testdata/storage")))

	h.submitsDir = filepath.Join(h.dir, "submits")
	require.NoError(t, os.CopyFS(h.submitsDir, os.DirFS("testdata/submits")))

	configDir := filepath.Join(h.dir, "configs")
	require.NoError(t, os.CopyFS(configDir, os.DirFS("testdata/configs")))

	configPath := filepath.Join(configDir, "config.yaml")
	h.initTSConfig(configPath, sandbox)

	h.ts = common.InitTestingSystem(configPath)

	h.client = resty.New().SetBaseURL("http://localhost:" + strconv.Itoa(h.ts.Config.Port))

	h.addProblems()

	require.NoError(t, invoker.SetupInvoker(h.ts))
	require.NoError(t, master.SetupMaster(h.ts))
	require.NoError(t, storage.SetupStorage(h.ts))

	h.finishWait.Add(1)

	return h
}

func (h *TSHolder) initTSConfig(configPath string, sandbox string) {
	configContent, err := os.ReadFile(configPath)
	require.NoError(h.t, err)
	cfg := new(config.Config)
	require.NoError(h.t, yaml.Unmarshal(configContent, cfg))
	cfg.Storage.StoragePath = h.storageDir

	cfg.Invoker.SandboxHomePath = filepath.Join(h.dir, "sandbox")
	require.NoError(h.t, os.MkdirAll(cfg.Invoker.SandboxHomePath, 0755))

	cfg.Invoker.CachePath = filepath.Join(h.dir, "invoker_cache")
	require.NoError(h.t, os.MkdirAll(cfg.Invoker.CachePath, 0755))

	cfg.Invoker.CompilerConfigsFolder = filepath.Join(filepath.Dir(configPath), "compiler")

	cfg.Invoker.SandboxType = sandbox
	cfg.LogLevel = pointer.Int(defaultLogLevel)

	configContent, err = yaml.Marshal(cfg)
	require.NoError(h.t, err)
	require.NoError(h.t, os.WriteFile(configPath, configContent, 0644))
}

func (h *TSHolder) addProblems() {
	h.addProblem(1)
	// TODO: Add more
}

func (h *TSHolder) addProblem(id uint) {
	probPath := filepath.Join(h.storageDir, "Problem", strconv.FormatUint(uint64(id), 10))

	probContent, err := os.ReadFile(filepath.Join(probPath, "problem.yaml"))
	require.NoError(h.t, err)

	prob := new(models.Problem)
	require.NoError(h.t, yaml.Unmarshal(probContent, prob))

	prob.ID = id
	require.NoError(h.t, h.ts.DB.Save(prob).Error)

	testlib, err := os.ReadFile(filepath.Join(h.storageDir, "testlib.h"))
	require.NoError(h.t, err)
	require.NoError(h.t, os.WriteFile(filepath.Join(probPath, "sources", "testlib.h"), testlib, 0777))

	cmd := exec.Command("g++", "check.cpp", "-std=c++17", "-o", "../checker/check")
	cmd.Dir = filepath.Join(probPath, "sources")
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	require.NoError(h.t, err)
}

func (h *TSHolder) stop() {
	logger.Warn("Stopping TS because testing is complete")
	h.ts.Stop()
	h.finishWait.Wait()
}

func (h *TSHolder) start() {
	h.ts.Run()
	h.finishWait.Done()
}
