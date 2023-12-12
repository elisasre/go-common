package integrationtest_test

import (
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	it "github.com/elisasre/go-common/integrationtest"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestApp(t *testing.T) {
	c := &it.Compose{}
	itr := it.NewIntegrationTestRunner(
		it.OptBase("./testdata"),
		it.OptTarget("./main.go"),
		it.OptBuildArgs("-ldflags", "-s -w"),
		it.OptBuildEnv("VERSION=dev"),
		it.OptRunArgs("-listen=:8080"),
		it.OptRunEnv("SOME_VAR=value"),
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
		it.OptWaitHTTPReady("http://127.0.0.1:8080", time.Second*10),
		it.OptTestFunc(t, testHealthy),
	)
	if err := itr.InitAndRun(); err != nil {
		log.Fatal(err)
	}
}

func TestOptMain(t *testing.T) {
	require.NoError(t, it.OptTestMain(&testing.M{})(&it.IntegrationTestRunner{}))
}

func testHealthy(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:8080")
	require.NoError(t, err)

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	const expectedResponse = "hello world\n"
	require.Equal(t, expectedResponse, string(data))
}
