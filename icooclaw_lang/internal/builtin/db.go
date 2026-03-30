package builtin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/issueye/icooclaw_lang/internal/object"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type nativeDBConn struct {
	driver string
	dsn    string
	db     *sql.DB
}

type nativeDBTx struct {
	driver string
	tx     *sql.Tx
	closed bool
}

type dbQueryExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

func newDBLib() *object.Hash {
	return hashObject(map[string]object.Object{
		"sqlite": newDBDriverLib("sqlite", "sqlite"),
		"mysql":  newDBDriverLib("mysql", "mysql"),
		"pg":     newDBDriverLib("pg", "pgx"),
	})
}

func newDBDriverLib(name, driver string) *object.Hash {
	return hashObject(map[string]object.Object{
		"open": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 1 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=1", len(args))
			}
			dsn, errObj := stringArg(args[0], "argument to `open` must be STRING, got %s")
			if errObj != nil {
				return errObj
			}

			db, err := sql.Open(driver, dsn)
			if err != nil {
				return object.NewError(0, "could not open %s database: %s", name, err.Error())
			}
			db.SetConnMaxIdleTime(30 * time.Second)
			db.SetConnMaxLifetime(5 * time.Minute)
			db.SetMaxIdleConns(2)
			db.SetMaxOpenConns(8)

			return newDBConnObject(&nativeDBConn{
				driver: name,
				dsn:    dsn,
				db:     db,
			})
		}),
	})
}

func newDBConnObject(conn *nativeDBConn) *object.Hash {
	return hashObject(map[string]object.Object{
		"driver": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.String{Value: conn.driver}
		}),
		"begin": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			if conn.db == nil {
				return object.NewError(0, "database connection is closed")
			}
			tx, err := conn.db.BeginTx(context.Background(), nil)
			if err != nil {
				return object.NewError(0, "could not begin transaction: %s", err.Error())
			}
			return newDBTxObject(&nativeDBTx{
				driver: conn.driver,
				tx:     tx,
			})
		}),
		"exec": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if conn.db == nil {
				return object.NewError(0, "database connection is closed")
			}
			return dbExecObject(conn.db, "exec", args)
		}),
		"query": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if conn.db == nil {
				return object.NewError(0, "database connection is closed")
			}
			return dbQueryObject(conn.db, "query", args, false)
		}),
		"query_one": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if conn.db == nil {
				return object.NewError(0, "database connection is closed")
			}
			return dbQueryObject(conn.db, "query_one", args, true)
		}),
		"ping": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := conn.db.PingContext(ctx); err != nil {
				return object.NewError(0, "db ping failed: %s", err.Error())
			}
			return boolObject(true)
		}),
		"stats": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			stats := conn.db.Stats()
			return hashObject(map[string]object.Object{
				"driver":              &object.String{Value: conn.driver},
				"open_connections":    &object.Integer{Value: int64(stats.OpenConnections)},
				"in_use":              &object.Integer{Value: int64(stats.InUse)},
				"idle":                &object.Integer{Value: int64(stats.Idle)},
				"wait_count":          &object.Integer{Value: int64(stats.WaitCount)},
				"max_idle_closed":     &object.Integer{Value: int64(stats.MaxIdleClosed)},
				"max_lifetime_closed": &object.Integer{Value: int64(stats.MaxLifetimeClosed)},
			})
		}),
		"close": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			if conn.db == nil {
				return &object.Null{}
			}
			if err := conn.db.Close(); err != nil {
				return object.NewError(0, "could not close database: %s", err.Error())
			}
			conn.db = nil
			return &object.Null{}
		}),
	})
}

func newDBTxObject(tx *nativeDBTx) *object.Hash {
	return hashObject(map[string]object.Object{
		"driver": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return &object.String{Value: tx.driver}
		}),
		"exec": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if tx.closed || tx.tx == nil {
				return object.NewError(0, "transaction is closed")
			}
			return dbExecObject(tx.tx, "exec", args)
		}),
		"query": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if tx.closed || tx.tx == nil {
				return object.NewError(0, "transaction is closed")
			}
			return dbQueryObject(tx.tx, "query", args, false)
		}),
		"query_one": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if tx.closed || tx.tx == nil {
				return object.NewError(0, "transaction is closed")
			}
			return dbQueryObject(tx.tx, "query_one", args, true)
		}),
		"commit": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			if tx.closed || tx.tx == nil {
				return object.NewError(0, "transaction is closed")
			}
			if err := tx.tx.Commit(); err != nil {
				return object.NewError(0, "could not commit transaction: %s", err.Error())
			}
			tx.closed = true
			tx.tx = nil
			return &object.Null{}
		}),
		"rollback": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			if tx.closed || tx.tx == nil {
				return object.NewError(0, "transaction is closed")
			}
			if err := tx.tx.Rollback(); err != nil {
				return object.NewError(0, "could not rollback transaction: %s", err.Error())
			}
			tx.closed = true
			tx.tx = nil
			return &object.Null{}
		}),
		"is_closed": builtinFunc(func(env *object.Environment, args ...object.Object) object.Object {
			if len(args) != 0 {
				return object.NewError(0, "wrong number of arguments. got=%d, want=0", len(args))
			}
			return boolObject(tx.closed || tx.tx == nil)
		}),
	})
}

