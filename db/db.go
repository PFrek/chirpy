package db

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

const DB_PATH = "database.json"

type Chirp struct {
	Id       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

type ChirpFilter struct {
	AuthorId *int
	Contains *string
}

func (filters ChirpFilter) testAuthorId(id int) bool {
	if filters.AuthorId == nil {
		return true
	}

	return id == *filters.AuthorId
}

func (filters ChirpFilter) testBodyContains(body string) bool {
	if filters.Contains == nil {
		return true
	}

	return strings.Contains(body, *filters.Contains)
}

type ChirpSorter struct {
	Order *string
}

func (sorter ChirpSorter) sort(a, b Chirp) int {
	if sorter.Order == nil || *sorter.Order != "desc" {
		return cmp.Compare(a.Id, b.Id)
	}

	return cmp.Compare(b.Id, a.Id)
}

func (sorter ChirpSorter) sortChirps(chirps []Chirp) {
	slices.SortFunc(chirps, sorter.sort)
}

type User struct {
	Id          int    `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
}

type RefreshToken struct {
	Token     string
	ExpiresAt time.Time
	Id        int
}

type DBStructure struct {
	Chirps        map[int]Chirp `json:"chirps"`
	Users         map[int]User  `json:"users"`
	RefreshTokens map[string]RefreshToken
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
		return nil, fmt.Errorf("Unable to create new DB: %v", err)
	}

	return &db, nil
}

func (db *DB) ensureDB() error {
	_, err := os.ReadFile(db.path)
	if errors.Is(err, os.ErrNotExist) {
		dbStruct := DBStructure{
			Chirps:        make(map[int]Chirp),
			Users:         make(map[int]User),
			RefreshTokens: make(map[string]RefreshToken),
		}

		json, err := json.Marshal(dbStruct)
		if err != nil {
			return fmt.Errorf("Failed to marshal new db's contents: %v", err)
		}

		err = os.WriteFile(db.path, json, 0644)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Failed to load/create db: %v", err)
	}

	return nil
}

func (db DBStructure) getNextChirpId() int {
	maxId := 0
	for _, chirp := range db.Chirps {
		if chirp.Id > maxId {
			maxId = chirp.Id
		}
	}

	return maxId + 1
}

func (db DBStructure) getNextUserId() int {
	maxId := 0
	for _, user := range db.Users {
		if user.Id > maxId {
			maxId = user.Id
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
		return nil, fmt.Errorf("DB: Failed to load db: %v", err)
	}

	var dbStruct DBStructure
	err = json.Unmarshal(file, &dbStruct)
	if err != nil {
		return nil, fmt.Errorf("DB: Failed to unmarshal db: %v", err)
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
		return fmt.Errorf("DB: Failed to marshal db: %v", err)
	}

	err = os.WriteFile(db.path, json, 0644)
	if err != nil {
		return fmt.Errorf("DB: Failed to write to file: %v", err)
	}

	return nil
}

// REFRESH TOKENS

func (db *DB) CreateRefreshToken(token string, id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return err
	}

	dbStruct.RefreshTokens[token] = RefreshToken{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(60 * 24 * time.Hour),
		Id:        id,
	}

	err = db.writeDB(*dbStruct)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) ValidateRefreshToken(token string) (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return 0, err
	}

	refreshToken, ok := dbStruct.RefreshTokens[token]
	if !ok {
		return 0, errors.New("Refresh token not found")
	}

	if refreshToken.ExpiresAt.Before(time.Now().UTC()) {
		return 0, errors.New("Refresh token expired")
	}

	return refreshToken.Id, nil
}

func (db *DB) RevokeRefreshToken(token string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return err
	}

	delete(dbStruct.RefreshTokens, token)

	err = db.writeDB(*dbStruct)
	if err != nil {
		return err
	}

	return nil
}

// CHIRPS

func (db *DB) CreateChirp(body string, authorId int) (Chirp, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	chirp := Chirp{
		Id:       dbStruct.getNextChirpId(),
		Body:     body,
		AuthorId: authorId,
	}

	dbStruct.Chirps[chirp.Id] = chirp

	err = db.writeDB(*dbStruct)
	if err != nil {
		return Chirp{}, err
	}

	return chirp, nil
}

func (db *DB) GetChirps(filters ChirpFilter, sorter ChirpSorter) ([]Chirp, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	chirps := []Chirp{}
	for _, chirp := range dbStruct.Chirps {
		match := filters.testAuthorId(chirp.AuthorId)
		match = match && filters.testBodyContains(chirp.Body)

		if match {
			chirps = append(chirps, chirp)
		}
	}

	sorter.sortChirps(chirps)

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
		return Chirp{}, NotFoundError{Model: "Chirp"}
	}

	return *chirp, nil
}

func (db *DB) DeleteChirp(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return err
	}

	delete(dbStruct.Chirps, id)

	err = db.writeDB(*dbStruct)
	if err != nil {
		return err
	}

	return nil
}

// USERS

func (db *DB) CreateUser(email string, password string) (User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for _, user := range dbStruct.Users {
		if user.Email == email {
			return User{}, ExistingEmailError{}
		}
	}

	user := User{
		Id:          dbStruct.getNextUserId(),
		Email:       email,
		Password:    password,
		IsChirpyRed: false,
	}

	dbStruct.Users[user.Id] = user

	err = db.writeDB(*dbStruct)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (db *DB) UpdateUser(user User) (User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	var existingUser *User
	for _, u := range dbStruct.Users {
		if u.Id == user.Id {
			existingUser = new(User)
			existingUser = &u
		} else {
			if u.Email == user.Email {
				return User{}, ExistingEmailError{}
			}
		}
	}

	if existingUser == nil {
		return User{}, NotFoundError{"User"}
	}

	dbStruct.Users[user.Id] = user

	err = db.writeDB(*dbStruct)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (db *DB) UpgradeUser(id int) (User, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	existingUser, ok := dbStruct.Users[id]
	if !ok {
		return User{}, NotFoundError{"User"}
	}

	existingUser.IsChirpyRed = true
	dbStruct.Users[id] = existingUser

	err = db.writeDB(*dbStruct)
	if err != nil {
		return User{}, err
	}

	return existingUser, nil
}

func (db *DB) GetUsers() ([]User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return []User{}, err
	}

	users := []User{}
	for _, user := range dbStruct.Users {
		users = append(users, user)
	}

	slices.SortFunc(users, func(a, b User) int {
		return cmp.Compare(a.Id, b.Id)
	})

	return users, nil
}

func (db *DB) GetUserById(id int) (User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	var user *User
	for _, u := range dbStruct.Users {
		if u.Id == id {
			user = &u
		}
	}

	if user == nil {
		return User{}, NotFoundError{Model: "User"}
	}

	return *user, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	dbStruct, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	var user *User
	for _, u := range dbStruct.Users {
		if u.Email == email {
			user = &u
		}
	}

	if user == nil {
		return User{}, NotFoundError{Model: "User"}
	}

	return *user, nil
}

type ExistingEmailError struct{}

func (err ExistingEmailError) Error() string {
	return fmt.Sprintf("Email already in use")
}

type NotFoundError struct {
	Model string
}

func (err NotFoundError) Error() string {
	return fmt.Sprintf("DB Error: %s not found", err.Model)
}
