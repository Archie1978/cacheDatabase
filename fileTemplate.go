package cacheDatabase

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strings"

	"github.com/golang/glog"
	"gorm.io/gorm"
)

var cacheFileCache map[string]cacheSQL
var DEBUG bool

type FileTemplate struct {
	ID       uint   `gorm:"primarykey"`
	ParentID uint   `gorm:"index:idx_parentid_name,unique"`
	Name     string `gorm:"index:idx_parentid_name,unique"`

	// Path: Indicatif
	//Parent *FileTemplate `sql:"-"`

	// Path: Indicatif
	Path       string `gorm:"-"`
	pathParent string
}

// GetParent
func (fileTemplate *FileTemplate) GetPathParent() (pathParent string) {
	if pathParent != "" {
		return pathParent
	}
	fileTemplate.pathParent, _ = splitPath(fileTemplate.Path)
	return fileTemplate.pathParent
}

// PreSave: function call just before save into routine with  transaction
func (fileTemplate *FileTemplate) PreSave(db *gorm.DB, objectPtr interface{}) error {

	if DEBUG {
		glog.Info("PreSave: 1:", objectPtr)
	}
	// Get Parent: Parent is always into cache. It was pushed from save function
	pathParent, name := splitPath(fileTemplate.Path)
	if name == "" {
		return nil
	}

	// Cache insert form create
	objectParent, err := GetCache(objectPtr, "Path", pathParent)
	if DEBUG {
		glog.Info("PreSave: GetCache parent:", pathParent, err)
	}
	if err != nil {
		if err == ErrRecordNotFound {
			if pathParent == "" {
				// Parent is root not neccessary into cache
				// Create Object
				objectRootPtrValue := reflect.TypeOf(objectPtr)
				newObjecttRootPtrValue := reflect.New(objectRootPtrValue.Elem())
				fileTemplateRoot := getFileTemplateIntoInterface(newObjecttRootPtrValue.Interface())
				fileTemplateRoot.ID = 1
				fileTemplateRoot.ParentID = 1
				objectParent = newObjecttRootPtrValue.Interface()
				InsertCache(objectParent)
			} else {
				objectRootPtrValue := reflect.TypeOf(objectPtr)
				newObjecttRootPtrValue := reflect.New(objectRootPtrValue.Elem())
				objectTemplateParent := getFileTemplateIntoInterface(newObjecttRootPtrValue.Interface())
				_, parentName := splitPath(pathParent)
				objectTemplateParent.Name = parentName
				objectTemplateParent.Path = pathParent
				objectParent = newObjecttRootPtrValue.Interface()
				InsertCache(objectParent)
			}
		} else {
			return err
		}
	}

	// get FileTemplate int parent
	fileTemplateParent := getFileTemplateIntoInterface(objectParent)
	if DEBUG {
		glog.Info("PreSave: GetCache parent: ID:", fileTemplateParent.ID, objectParent)
	}
	if fileTemplateParent.ID == 0 {
		err = fileTemplateParent.PreSave(db, objectParent)
		if err != nil {
			return err
		}
	}
	if DEBUG {
		glog.Info("PreSave: fileTemplate.ParentID=", fileTemplate.ParentID)
	}

	// Update ParentID
	if fileTemplate.ParentID == 0 {
		fileTemplate.ParentID = fileTemplateParent.ID
	}

	// Save into BDD
	ret := db.Save(objectPtr)

	//Error data from another node
	if ret.Error != nil {
		if strings.Contains(ret.Error.Error(), "constraint failed") {
			retFind := db.Where("name = ? AND parent_id = ?", fileTemplate.Name, fileTemplate.ParentID).First(objectPtr)
			if retFind.Error != nil {
				log.Fatal("Err:", retFind.Error)
			}
			InsertCache(objectPtr)
			return nil
		}
	}

	if DEBUG {
		glog.Info("PreSave: insert object into cache:", objectPtr)
	}
	InsertCache(objectPtr)
	return ret.Error
}

// ListKeyUnique: List index into cache memory
func (fileTemplate FileTemplate) ListKeyUnique() []string {
	return []string{"ID", "Path"}
}

