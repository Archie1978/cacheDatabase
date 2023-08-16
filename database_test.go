package cacheDatabase

import (
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"testing"
)

type SchoolSimply struct {
	ID   uint `gorm:"primarykey"`
	Name string
}

func (school SchoolSimply) ListKeyUnique() []string {
	return []string{"ID", "Name"}
}

type SchoolSimply2 struct {
	ID                   uint `gorm:"primarykey"`
	Name                 string
	StreetAdministration string
}

func (school SchoolSimply2) ListKeyUnique() []string {
	return []string{"ID", "Name"}
}

type SchoolSimply3 struct {
	ID                   uint `gorm:"primarykey;autoIncrement:true"`
	Name                 string
	ParentID             uint
	StreetAdministration string
}

func (school SchoolSimply3) ListKeyUnique() []string {
	return []string{"ID", "ParentID,Name"}
}

func TestCache(t *testing.T) {
	s1 := &SchoolSimply{ID: 1, Name: "Jaures"}
	err := InsertCache(&s1)
	if err == nil {
		t.Fatal("Cas impossible car il faut mettre un pointer")
	}
	s2 := SchoolSimply{ID: 1, Name: "Jaures"}
	err = InsertCache(&s2)
	if err != nil {
		t.Fatal("Error cache:", err)
	}

	t.Log(cacheTableSQL["*cacheDatabase.SchoolSimply"]["ID"].Get("1"))
	t.Log(cacheTableSQL["*cacheDatabase.SchoolSimply"]["Name"].Get("Jaures"))

	// Recupere cache
	var school *SchoolSimply
	_, err = GetCache(&school, "ID", "1")
	t.Log(err, school)
}

