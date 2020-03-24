# Database-sql 包

## import package

不要直接使用驱动, 而是使用 database/sql 包.

``` go
import (
  "database/sql"
  // for mysql
  _ "github.com/go-sql-driver/mysql"
)
```

## Open/Close

sql.Open 返回一个 *sql.DB 对象. 但是, 不会建立实际的连接. 如果要检查连接, 使用 db.Ping(). 记住, 检查错误.
*sql.DB 对象底层有连接池. 不要频繁的 Open, Close *sql.DB 对象. 如果需要, 传递它.

``` go
db, err := sql.Open("mysql",
  "user:password@tcp(127.0.0.1:3306)/db-name[?param1=value1&...&paramN=valueN]") // DSN 参考: https://github.com/go-sql-driver/mysql
if err != nil {
  log.Fatal(err)
}
defer db.Close()

err = db.Ping()
if err != nil {
  log.Fatal(err)
}
```

## 连接池

db.SetMaxIdleConns(N) 用于设置最多保留处于 idle 的连接数. 默认: 2. 如果 N <= 0, 表示不保留 idle 连接. 空闲连接太多, 占用资源.
db.SetMaxOpenConns(N) 用于设置最多打开的连接数. 默认: 0, 无限制. 如果 MaxIdleConns > 0 and MaxIdleConns > MaxOpenConns, 那么 MaxIdleConns 将缩小, 以匹配新的 MaxOpenConns.
db.SetConnMaxLifetime(time.Duration) 用于设置连接的最大生存时间. 如果 Duration <= 0  表示无限制. 要小于后端数据库的 wait_timeout或数据库反向代理 Nginx 设置的 proxy_timeout. 否则, 会被后端服务关闭, 造成 invalid connection 错误.

## 获取结果集

有如下方法获取结果集:

1. Execute 一个查询返回 rows.
2. Prepare 一个 statement 对象, 重复使用之, 然后再 Close. Prepare 语句是为多次执行设计的.
3. Execute 一个一次性的语句
4. Execute 一个查询返回一个单行. 有一个快捷方式用于这个案例.

如果函数名包含 Query , 那么这个函数会查询数据库返回一个 rows , 即使是空的. 必须 defer rows.Close() 保证 rows 关闭. 否则, rows 关联的数据库连接始终被持有.

