package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"d01t.now/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func openDB(dsn string) (*gorm.DB, error) {
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	return database, nil
}

func configConn(db *sql.DB) {
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	db.SetMaxIdleConns(2)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	db.SetMaxOpenConns(10)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	db.SetConnMaxLifetime(time.Minute * 10)
}

func merge(db *gorm.DB, inits ...func(db *gorm.DB)) {
	for _, f := range inits {
		f(db)
	}
}

func genEmployee(lastNumber uint) *model.Employee {
	return &model.Employee{
		Name:   fmt.Sprintf("emp%03d", lastNumber+1),
		Email:  fmt.Sprintf("emp03d@example.org", lastNumber),
		Number: lastNumber + 1,
	}
}

// 参考: https://gorm.io/zh_CN/docs/create.html
func insertEmployee(db *gorm.DB) {
	// 查询最后一个, 按主键降序
	var empLast model.Employee
	db.Last(&empLast)
	fmt.Println("last employee ID: ", empLast.ID, ", Number: ", empLast.Number)
	emp := genEmployee(empLast.Number)
	// 创建单一对象
	result := db.Create(&emp)
	// 批量插入, 使用 slice, 比如: []Employee

	// 根据 Map 创建
	// 单一对象: map[string]interface{}
	// 批量: []map[string]interface{}{}
	// 注意: 根据 map 创建时, 关联不会被调用, 且主键也不会自动填充

	if result.Error != nil {
		fmt.Println(result.Error)
		return
	}
	// 影响的行数
	fmt.Println("影响的行数: ", result.RowsAffected)
	// 返回插入的 主键
	fmt.Println("新建数据的主键: ", emp.ID)
}

func main() {
	var err error
	db, err = openDB("test.sqlite3")
	// GORM 使用 database/sql 维护连接池
	if err != nil {
		log.Fatal(err)
	}
	// gorm.DB.DB() 可以返回 sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	configConn(sqlDB)
	merge(db, model.InitEmployee)
	insertEmployee(db)
}
