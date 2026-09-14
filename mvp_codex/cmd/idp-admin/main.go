package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/thanhnt1/final-idp/mvp-codex/internal/domain"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/fixture"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/platform"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/store/postgres"
)

func main() {
	command := flag.String("command", "plan", "plan, migrate, or seed")
	fixturePath := flag.String("fixture", "fixtures/uc3-demo.json", "path to the UC3 fixture")
	stateRoot := flag.String("state-root", ".state/resources", "durable Terraform resource state root")
	databaseURL := flag.String("database-url", os.Getenv("DATABASE_URL"), "metadata PostgreSQL URL")
	flag.Parse()
	input, err := fixture.Load(*fixturePath)
	fatalIf(err)
	if *command != "plan" {
		runDatabaseCommand(*command, *databaseURL, input)
		return
	}
	printPlan(input, *stateRoot)
}

func printPlan(input fixture.Data, stateRoot string) {
	instances := make(map[string]domain.ResourceInstance, len(input.Instances))
	for _, instance := range input.Instances {
		instances[instance.ResourceRequirementID] = instance
	}
	prepared, err := platform.NewEngine(stateRoot).Prepare(input.Snapshot, input.Definitions, instances)
	fatalIf(err)
	out, err := json.MarshalIndent(prepared, "", "  ")
	fatalIf(err)
	fmt.Println(string(out))
}

func runDatabaseCommand(command, databaseURL string, input fixture.Data) {
	if databaseURL == "" {
		fatalIf(fmt.Errorf("database-url or DATABASE_URL is required for %s", command))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := postgres.Open(ctx, databaseURL)
	fatalIf(err)
	defer store.Close()
	switch command {
	case "migrate":
		fatalIf(store.Migrate(ctx))
		fmt.Println("metadata migrations applied")
	case "seed":
		fatalIf(store.Migrate(ctx))
		fatalIf(store.SeedFixture(ctx, input))
		fmt.Println("UC3 fixture seeded")
	default:
		fatalIf(fmt.Errorf("unknown command %q", command))
	}
}

func fatalIf(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
