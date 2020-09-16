package model

import "reflect"

// GetUpsertAllColumns return all column names
func GetUpsertAllColumns(value interface{}) []string {
	tx := db.Model(value)
	el := reflect.ValueOf(value).Elem()
	s := []string{}
	for i := 0; i < el.NumField(); i++ {
		t := el.Type().Field(i)
		f := tx.Statement.Schema.ParseField(t)
		if !f.Updatable {
			continue
		}
		if f.DBName == "" {
			f.DBName = tx.NamingStrategy.ColumnName("", f.Name)
		}
		s = append(s, f.DBName)
	}
	return s
}

// GetUpsertAppendColumns return all column names
func GetUpsertAppendColumns(value interface{}, columns []string) []string {
	tx := db.Model(value)
	el := reflect.ValueOf(value).Elem()
	for i := 0; i < el.NumField(); i++ {
		t := el.Type().Field(i)
		f := tx.Statement.Schema.ParseField(t)

		if _, ok := f.TagSettings["AUTOUPDATETIME"]; ok || (f.Name == "UpdatedAt" && (f.DataType == "time" || f.DataType == "int" || f.DataType == "uint")) {
			if f.DBName == "" {
				f.DBName = tx.NamingStrategy.ColumnName("", f.Name)
			}
			columns = append(columns, f.DBName)
			continue
		}
	}

	return columns
}