最好避免使用 [NULL](http://go-database-sql.org/nulls.html) 的列.

### 查询

查询并获取数据的方法如下.

``` go
var (
  id int
  name string
)
rows, err := db.Query("select id,name from users where id = ?", 1)
if err != nil {
  log.Fatal(err)
}
// 重要. 不关闭, 可能会资源一直占用.
defer rows.Close()

for rows.Next() {
  err := rows.Scan(&id, &name)
  // 重要.
  if err != nil {
    log.Fatal(err)
  }
  log.Println(id, name)
}
//重要. for rows.Next() 也可能有错误
err = rows.Err()
if err != nil {
  log.Fatal(err)
}

```

rows.Close() 必须要调用, 多次调用是无害操作. 但是, 不调用, 如果不关闭, 底层的连接一直占用. 虽然, 当遍历到最后一条记录时, 会发生内部 EOF 错误, 自动调用 rows.Close(), 但是, 如果循环提前退出, rows 不会被自动关闭.

### 单行查询

如果查询最多返回一行结果. 那么, 有一个快捷的方式用于该案例.

``` go
var name string

// 在 Scan 后才可能返回 Error
err := db.QueryRow("select name from users where id = ?", 1).Scan(&name)
if err != nil {
  if err == sql.ErrNoRows {
    // there were no rows
  } else {
    log.Fatal(err)
  }
}

```

当仅返回单行时, 也可以在 Prepare statement 对象上使用 QueryRow.

``` go
stmt, err := db.Prepare("select name from users where id = ?")
if err != nil {
  log.Fatal(err)
}
defer stmt.Close()
var name string
err = stmt.QueryRow(1).Scan(&name)
if err != nil {
  log.Fatal(err)
}

```

### Prepared Statement

Prepared Statement 有很多好处: 安全, 高效, 方便.
Prepared Statement 执行过程(至少 3 次通讯):

1. Prepare 执行时绑定一个连接, 发送带占位符的语句到服务端. 服务端响应一个 statement ID 给客户端
2. 客户端调用 Exec 时发送 statement ID 和 参数给服务端. 服务端返回结果.
3. 最后, 客户端关闭 statement 对象, 同时会将关闭请求发给服务端, 服务端关闭 statement 对象.

由于在 Golang 中, 连接不直接暴露给 database/sql 包. 没办法在指定连接上进行 Prepare Statement. 只能在 DB 或者 Tx 对象上 Prepare Statement. 同时, 在底层连接无效时, database/sql 包提供自动重试获取连接功能. 在下面的特定情况下会造成创建大量连接.

1. 当客户端 Prepare statement 时, 它是在一个连接池上 Prepare 的. 它会取一个空闲的连接执行 Prepare 语句, 然后把该连接释放回连接池.
2. Statement 对象会记住这个 Prepare 使用的连接.
3. 在 stmt.Exec 时, 如果 Prepare 的连接正被使用或已被关闭, 底层实现会重新选一个空闲(没有空闲就创建)连接重新执行 Prepare, 执行完又释放连接回连接池.
4. 在高并发场景, 很可能出现不断重复执行 Prepare 的情况.

使用 Prepare Statement 需要注意:

1. 单次查询不要使用 Prepared Statement. 每次使用最少 3 次通讯.
2. 不要在循环中调用 Prepare 语句.
3. 要关闭 statement 对象.
4. [MySQL Prepared Statement](https://dev.mysql.com/doc/refman/5.7/en/prepare.html) 的作用域是 session 级别的, 只对创建它的 session 有效, 不在 session 间共享. session 关闭, Prepared Statement 被释放(DEALLOCATE PREPARE).

#### 事务中使用 Prepared Statement

事务对象是绑定连接的. 在事务提交或回滚后, 其绑定的连接释放回连接池.
注意: 在 tx 变量作用域内不要再调用 db 变量的任何函数.

``` go
tx, err := db.Begin()
if err != nil {
  log.Fatal(err)
}

defer tx.Rollback()
// 截止 Go 1.13.5 , 还不支持一个事务中同时存在多个语句. 参考: http://go-database-sql.org/surprises.html
// 每个语句必须按顺序执行, 并且必须扫描或关闭结果中的资源(Row 或 Rows).
// 然后再执行下一个语句.
stmt, err := tx.Prepare("insert into foo values(?")
if err != nil {
  log.Fatal(err)
}
defer stmt.Close()
for i := 0; i < 10; i++ {
  _, err = stmt.Exec(i)
  // 重要. 需要检查 err
  if err != nil {
    log.Fatal(err)
  }
}
// tx.Commit() 或 tx.Rollback() 会关闭 stmt
err = tx.Commit()
if err != nil {
  log.Fatal(err)
}
```

#### Parameter Placeholder Syntax

| MySQL | Oracel | PostgreSQL |
| --- | --- | --- |
| WHERE col = ? | WHERE col = :col | WHERE col = $1 |
| VALUE(?, ?, ?) | VALUE(:val1, :val2, :val3) | VALUE($, $2, $3) |

## 修改数据

### 使用 Statement 修改数据

使用 stmt.Exec() 完成 INSERT, UPDATE, DELETE 语句, 不会返回 rows.

``` go
stmt, err := db.Prepare("INSERT INTO users(name) VALUES(?)")
if err != nil {
  log.Fatal(err)
}
defer stmt.Close()
res, err := stmt.Exec("Dolly")
if err != nil {
  log.Fatal(err)
}
lastId, err := res.LastInsertId()
if err != nil {
  log.Fatal(err)
}
rowCnt, err := res.RowsAffected()
if err != nil {
  log.Fatal(err)
}

```

如果不关心返回的结果, 但必须检查错误可以使用 Exec()

``` go
_, err := db.Exec("DELETE FROM users") // OK
_, err := db.Query("DELETE FROM users") // BAD. 造成 rows 无法关闭.
```

## 错误处理

### 处理特定的数据库错误

``` go
import (
  "github.com/VividCortex/mysqlerr"
)
if driverErr, ok := err.(*mysql.MySQLError); ok {
  if driverErr == mysqlerr.ER_ACCESS_DENIED_ERROR {
    // 处理访问被拒绝的错误
  }
}
```

### 处理连接错误

不需要处理连接错误. databbase/sql 连接池会处理连接错误. 它会重新获取一个(新)连接进行重试, 最多 10 次.

## 安全

[避免 SQL 注入](https://learnku.com/docs/build-web-application-with-golang/094-avoids-sql-injection/3212)
[sqli-lab 教程](https://blog.csdn.net/qq_41420747/article/details/81836327)
