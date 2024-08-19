package integrationtest_test

import (
	"context"

	it "github.com/elisasre/go-common/v2/integrationtest"
)

func ExampleComposeComposer() {
	c := &it.Compose{}

	itr := it.NewIntegrationTestRunner(
		it.OptCompose("docker-compose.yaml", it.ComposeComposer(c)),
	)

	if err := itr.Init(); err != nil {
		return
	}

	dbContainer, err := c.ServiceContainer(context.Background(), "postgres")
	if err != nil {
		return
	}

	_, _, err = dbContainer.Exec(
		context.Background(),
		[]string{"psql", "-U", "demo", "-d", "demo", "-c", "'SELECT * FROM demo_table'"},
	)
	if err != nil {
		return
	}
}