func getFileTemplateIntoInterface(pathInterfaceStructPtr interface{}) *FileTemplate {

	// Check strut native
	if fmt.Sprintf("%T", pathInterfaceStructPtr) == "*cacheDatabase.FileTemplate" {
		return pathInterfaceStructPtr.(*FileTemplate)
	}

	valuePtr := reflect.ValueOf(pathInterfaceStructPtr)
	if valuePtr.Type().Kind() != reflect.Ptr {
		return nil
	}
	te := valuePtr.Elem()

	// Looking for membre parents
	var fileTemplate *FileTemplate
	for i := 0; i < te.NumField(); i++ {
		typeField := te.Field(i).Type()
		if typeField.Name() == "FileTemplate" && typeField.PkgPath() == "cacheDatabase" {
			return te.Field(i).Addr().Interface().(*FileTemplate)
		}
	}
	return fileTemplate
}

// splitPath: retourne path parent and name of file
func splitPath(p string) (string, string) {
	listPath := strings.Split(p, "/")
	l := len(listPath)
	if len(listPath) == 1 {
		//Root
		return p, ""
	}
	nameFile := listPath[l-1]
	pathParent := listPath[0 : l-1]
	return strings.Join(pathParent, "/"), nameFile
}

// GetPath: function return object Ptr From pathInterfaceStructPtr Type with the path
func GetPath(pathInterfaceStructPtr interface{}, path string) (interface{}, error) {
	instance := rand.Int() % 1000
	if DEBUG {
		glog.Info("GetPath: %v Init  %T, %v\n", instance, pathInterfaceStructPtr, path)
	}

	// Check into Cache
	recordCache, err := GetCache(pathInterfaceStructPtr, "Path", path)
	if err == nil {
		if DEBUG {
			glog.Info("GetCache: ", instance, "Use Cache : ", path)
		}
		return recordCache, nil
	}

	// Split Name and parent path
	pathParent, nameFile := splitPath(path)
	if DEBUG {
		glog.Info("GetPath:", instance, " Split  ", nameFile, pathParent)
	}
	if nameFile == "" {
		// Stop propagation
		fmt.Println("GetPath: ", instance, ",Root Found return ID:1,Name:\"\"  ")
		return &FileTemplate{ID: 1, ParentID: 1, Name: ""}, nil
	}

	// Check path into cache of path so to down root
	interfacePathParentPtr, err := GetPath(pathInterfaceStructPtr, pathParent)
	if err != nil {
		if DEBUG {
			glog.Info("Failed path parent", pathParent, err)
		}
		return nil, err
	}

	// Get Template of parent
	fileTemplateParent := getFileTemplateIntoInterface(interfacePathParentPtr)
	if fileTemplateParent == nil {
		return nil, fmt.Errorf("FileTemplate not found into: [%T] %v	", interfacePathParentPtr, interfacePathParentPtr)
	}
	if DEBUG {
		glog.Info("GetPath: ", instance, " databaseGetWithCache:", pathInterfaceStructPtr)
	}
	record, _, err := databaseGetCompositeWithCache(pathInterfaceStructPtr,
		func(newObject interface{}) error {
			// Rempli le path avant d'index
			fileTemplateNewRecord := getFileTemplateIntoInterface(newObject)
			fileTemplateNewRecord.Path = fileTemplateParent.Path + "/" + fileTemplateNewRecord.Name
			return nil
		},
		"Name"+SEPARATOR_FIELDS+"ParentID",
		nameFile,
		fmt.Sprintf("%v", fileTemplateParent.ID),
	)

	if DEBUG {
		fmt.Println("GetPath: ", instance, " databaseGetWithCache", err, pathParent)
	}
	if err != nil {
		return nil, err
	}
	return record, nil

}

