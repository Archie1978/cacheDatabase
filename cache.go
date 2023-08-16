package cacheDatabase

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	gocache "github.com/patrickmn/go-cache"

	//"gorm.io/gorm/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	schemaSQL "gorm.io/gorm/schema"
)

var (

	// SEPARATOR_FIELDS filedName/value into string transformation cache
	SEPARATOR_FIELDS string = "||"

	// ErrRecordNotFound record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound

	// ErrRecordNotFound stucture not found error
	ErrStructureNotFound = fmt.Errorf("Impossible trouver la structure")

	// ErrErrNotPointer
	ErrErrNotPointer = fmt.Errorf("LA structure n'est pas un pointer")
)

/*
 *
 * Cache read
 *
 */

// Cache of many TABLE (example *cacheDatabase.TestFile)
var cacheTableSQL map[string]cacheSQL

// Cache of many keys ("ID",....)
type cacheSQL map[string]*gocache.Cache

func getStructName(object interface{}) string {
	return reflect.TypeOf(object).String()
}

func InsertCache(object interface{}) error {

	objectType := reflect.TypeOf(object)
	if objectType.Kind() != reflect.Ptr {
		return ErrErrNotPointer
	}

	listKeyUnique := getListKeys(object)
	for _, keyUnique := range listKeyUnique {
		listField := strings.Split(keyUnique, SEPARATOR_FIELDS)

		var key = ""
		var keyValue = ""
		for _, fieldname := range listField {
			key += fieldname + SEPARATOR_FIELDS
			keyValue += getValueOfKey(object, fieldname) + SEPARATOR_FIELDS
		}
		if len(key) >= 2 {
			key = key[:len(key)-2]
		}
		if len(keyValue) >= 2 {
			keyValue = keyValue[:len(keyValue)-2]
		}

		structName := getStructName(object)

		if _, ok := cacheTableSQL[structName]; !ok {
			cacheTableSQL[structName] = make(cacheSQL)
		}
		if _, ok := cacheTableSQL[structName][key]; !ok {
			cacheTableSQL[structName][key] = gocache.New(5*time.Minute, 10*time.Minute)
		}

		if (keyValue != "") && (keyValue != "0") {
			cacheTableSQL[structName][key].Add(keyValue, object, gocache.DefaultExpiration)
		}
	}
	return nil
}

// GetCache recupere une donnée de type object et retourne le resultat
func GetCache(object interface{}, fieldIndexName string, values ...string) (interface{}, error) {

	objectValue := reflect.ValueOf(object)
	if objectValue.Kind() == reflect.Ptr {
		return GetCache(objectValue.Elem().Interface(), fieldIndexName, values...)
	}

	if objectValue.Kind() != reflect.Struct {
		return nil, ErrStructureNotFound
	}

	structName := "*" + getStructName(objectValue.Interface())
	if _, ok := cacheTableSQL[structName]; !ok {
		return nil, ErrRecordNotFound
	}

	if _, ok := cacheTableSQL[structName][fieldIndexName]; !ok {
		return nil, ErrRecordNotFound
	}

	object, ok := cacheTableSQL[structName][fieldIndexName].Get(strings.Join(values, SEPARATOR_FIELDS))
	if !ok {
		return nil, ErrRecordNotFound
	}

	// Update use cache
	InsertCache(object)

	return object, nil
}

// databaseGetWithCache: get value into cache or into database, Return l'address de l'object,  cache used and error
func databaseGetWithCache(objectPtr interface{}, fieldIndexName string, values ...string) (interface{}, bool, error) {
	return databaseGetCompositeWithCache(objectPtr, nil, fieldIndexName, values...)
}

// Utilisation d'un cache suivant index demander la valeur du champs est installé dans la structure
// Return l'address de l'object,  cache used and error
func databaseGetCompositeWithCache(objectPtr interface{}, compositfonction func(newObjectPtr interface{}) error, fieldIndexName string, values ...string) (interface{}, bool, error) {

	// Check fieldIndexName
	objectCache, err := GetCache(objectPtr, fieldIndexName, values...)
	if err == nil {
		return objectCache, true, nil
	}

	// Create new element for SQL request
	objectPtrValue := reflect.ValueOf(objectPtr)
	objectValue := objectPtrValue.Elem()
	newObjectPtrValue := reflect.New(objectValue.Type())
	newObjectPtr := newObjectPtrValue.Interface()

	// Get real name into SQL
	s, err := schemaSQL.Parse(objectPtr, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return nil, false, err
	}

	// Create request filter
	listColumns := strings.Split(fieldIndexName, SEPARATOR_FIELDS)
	columnRequest := make([]string, 0, 3)
	for _, columnNameQuery := range listColumns {
		var columnName string
		for _, f := range s.Fields {
			if columnNameQuery == f.Name {
				columnName = f.DBName
			}
		}
		columnRequest = append(columnRequest, columnName+"=?")
		if columnName == "" {
			return nil, false, fmt.Errorf("ColumnName not found: %v into %v", columnNameQuery, fieldIndexName)
		}
	}
	columnRequestQuery := strings.Join(columnRequest, " AND ")
	query := make([]interface{}, 1+len(columnRequest))
	query[0] = columnRequestQuery
	for i, val := range values {
		query[i+1] = val
	}

	// send request SQL
	glog.Info("databaseGetWithCache:", newObjectPtr)
	glog.Info("databaseGetWithCache:", len(query), query)
	glog.Info("databaseGetWithCache:", query[0], query[1])

	// get into Database
	ret := DB.First(newObjectPtr, query...)
	if ret.Error == nil {
		if compositfonction != nil {
			compositfonction(newObjectPtr)
		}
		err := InsertCache(newObjectPtr)
		if err != nil {
			return nil, false, err
		}
	}
	return newObjectPtr, false, ret.Error
}

func databaseGet(listObjectInterface interface{}, where string) error {
	// create list object
	ret := DB.Where(where).Find(listObjectInterface)
	if ret.Error != nil {
		return ret.Error
	}

	return nil
}

func DisplayCache(cache cacheSQL) {
	for key, store := range cache {
		glog.Info("Index:", key)
		for key, items := range store.Items() {
			glog.Info("\t", key, "  ", items.Object)
		}
	}
}
