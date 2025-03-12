package integrationtest_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	it "github.com/elisasre/go-common/v2/integrationtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNewIntegrationTestRunner(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/target")) })

	const (
		listenAddr = ":8080"
		testAddr   = "http://127.0.0.1" + listenAddr
	)
	c := &it.Compose{}
	itr := it.NewIntegrationTestRunner(
		it.OptBase("./testdata"),
		it.OptTarget("./main.go"),
		it.OptBuildArgs("-ldflags", "-s -w"),
		it.OptBuildEnv("VERSION=dev"),
		it.OptRunArgs("-some=flag"),
		it.OptRunEnv("ITR_TEST_ADDR="+listenAddr),
		it.OptCoverDir(it.IntegrationTestCoverDir),
		it.OptOutput("../../target/testbin/main"),
		it.OptCompose("docker-compose.yaml",
			it.ComposeComposer(c),
			it.ComposeUpOptions(tc.Wait(true)),
			it.ComposeDownOptions(tc.RemoveOrphans(true)),
			it.ComposeOsEnv(),
			it.ComposeEnv(nil),
			it.ComposeWaitForService("echo", wait.ForHTTP("/")),
		),
		it.OptWaitHTTPReady(testAddr+"/ready", time.Second*10),
		it.OptTestFunc(t, testRemote(testAddr+"/remote")),
		it.OptPreHandlerFn(nil, nil),
	)
	require.NoError(t, itr.InitAndRun())
}

func TestOptBinHandler(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/target")) })

	bh := it.NewBinHandler(
		it.BinOptBase("./testdata"),
		it.BinOptTarget("./main.go"),
		it.BinOptOutput("../../target/testbin/main"),
	)
	require.NoError(t, bh.Init())
	require.NoError(t, bh.Build())

	itr := it.NewIntegrationTestRunner(
		it.OptBase("./testdata"),
		it.OptBinHandler(bh),
		it.OptRunStdout(os.Stdout),
		it.OptRunStderr(os.Stderr),
		it.OptRunInheritEnv(true),
		it.OptRunEnv("ITR_TEST_ADDR=:8180"),
		it.OptWaitHTTPReady("http://127.0.0.1:8180/ready", time.Second*10),
		it.OptCompose("docker-compose.yaml",
			it.ComposeUpOptions(tc.Wait(true)),
			it.ComposeDownOptions(tc.RemoveOrphans(true)),
			it.ComposeOsEnv(),
			it.ComposeEnv(nil),
			it.ComposeWaitForService("echo", wait.ForHTTP("/")),
		),
		it.OptTestFunc(t, func(t *testing.T) {
			for i := range 2 {
				addr := fmt.Sprintf(":%d", 8180+i+1)
				testAddr := "http://127.0.0.1" + addr
				newBH := bh.Copy()
				newBH.AddOpts(
					it.BinOptSetRunEnv("ITR_TEST_ADDR="+addr),
					it.BinOptSetRunArgs(),
				)
				require.NoError(t, newBH.Init())
				t.Run(fmt.Sprintf("parallel_env_%d", i), func(t *testing.T) {
					itr := it.NewIntegrationTestRunner(
						it.OptBase("./testdata"),
						it.OptBinHandler(newBH),
						it.OptWaitHTTPReady(testAddr+"/ready", time.Second*10),
						it.OptTestFunc(t, testRemote(testAddr+"/remote")),
					)
					require.NoError(t, itr.InitAndRun())
				})
			}
		}),
	)
	require.NoError(t, itr.InitAndRun())
}

func TestOptMain(t *testing.T) {
	require.NoError(t, it.OptTestMain(&testing.M{})(&it.IntegrationTestRunner{}))
}

func testRemote(addr string) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(addr) //nolint:gosec
		require.NoError(t, err)

		data, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		const expectedResponse = "hello world\n"
		require.Equal(t, expectedResponse, string(data))
	}
}