// SavePath:  Save data
func SavePath(pathInterfaceStructPtr interface{}) (pathInterfaceStructPtrInCache interface{}, err error) {
	instance := rand.Int() % 1000
	if DEBUG {
		fmt.Printf("SavePath: %v Init  %T, %v\n", instance, pathInterfaceStructPtr, pathInterfaceStructPtr)
	}

	// Get Template of parent
	fileTemplate := getFileTemplateIntoInterface(pathInterfaceStructPtr)
	if fileTemplate == nil {
		return nil, fmt.Errorf("FileTemplate not found into: [%T] %v	", pathInterfaceStructPtr, pathInterfaceStructPtr)
	}

	// Check Name
	if fileTemplate.Name == "" && fileTemplate.Path != "" {
		_, fileTemplate.Name = splitPath(fileTemplate.Path)
	}

	// Check into Cache
	_, err = GetCache(pathInterfaceStructPtr, "Path", fileTemplate.Path)
	fmt.Println("<==========fileTemplate=======>", fileTemplate, err)
	if err == nil {
		if DEBUG {
			glog.Info("GetCache: ", instance, "Use Cache : ", fileTemplate.Path)
		}
		return SaveData(pathInterfaceStructPtr), fmt.Errorf("Impossible de sauver, existe deja en read cache")
	}

	// Insert parent into cache for speed reading
	err = insertParentIntoCache(pathInterfaceStructPtr)
	if err != nil {
		return pathInterfaceStructPtr, err
	}

	//Backup data
	err = SaveData(pathInterfaceStructPtr)

	return pathInterfaceStructPtr, err
}

// insert Parent: unexist into Cache for record
func insertParentIntoCache(objectInterfaceStructPtr interface{}) error {
	glog.Info("insertParentIntoCache: ", objectInterfaceStructPtr)

	// Get Template of parent
	fileTemplate := getFileTemplateIntoInterface(objectInterfaceStructPtr)
	if fileTemplate == nil {
		return fmt.Errorf("FileTemplate not found into: [%T] %v	", objectInterfaceStructPtr, objectInterfaceStructPtr)
	}

	if fileTemplate.GetPathParent() == "" {
		return nil
	}

	// Check if parent existe into database
	_, err := GetCache(
		objectInterfaceStructPtr,
		"Path",
		fileTemplate.GetPathParent(),
	)

	if DEBUG {
		fmt.Println("insertParentIntoCache:", fileTemplate.GetPathParent(), ":", err, err == ErrRecordNotFound)
	}
	if err == ErrRecordNotFound {
		// Create parent into cache
		objectParentInterface, err := CreatePath(objectInterfaceStructPtr, fileTemplate.GetPathParent())
		if err == ErrRecordExistCache {
			return nil
		} else {
			if err == nil {
				return insertParentIntoCache(objectParentInterface)
			}
			return err
		}

	}
	return nil
}

var ErrRecordExistCache error = fmt.Errorf("Record Exist into cache")

// Create Path into cache
func CreatePath(objectInterfaceStructPtr interface{}, path string) (interface{}, error) {
	if DEBUG {
		glog.Info("CreatePath:", objectInterfaceStructPtr)
	}

	// Check if parent existe into database
	objectCache, _, err := databaseGetWithCache(
		objectInterfaceStructPtr,
		"Path",
		path,
	)
	if DEBUG {
		glog.Info("CreatePath2:", objectCache, err)
	}
	if err == nil {
		return objectCache, ErrRecordExistCache
	}

	// Create Object
	objectPtrValue := reflect.TypeOf(objectInterfaceStructPtr)
	newObjectPtrValue := reflect.New(objectPtrValue.Elem())

	if DEBUG {
		glog.Info("CreatePath3:", newObjectPtrValue.Interface())
	}

	// Get Template of parent
	fileTemplate := getFileTemplateIntoInterface(newObjectPtrValue.Interface())
	if fileTemplate == nil {
		return nil, fmt.Errorf("FileTemplate not found into: [%T] %v	", newObjectPtrValue.Interface(), newObjectPtrValue.Interface())
	}

	_, namefile := splitPath(path)
	fileTemplate.Name = namefile
	fileTemplate.Path = path

	// Insert into Cache Read
	err = InsertCache(newObjectPtrValue.Interface())

	return newObjectPtrValue.Interface(), err
}
