package evaluator

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/issueye/icooclaw_lang/internal/object"
)

func TestSQLiteDatabaseLibrary(t *testing.T) {
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "demo.db"))

	env, result := evalSource(t, fmt.Sprintf(`
conn = db.sqlite.open("%s")
driver_name = conn.driver()
ping_ok = conn.ping()

conn.exec("create table users (id integer primary key autoincrement, name text, age integer)")
insert_one = conn.exec("insert into users (name, age) values (?, ?)", ["alice", 20])
insert_two = conn.exec("insert into users (name, age) values (?, ?)", ["bob", 30])
rows = conn.query("select id, name, age from users order by id")
first = conn.query_one("select name, age from users where age > ?", [25])
stats = conn.stats()
conn.close()
`, dbPath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "driver_name"); got != "sqlite" {
		t.Fatalf("expected driver_name=sqlite, got %s", got)
	}
	if got := testStringValue(t, env, "ping_ok"); got != "true" {
		t.Fatalf("expected ping_ok=true, got %s", got)
	}

	insertOne := testHashValue(t, env, "insert_one")
	if insertOne.Pairs["rows_affected"].Value.Inspect() != "1" {
		t.Fatalf("expected insert_one rows_affected=1, got %s", insertOne.Pairs["rows_affected"].Value.Inspect())
	}
	insertTwo := testHashValue(t, env, "insert_two")
	if insertTwo.Pairs["last_insert_id"].Value.Inspect() != "2" {
		t.Fatalf("expected insert_two last_insert_id=2, got %s", insertTwo.Pairs["last_insert_id"].Value.Inspect())
	}

	rows := testArrayValue(t, env, "rows")
	if len(rows.Elements) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows.Elements))
	}
	firstRow, ok := rows.Elements[0].(*object.Hash)
	if !ok {
		t.Fatalf("expected first row hash, got %T", rows.Elements[0])
	}
	if firstRow.Pairs["name"].Value.Inspect() != "alice" || firstRow.Pairs["age"].Value.Inspect() != "20" {
		t.Fatalf("unexpected first row: %s", firstRow.Inspect())
	}

	first := testHashValue(t, env, "first")
	if first.Pairs["name"].Value.Inspect() != "bob" || first.Pairs["age"].Value.Inspect() != "30" {
		t.Fatalf("unexpected query_one row: %s", first.Inspect())
	}

	stats := testHashValue(t, env, "stats")
	if stats.Pairs["driver"].Value.Inspect() != "sqlite" {
		t.Fatalf("expected stats.driver=sqlite, got %s", stats.Pairs["driver"].Value.Inspect())
	}
}

func TestMySQLAndPGDatabaseDriversOpenAndClose(t *testing.T) {
	env, result := evalSource(t, `
mysql_conn = db.mysql.open("user:pass@tcp(127.0.0.1:1)/demo")
mysql_driver = mysql_conn.driver()
mysql_conn.close()

pg_conn = db.pg.open("postgres://user:pass@127.0.0.1:1/demo?sslmode=disable")
pg_driver = pg_conn.driver()
pg_conn.close()
`)

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "mysql_driver"); got != "mysql" {
		t.Fatalf("expected mysql_driver=mysql, got %s", got)
	}
	if got := testStringValue(t, env, "pg_driver"); got != "pg" {
		t.Fatalf("expected pg_driver=pg, got %s", got)
	}
}
