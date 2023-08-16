package cacheDatabase

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/glog"
	"gorm.io/gorm"
)

var DB *gorm.DB

var cacheWriteTable map[string]typeArrayCacheWrite

type typeArrayCacheWrite []interface{}

var cacheWriteLocker sync.Mutex

var chanelCommit chan (bool)
var commitRelease sync.Mutex

func init() {
	// Init cache read
	cacheTableSQL = make(map[string]cacheSQL)

	// Init cache Write
	cacheWriteTable = make(map[string]typeArrayCacheWrite)
	chanelCommit = make(chan (bool))
	go startSaveGoroutine()

}

// saveData: Save data and insert into cache
func SaveData(data interface{}) error {
	nameObject := reflect.TypeOf(data).String()

	err := InsertCache(data)
	if err != nil {
		return err
	}

	// Wait write database
	cacheWriteLocker.Lock()
	defer cacheWriteLocker.Unlock()

	if _, ok := cacheWriteTable[nameObject]; !ok {
		cacheWriteTable[nameObject] = make(typeArrayCacheWrite, 0, 20)
	}

	cacheWriteTable[nameObject] = append(cacheWriteTable[nameObject], data)
	return nil
}

func CommitData() {
	commitRelease.TryLock()
	chanelCommit <- true
	commitRelease.Lock()
}

func startSaveGoroutine() {
	for {

		channelCommit := false
		select {
		case <-time.After(5 * time.Second):
		case <-chanelCommit:
			channelCommit = true
		}

		trans := DB.Begin()

		// Start write into transaction
		cacheWriteLocker.Lock()
		for nametable, listRecordTable := range cacheWriteTable {

			for _, record := range listRecordTable {
				fmt.Println(record)

				// Use PreSave Instruction
				err := callRecordPreSaveStruct(trans, record)
				if err != nil {
					glog.Error("PreSaveFailed: %v,%v", record, err)
				} else {
					// Update sql
					ret := trans.Save(record)
					if ret.Error != nil {
						glog.Error("Error write database: %s", ret.Error)
					} else {
						InsertCache(record)
					}
				}
			}

			// Clean cache into cache
			cacheWriteTable[nametable] = make(typeArrayCacheWrite, 0)
		}

		// Unlock Writer cache lock
		cacheWriteLocker.Unlock()

		// Commit transaction
		trans.Commit()

		// Notify commit
		if channelCommit {
			commitRelease.Unlock()
		}
	}
}

func callRecordPreSaveStruct(trans *gorm.DB, record interface{}) error {
	transValue := reflect.ValueOf(trans)
	recordValue := reflect.ValueOf(record)
	functionPreSaveValue := reflect.ValueOf(record).MethodByName("PreSave")
	if functionPreSaveValue.IsValid() {
		returnlistValue := functionPreSaveValue.Call([]reflect.Value{transValue, recordValue})
		if len(returnlistValue) == 0 {
			return fmt.Errorf("Le parametre renvoie au moins une valeur")
		}

		if !returnlistValue[0].IsNil() {
			return fmt.Errorf("%v", returnlistValue[0].Interface())
		}
	}
	return nil
}
