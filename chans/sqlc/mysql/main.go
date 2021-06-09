// Package main exists to act as a Go plug-in for a MySQL driver for
// the Plax SQL channel type.
package main

import (
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// We do not have to register the mysql driver manually
	// because the Go plug-in runtime will call
	// github.com/go-sql-driver/mysql's init function
	// automatically, and that function registers the driver.
	//
	// sql.Register("mysql", &mysql.MySQLDriver{})
}

func main() {
}
