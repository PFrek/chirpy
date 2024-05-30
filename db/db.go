package db

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
)

const DB_PATH = "database.json"

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

type DB struct {
	path string
	mu   *sync.RWMutex
}

func NewDB(path string) (*DB, error) {
	db := DB{
		path: path,
		mu:   &sync.RWMutex{},
	}

	err := db.ensureDB()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to create new DB: %v", err))
	}

	return &db, nil
}

func (db *DB) ensureDB() error {
	_, err := os.ReadFile(db.path)
	if errors.Is(err, os.ErrNotExist) {
		err = os.WriteFile(db.path, []byte("{\"chirps\":{}}"), 0644)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New(fmt.Sprintf("Failed to load/create db: %v", err))
	}

	return nil
}

func (db DBStructure) getNextId() int {
	maxId := 0
	for _, chirp := range db.Chirps {
		if chirp.Id > maxId {
			maxId = chirp.Id
		}
	}

	return maxId + 1
}

func (db *DB) loadDB() (*DBStructure, error) {
	err := db.ensureDB()
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(db.path)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("DB: Failed to load db: %v", err))
	}

	var dbStruct DBStructure
	err = json.Unmarshal(file, &dbStruct)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("DB: Failed to unmarshal db: %v", err))
	}
	if db == nil {
		return nil, errors.New("DB: Failed to unmarshal db: Empty")
	}

	return &dbStruct, nil
}

func (db *DB) writeDB(dbStruct DBStructure) error {
	err := db.ensureDB()
	if err != nil {
		return err
	}

	json, err := json.Marshal(dbStruct)
	if err != nil {
		return errors.New(fmt.Sprintf("DB: Failed to marshal db: %v", err))
	}

	err = os.WriteFile(db.path, json, 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("DB: Failed to write to file: %v", err))
	}

	return nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	chirp := Chirp{
		Id:   dbStruct.getNextId(),
		Body: body,
	}

	dbStruct.Chirps[chirp.Id] = chirp

	err = db.writeDB(*dbStruct)
	if err != nil {
		return Chirp{}, err
	}

	return chirp, nil
}

func (db *DB) GetChirps() ([]Chirp, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	chirps := []Chirp{}
	for _, chirp := range dbStruct.Chirps {
		chirps = append(chirps, chirp)
	}

	slices.SortFunc(chirps, func(a, b Chirp) int {
		return cmp.Compare(a.Id, b.Id)
	})

	return chirps, nil
}

func (db *DB) GetChirpById(id int) (Chirp, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	var chirp *Chirp
	for _, c := range dbStruct.Chirps {
		if c.Id == id {
			chirp = &c
		}
	}

	if chirp == nil {
		return Chirp{}, errors.New("not found")
	}

	return *chirp, nil
}
