package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type adder interface {
	add(string, string)
}

type account struct {
	username string
	password string
}

type users []account

type usersHolder struct {
	users
	current int
	getMut  sync.Mutex
}

func createUsersHolder(filePath string) *usersHolder {
	var u usersHolder
	readConfFile(filePath, &u)
	return &u
}

func (u *usersHolder) GetNext() *account {
	u.getMut.Lock()
	if u.current >= len(u.users) {
		u.current = 0
	}
	defer func() {
		u.current++
		u.getMut.Unlock()
	}()
	return &u.users[u.current]
}

func (u *usersHolder) GetRandom() *account {
	u.getMut.Lock()
	defer u.getMut.Unlock()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &u.users[r.Int31n(int32(len(u.users)))]
}

func (u *usersHolder) findPassword(username string) (string, error) {
	for _, a := range u.users {
		if a.username == username {
			return a.password, nil
		}
	}
	return "", fmt.Errorf("Username: %s not found", username)
}

func (u *usersHolder) add(username string, password string) {
	if len(username) == 0 && len(password) == 0 {
		return
	}

	u.users = append(u.users, account{username, password})
}
