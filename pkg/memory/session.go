package memory

import "time"

// Session holds conversation history
type Session struct {
	ID        string
	UserID    string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	Role    string
	Content string
}

// Store manages conversation sessions
type Store struct {
	sessions map[string]*Session
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]*Session),
	}
}

func (s *Store) Get(userID string) *Session {
	return s.sessions[userID]
}

func (s *Store) Save(session *Session) {
	s.sessions[session.UserID] = session
}