func parseDBCallArgs(name string, args []object.Object) (string, []any, *object.Error) {
	if len(args) != 1 && len(args) != 2 {
		return "", nil, object.NewError(0, "wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
	query, errObj := stringArg(args[0], fmt.Sprintf("first argument to `%s` must be STRING, got %%s", name))
	if errObj != nil {
		return "", nil, errObj
	}
	if len(args) == 1 {
		return query, nil, nil
	}
	if _, ok := args[1].(*object.Null); ok {
		return query, nil, nil
	}
	paramArray, ok := args[1].(*object.Array)
	if !ok {
		return "", nil, object.NewError(0, "second argument to `%s` must be ARRAY or NULL, got %s", name, args[1].Type())
	}
	params := make([]any, 0, len(paramArray.Elements))
	for _, item := range paramArray.Elements {
		value, errObj := dbValueFromObject(item)
		if errObj != nil {
			return "", nil, errObj
		}
		params = append(params, value)
	}
	return query, params, nil
}

func dbExecObject(executor dbQueryExecutor, name string, args []object.Object) object.Object {
	query, params, errObj := parseDBCallArgs(name, args)
	if errObj != nil {
		return errObj
	}

	result, err := executor.Exec(query, params...)
	if err != nil {
		return object.NewError(0, "db exec failed: %s", err.Error())
	}
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	return hashObject(map[string]object.Object{
		"rows_affected":  &object.Integer{Value: rowsAffected},
		"last_insert_id": &object.Integer{Value: lastInsertID},
	})
}

func dbQueryObject(executor dbQueryExecutor, name string, args []object.Object, one bool) object.Object {
	query, params, errObj := parseDBCallArgs(name, args)
	if errObj != nil {
		return errObj
	}
	rows, err := executor.Query(query, params...)
	if err != nil {
		return object.NewError(0, "db query failed: %s", err.Error())
	}
	defer rows.Close()

	records, errObj := rowsToArray(rows)
	if errObj != nil {
		return errObj
	}
	if one {
		if len(records.Elements) == 0 {
			return &object.Null{}
		}
		return records.Elements[0]
	}
	return records
}

func dbValueFromObject(obj object.Object) (any, *object.Error) {
	switch value := obj.(type) {
	case *object.String:
		return value.Value, nil
	case *object.Integer:
		return value.Value, nil
	case *object.Float:
		return value.Value, nil
	case *object.Boolean:
		return value.Value, nil
	case *object.Null:
		return nil, nil
	case *object.Array, *object.Hash:
		payload, err := json.Marshal(nativeValue(obj))
		if err != nil {
			return nil, object.NewError(0, "could not encode database parameter: %s", err.Error())
		}
		return string(payload), nil
	default:
		return nil, object.NewError(0, "unsupported database parameter type %s", obj.Type())
	}
}

func rowsToArray(rows *sql.Rows) (*object.Array, *object.Error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, object.NewError(0, "could not read row columns: %s", err.Error())
	}

	results := make([]object.Object, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		scans := make([]any, len(columns))
		for i := range values {
			scans[i] = &values[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, object.NewError(0, "could not scan row: %s", err.Error())
		}

		rowMap := make(map[string]object.Object, len(columns))
		for i, column := range columns {
			rowMap[column] = objectFromDBValue(values[i])
		}
		results = append(results, hashObject(rowMap))
	}
	if err := rows.Err(); err != nil {
		return nil, object.NewError(0, "row iteration failed: %s", err.Error())
	}
	return &object.Array{Elements: results}, nil
}

func objectFromDBValue(value any) object.Object {
	switch v := value.(type) {
	case nil:
		return &object.Null{}
	case string:
		return &object.String{Value: v}
	case []byte:
		return &object.String{Value: string(v)}
	case int64:
		return &object.Integer{Value: v}
	case float64:
		return &object.Float{Value: v}
	case bool:
		return boolObject(v)
	case time.Time:
		return &object.String{Value: v.Format(time.RFC3339)}
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", v))
		return &object.String{Value: text}
	}
}
