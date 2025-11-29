package state

import (
	"sync"
)

// DialogState возможные состояния диалога
type DialogState string

const (
	StateIdle                DialogState = "idle"
	StateAddingIncome        DialogState = "adding_income"
	StateAddingIncomeAmount  DialogState = "adding_income_amount"
	StateAddingIncomeDay     DialogState = "adding_income_day"
	StateAddingExpense       DialogState = "adding_expense"
	StateAddingExpenseAmount DialogState = "adding_expense_amount"
	StateCreatingGoal        DialogState = "creating_goal"
	StateCreatingGoalTarget  DialogState = "creating_goal_target"
)

// UserSession хранит информацию о сессии пользователя
type UserSession struct {
	UserID      int64
	State       DialogState
	TempData    map[string]string
	LastMessage int64 // ID последнего сообщения для отслеживания
}

// StateManager управляет состояниями пользователей
type StateManager struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

// NewStateManager создает новый менеджер состояний
func NewStateManager() *StateManager {
	return &StateManager{
		sessions: make(map[int64]*UserSession),
	}
}

// GetSession получает или создает сессию пользователя
func (sm *StateManager) GetSession(userID int64) *UserSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		return session
	}

	// Создаем новую сессию
	session := &UserSession{
		UserID:   userID,
		State:    StateIdle,
		TempData: make(map[string]string),
	}
	sm.sessions[userID] = session
	return session
}

// SetState устанавливает состояние пользователя
func (sm *StateManager) SetState(userID int64, state DialogState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.State = state
	} else {
		sm.sessions[userID] = &UserSession{
			UserID:   userID,
			State:    state,
			TempData: make(map[string]string),
		}
	}
}

// GetState получает текущее состояние пользователя
func (sm *StateManager) GetState(userID int64) DialogState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if session, exists := sm.sessions[userID]; exists {
		return session.State
	}
	return StateIdle
}

// SetTempData сохраняет временные данные
func (sm *StateManager) SetTempData(userID int64, key, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.TempData[key] = value
	} else {
		sm.sessions[userID] = &UserSession{
			UserID:   userID,
			State:    StateIdle,
			TempData: map[string]string{key: value},
		}
	}
}

// GetTempData получает временные данные
func (sm *StateManager) GetTempData(userID int64, key string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if session, exists := sm.sessions[userID]; exists {
		return session.TempData[key]
	}
	return ""
}

// ClearSession очищает сессию пользователя
func (sm *StateManager) ClearSession(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, userID)
}

// ClearState очищает состояние и временные данные, но сохраняет сессию
func (sm *StateManager) ClearState(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.State = StateIdle
		session.TempData = make(map[string]string)
	}
}
