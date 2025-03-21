package http

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
)

var host string

func init() {
	flag.StringVar(&host, "host", "localhost", "Host to connect to")
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func Test_EndToEnd(t *testing.T) {
	if _, ok := os.LookupEnv("CIRCLECI"); !ok {
		t.Skip("Skipping end-to-end test")
	}
	ctx := context.Background()

	client := NewClient(fmt.Sprintf("http://%s:4001", host), nil)
	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)"); err != nil {
		t.Fatalf("Error creating table: %s", err)
	}
	defer client.Close()

	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)"); err != nil {
		t.Fatalf("Unexpected error creating an already created table: %s", err)
	}
	client.PromoteErrors(true)
	if _, err := client.ExecuteSingle(ctx, "CREATE TABLE foo (id INT, name TEXT)"); err == nil {
		t.Fatalf("Expected error creating table duplicate table")
	}

	stmt, err := NewSQLStatement("INSERT INTO foo(name) VALUES(?)", "fiona")
	if err != nil {
		t.Fatalf("Error creating statement: %s", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := client.Execute(ctx, SQLStatements{stmt}, nil); err != nil {
			t.Fatalf("Error inserting record %d: %s", i, err)
		}
	}

	stmt, err = NewSQLStatement("SELECT COUNT(*) FROM foo")
	if err != nil {
		t.Fatalf("Error creating statement: %s", err)
	}
	resp, err := client.Query(ctx, SQLStatements{stmt}, nil)
	if err != nil {
		t.Fatalf("Error counting records: %s", err)
	}

	results, ok := resp.Results.([]QueryResult)
	if !ok {
		t.Fatalf("Unexpected result type")
	}
	if len(results) != 1 {
		t.Fatalf("Unexpected number of results")
	}
	if len(results[0].Values) != 1 {
		t.Fatalf("Unexpected number of rows")
	}
	v, ok := results[0].Values[0][0].(float64)
	if !ok {
		t.Fatalf("Unexpected value type: %T", results[0].Values[0][0])
	}
	if v != 10 {
		t.Fatalf("Unexpected value")
	}
}