func TestSaveDB(t *testing.T) {

	os.Remove("gorm_test.db")
	db, err := gorm.Open(sqlite.Open("gorm_test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&SchoolSimply2{})

	DB = db
	s := SchoolSimply2{Name: "coucou"}
	SaveData(&s)
	CommitData()

	var s4 SchoolSimply2
	ret := db.Debug().First(&s4, 1)
	if ret.Error != nil {
		t.Fatal(ret.Error)
	}

	s6, ok := cacheTableSQL["*cacheDatabase.SchoolSimply2"]["Name"].Get("coucou")
	if !ok {
		t.Fatal("Il doit noralement y avoir un cache")
	}
	if s6 == nil {
		t.Fatal("Il doit noralement y avoir un cache")
	}
	if s6.(*SchoolSimply2).Name != "coucou" {
		t.Fatal("Il doit erreur de cache")
	}
	if s6.(*SchoolSimply2).ID != 1 {
		t.Fatal("Il doit erreur de cache apres enregistrement")
	}

	if s4.Name != "coucou" {
		t.Fatal("Mauvais enregistrement dans la base du Name")
	}
	if s4.ID != 1 {
		t.Fatal("Mauvais enregistrement dans la base de ID")
	}

	conn, _ := DB.DB()
	conn.Close()
}

func TestLoadDB(t *testing.T) {
	os.Remove("gorm_test2.db")
	db, err := gorm.Open(sqlite.Open("gorm_test2.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	db.AutoMigrate(&SchoolSimply2{}, &SchoolSimply3{})
	DB = db
	Name := "It is me"
	s := SchoolSimply2{Name: Name}
	err = SaveData(&s)
	if err != nil {
		t.Fatal("Error save")
	}
	CommitData()

	// Get venant du cache
	var s2 SchoolSimply2
	databaseGetWithCache(&s2, "Name", Name)
	if s2.Name != Name {
		t.Fatal("Mauvais enregistrement dans la base du Name")
	}
	if s2.ID != 1 {
		t.Fatal("Mauvais enregistrement dans la base de ID")
	}

	//destroyed cache
	cacheTableSQL = make(map[string]cacheSQL)

	// get without cache
	var s3 SchoolSimply2
	databaseGetWithCache(&s3, "Name", Name)
	if s3.Name != Name {
		t.Fatal("Mauvais enregistrement dans la base du Name")
	}
	if s3.ID != 1 {
		t.Fatal("Mauvais enregistrement dans la base de ID")
	}

	// Multi key
	s21 := SchoolSimply3{Name: Name}
	err = SaveData(&s21)
	assetBoolTest(t, err != nil, "Error save")
	CommitData()
	assetBoolTest(t, s21.ID == 0, "Error no ID")

	s22 := SchoolSimply3{Name: Name, ParentID: s21.ID}
	err = SaveData(&s22)
	assetBoolTest(t, err != nil, "Error save")

	CommitData()

	cacheMultiKey := cacheTableSQL["*cacheDatabase.SchoolSimply3"]["ParentID,Name"]
	valeurcache, ok := cacheMultiKey.Get(fmt.Sprintf("%v||%v", s21.ParentID, s21.Name))
	assetBoolTest(t, !ok, "Key not found")
	assetBoolTest(t, valeurcache.(*SchoolSimply3).ID != 1, fmt.Sprintf("Bad ID: %v", valeurcache.(*SchoolSimply3).ID))

	valeurcache2, ok := cacheMultiKey.Get(fmt.Sprintf("%v||%v", s21.ID, s22.Name))
	assetBoolTest(t, !ok, "Key not found")
	assetBoolTest(t, valeurcache2.(*SchoolSimply3).ID != 2, fmt.Sprintf("Bad ID: %v", valeurcache2.(*SchoolSimply3).ID))

	//destroyed cache
	cacheTableSQL = make(map[string]cacheSQL)

	// Check cache multi-columns
	s4 := SchoolSimply3{}
	databaseGetWithCache(&s4, "ParentID,Name", fmt.Sprintf("%v", s21.ID), s22.Name)
	assetBoolTest(t, s4.ID != 2, fmt.Sprintf("N'utilise pas le bon ID: %v", s4.ID))

	_, ok = cacheTableSQL["*cacheDatabase.SchoolSimply3"]
	assetBoolTest(t, !ok, fmt.Sprintf("Le cache n'est pas implementé"))

	cacheMultiKey = cacheTableSQL["*cacheDatabase.SchoolSimply3"]["ParentID,Name"]
	valeurcache12, ok := cacheMultiKey.Get(fmt.Sprintf("%v||%v", s21.ID, s21.Name))
	assetBoolTest(t, !ok, fmt.Sprintf("Key not found"))
	assetBoolTest(t, valeurcache12.(*SchoolSimply3).ID != 2, fmt.Sprintf("Bad ID: %v", valeurcache12.(*SchoolSimply3).ID))

	conn, _ := DB.DB()
	conn.Close()
}

func TestList(t *testing.T) {

	DB = openBaseTest(t, "gorm_test2.db")
	DB.AutoMigrate(&SchoolSimply2{})

	// Save
	SaveData(&SchoolSimply2{Name: "Jean jaures"})
	SaveData(&SchoolSimply2{Name: "romain rolland"})
	SaveData(&SchoolSimply2{Name: "julio curie"})
	CommitData()

	// get All record from database
	var list []SchoolSimply2
	databaseGet(&list, "")

	if len(list) != 3 {
		t.Fatal("Il n'y pas le bon nombre element: ", len(list))
	}

	// Close database
	conn, _ := DB.DB()
	conn.Close()
}

type TestFile struct {
	FileTemplate
	Mode uint
}

func TestDir(t *testing.T) {
	// Create file database
	DB = openBaseTest(t, "gorm_dir.db")
	DB.AutoMigrate(&TestFile{})

	// Set Root
	createRecordFileTest(t, 1, 1, "", true)

	// Create path
	createRecordFileTest(t, 0, 1, "tmp", true)
	createRecordFileTest(t, 0, 2, "dataCache", true)
	createRecordFileTest(t, 0, 3, "Welcome", true)
	createRecordFileTest(t, 0, 4, "Me", true)

	// Create path Test FAILED
	createRecordFileTest(t, 0, 999999, "Me", false)

	// Active debug
	DEBUG = true

	// Get WElcome
	fi, err := GetPath(&TestFile{}, "/tmp/dataCache/Welcome/Me")
	assetBoolTest(t, err != nil, fmt.Sprintf("Error: %v", err))
	assetBoolTest(t, fi.(*TestFile).ID != 5, fmt.Sprintf("Mauvais ID retourné: %v", err))
	assetBoolTest(t, fi.(*TestFile).Path != "/tmp/dataCache/Welcome/Me", fmt.Sprintf("Mauvais pathParent: %v", fi.(*TestFile).Path))
	assetBoolTest(t, fi.(*TestFile).Name != "Me", fmt.Sprintf("Mauvais Name: %v", fi.(*TestFile).Name))

	// Get WElcome failed
	_, err = GetPath(&TestFile{}, "/tmp/dataCache2/Welcome/Me")
	assetBoolTest(t, err != ErrRecordNotFound, fmt.Sprintf("Error: %v", err))

	_, err = GetPath(&TestFile{}, "/tmp/dataCache/Welcome/Not found")
	assetBoolTest(t, err != ErrRecordNotFound, fmt.Sprintf("Error: %v", err))

	fmt.Println("=DISPLAY Table Cache========================")
	//Check Cache
	for nom, val := range cacheTableSQL {
		fmt.Println(nom, val)
	}
	for nom, val := range cacheTableSQL["*cacheDatabase.TestFile"] {
		fmt.Println(nom, val)
	}

	fmt.Println("Display Cache TestFile into ID index")
	for nom, val := range cacheTableSQL["*cacheDatabase.TestFile"]["Path"].Items() {
		fmt.Println(nom, val.Object.(*TestFile))
	}
	// Close database
	conn, _ := DB.DB()
	conn.Close()

	return
}
func TestDirSave(t *testing.T) {

	DEBUG = false

	// Create file database
	DB = openBaseTest(t, "gorm_dir.db")
	DB.AutoMigrate(&TestFile{})
	DB = DB.Debug()

	// Set Path
	createRecordFileTest(t, 1, 1, "", true)
	createRecordFileTest(t, 0, 1, "tmp", true)

	var f TestFile = TestFile{}
	f.FileTemplate.Path = "/tmp/gorm/test/a.txt"
	_, err := SavePath(&f)
	if err != nil {
		t.Fatal(err)
	}

	CommitData()

	DisplayCache(cacheTableSQL["*cacheDatabase.TestFile"])

	// Check record

	// Add new record
	var f2 TestFile = TestFile{}
	f2.FileTemplate.Path = "/tmp/cachedatabase/test/a.txt"
	_, err = SavePath(&f2)
	if err != nil {
		t.Fatal(err)
	}
	CommitData()

	// Close database
	conn, _ := DB.DB()
	conn.Close()

	/*
		var fileTemplate FileTemplate

		f, err := GetPath(&TestFile{}, "/tmp/pake/Ere/a")
		if err != gorm.ErrRecordNotFound {
			t.Fatal("Error:", err)
		}

		fmt.Println("ll", f, err)
		fmt.Println("================================================")

		// Save file
		_, err = SavePath(f)
		if err != gorm.ErrRecordNotFound {
			t.Fatal("Error:", err)
		}
		if err != nil {
			t.Fatal("Error Save dir:", err)
		}

		CommitData()

		fmt.Println(f)
	*/
}
func openBaseTest(t *testing.T, path string) *gorm.DB {
	os.Remove(path)
	if _, err := os.Stat(path); os.IsExist(err) {
		t.Fatal("Base impossible à detruire: ", err)
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if res := db.Exec("PRAGMA foreign_keys = ON", nil); res.Error != nil {
		t.Fatal("Impossible create foreign")
	}
	return db
}
func createRecordFileTest(t *testing.T, id, parentID uint, name string, checkError bool) {
	file := TestFile{}
	file.ID = id
	file.ParentID = parentID
	file.Name = name
	ret := DB.Save(&file)
	if checkError {
		if ret.Error != nil {
			t.Fatal("Error: Ca ne nous retourne pas nil alors qu'il est Impossible de créer l'enregistrement ", file, " ", ret.Error)
		}
	} else {
		if ret.Error == nil {
			t.Fatal("Error: Ca nous retourne  nil alors qu'il est Impossible de créer l'enregistrement :", file)
		}
	}
}

func assetBoolTest(t *testing.T, test bool, message string) {
	if test {
		t.Fatal(message)
	}
}
