package session

import (
    "sync"
    "time"
)

// SiderSessionManager tracks upstream Sider conversation/message IDs in memory.
type SiderSessionManager struct {
    mu             sync.RWMutex
    sessions       map[string]*SiderSessionState
    maxAge         time.Duration
    continuousCID  string
}

// SiderSessionState mirrors TS session shape.
type SiderSessionState struct {
    CID              string
    UserMessageID    string
    AssistantMessageID string
    Model            string
    CreatedAt        time.Time
    LastActivity     time.Time
    MessageCount     int
}

// NewSiderSessionManager constructs a manager with maxAge and continuousCID hint.
func NewSiderSessionManager(maxAge time.Duration, continuousCID string) *SiderSessionManager {
    return &SiderSessionManager{
        sessions:      make(map[string]*SiderSessionState),
        maxAge:        maxAge,
        continuousCID: continuousCID,
    }
}

// Save stores or updates a session.
func (m *SiderSessionManager) Save(cid, userMsgID, assistantMsgID, model string) *SiderSessionState {
    m.mu.Lock()
    defer m.mu.Unlock()
    now := time.Now()
    s, ok := m.sessions[cid]
    if !ok {
        s = &SiderSessionState{CID: cid, CreatedAt: now}
        m.sessions[cid] = s
    }
    s.UserMessageID = userMsgID
    s.AssistantMessageID = assistantMsgID
    s.Model = model
    s.LastActivity = now
    s.MessageCount++
    return s
}

// Get returns a session by CID.
func (m *SiderSessionManager) Get(cid string) (*SiderSessionState, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    s, ok := m.sessions[cid]
    return s, ok
}

// NextParentMessageID returns assistant message id for given CID.
func (m *SiderSessionManager) NextParentMessageID(cid string) string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if s, ok := m.sessions[cid]; ok {
        return s.AssistantMessageID
    }
    return ""
}

// IsContinuous reports if cid matches the special continuous conversation id.
func (m *SiderSessionManager) IsContinuous(cid string) bool {
    return cid == m.continuousCID
}

// GetOrCreateContinuous returns the continuous conversation session state.
func (m *SiderSessionManager) GetOrCreateContinuous(model string) *SiderSessionState {
    m.mu.Lock()
    defer m.mu.Unlock()
    s, ok := m.sessions[m.continuousCID]
    if !ok {
        s = &SiderSessionState{CID: m.continuousCID, Model: model, CreatedAt: time.Now()}
        m.sessions[m.continuousCID] = s
    }
    s.LastActivity = time.Now()
    return s
}

// Cleanup removes expired sessions and returns count.
func (m *SiderSessionManager) Cleanup() int {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.maxAge <= 0 {
        return 0
    }
    now := time.Now()
    removed := 0
    for cid, s := range m.sessions {
        if now.Sub(s.LastActivity) > m.maxAge {
            delete(m.sessions, cid)
            removed++
        }
    }
    return removed
}

// Stats returns a lightweight snapshot for diagnostics.
func (m *SiderSessionManager) Stats() []SiderSessionState {
    m.mu.RLock()
    defer m.mu.RUnlock()
    out := make([]SiderSessionState, 0, len(m.sessions))
    for _, s := range m.sessions {
        out = append(out, *s)
    }
    return out
}
