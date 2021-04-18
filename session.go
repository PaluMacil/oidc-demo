package main

import (
	"github.com/google/uuid"
	"sync"
)

type Session struct {
	ID string
	Authenticated bool
	GoogleClaims
}

type SessionStore struct {
	lock sync.RWMutex
	sessions map[string]Session
}

func (store *SessionStore) assureSessionMap() {
	if store.sessions == nil {
		store.sessions = make(map[string]Session)
	}
}

func (store *SessionStore) Get(id string) Session {
	store.lock.RLock()
	defer store.lock.RUnlock()

	return store.sessions[id]
}

func (store *SessionStore) Set(claims GoogleClaims) Session {
	store.lock.Lock()
	defer store.lock.Unlock()

	store.assureSessionMap()

	id := uuid.NewString()
	session := Session{
		ID:            id,
		Authenticated: true,
		GoogleClaims:  claims,
	}
	store.sessions[id] = session

	return session
}

func (store *SessionStore) Delete(id string) {
	store.lock.Lock()
	defer store.lock.Unlock()

	store.assureSessionMap()

	delete(store.sessions, id)
}