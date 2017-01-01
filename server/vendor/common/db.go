package common

import "github.com/mmitevski/transactions/db"

var database db.Database

func DB() db.Database {
	if database == nil {
		config := GetConfig()
		database = db.NewDatabase(&config.Database)
	}
	return database
}