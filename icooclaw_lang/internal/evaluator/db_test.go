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

func TestSQLiteTransactionsCommitAndRollback(t *testing.T) {
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "tx.db"))

	env, result := evalSource(t, fmt.Sprintf(`
conn = db.sqlite.open("%s")
conn.exec("create table ledger (id integer primary key autoincrement, name text, amount integer)")

tx1 = conn.begin()
tx1.exec("insert into ledger (name, amount) values (?, ?)", ["income", 100])
tx1_status_before = tx1.is_closed()
tx1.commit()
tx1_status_after = tx1.is_closed()

tx2 = conn.begin()
tx2.exec("insert into ledger (name, amount) values (?, ?)", ["expense", 30])
tx2.rollback()

rows = conn.query("select name, amount from ledger order by id")
row = conn.query_one("select name, amount from ledger order by id limit 1")
conn.close()
`, dbPath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "tx1_status_before"); got != "false" {
		t.Fatalf("expected tx1_status_before=false, got %s", got)
	}
	if got := testStringValue(t, env, "tx1_status_after"); got != "true" {
		t.Fatalf("expected tx1_status_after=true, got %s", got)
	}

	rows := testArrayValue(t, env, "rows")
	if len(rows.Elements) != 1 {
		t.Fatalf("expected 1 committed row, got %d", len(rows.Elements))
	}
	row := testHashValue(t, env, "row")
	if row.Pairs["name"].Value.Inspect() != "income" || row.Pairs["amount"].Value.Inspect() != "100" {
		t.Fatalf("unexpected committed row: %s", row.Inspect())
	}
}

func TestSQLitePreparedStatements(t *testing.T) {
	dbPath := filepath.ToSlash(filepath.Join(t.TempDir(), "stmt.db"))

	env, result := evalSource(t, fmt.Sprintf(`
conn = db.sqlite.open("%s")
conn.exec("create table users (id integer primary key autoincrement, name text, age integer)")

insert_stmt = conn.prepare("insert into users (name, age) values (?, ?)")
insert_stmt_sql = insert_stmt.sql()
insert_stmt.exec(["alice", 20])
insert_stmt.exec(["bob", 30])
insert_stmt.close()
insert_stmt_closed = insert_stmt.is_closed()

select_stmt = conn.prepare("select name, age from users where age >= ? order by age")
rows = select_stmt.query([20])
row = select_stmt.query_one([30])

tx = conn.begin()
tx_stmt = tx.prepare("insert into users (name, age) values (?, ?)")
tx_stmt.exec(["carol", 40])
tx_stmt.close()
tx.commit()

final_rows = conn.query("select name from users order by id")
select_stmt.close()
conn.close()
`, dbPath))

	if object.IsError(result) {
		t.Fatalf("unexpected eval error: %s", result.Inspect())
	}

	if got := testStringValue(t, env, "insert_stmt_sql"); got != "insert into users (name, age) values (?, ?)" {
		t.Fatalf("unexpected insert_stmt_sql: %s", got)
	}
	if got := testStringValue(t, env, "insert_stmt_closed"); got != "true" {
		t.Fatalf("expected insert_stmt_closed=true, got %s", got)
	}

	rows := testArrayValue(t, env, "rows")
	if len(rows.Elements) != 2 {
		t.Fatalf("expected 2 selected rows, got %d", len(rows.Elements))
	}

	row := testHashValue(t, env, "row")
	if row.Pairs["name"].Value.Inspect() != "bob" || row.Pairs["age"].Value.Inspect() != "30" {
		t.Fatalf("unexpected prepared query_one row: %s", row.Inspect())
	}

	finalRows := testArrayValue(t, env, "final_rows")
	if len(finalRows.Elements) != 3 {
		t.Fatalf("expected 3 final rows, got %d", len(finalRows.Elements))
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
