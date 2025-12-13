package state

import (
	"sync"
)

type DialogState string

const (
	StateChangingGoalPriority  DialogState = "changing_goal_priority"
	StatePaydayEnteringAmount              = "payday_entering_amount"
	StateIdle                  DialogState = "idle"
	StateAddingIncome          DialogState = "adding_income"
	StateAddingIncomeAmount    DialogState = "adding_income_amount"
	StateAddingIncomeFrequency DialogState = "adding_income_frequency"
	StateAddingIncomeDay       DialogState = "adding_income_day"
	StateAddingIncomeHour      DialogState = "adding_income_hour"
	StateAddingExpense         DialogState = "adding_expense"
	StateAddingExpenseAmount   DialogState = "adding_expense_amount"
	StateAddingContribution    DialogState = "adding_contribution"
	StateCreatingGoal          DialogState = "creating_goal"
	StateCreatingGoalTarget    DialogState = "creating_goal_target"
	StateWithdrawingFromGoal   DialogState = "withdrawing_from_goal"
)

type UserSession struct {
	UserID   int64
	State    DialogState
	TempData map[string]string
	mu       sync.RWMutex
}

type StateManager struct {
	sessions map[int64]*UserSession
	mu       sync.RWMutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		sessions: make(map[int64]*UserSession),
	}
}

func (sm *StateManager) GetSession(userID int64) *UserSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		return session
	}

	session := &UserSession{
		UserID:   userID,
		State:    StateIdle,
		TempData: make(map[string]string),
	}
	sm.sessions[userID] = session
	return session
}

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

func (sm *StateManager) GetState(userID int64) DialogState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if session, exists := sm.sessions[userID]; exists {
		return session.State
	}
	return StateIdle
}

func (sm *StateManager) SetTempData(userID int64, key, value string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.mu.Lock()
		defer session.mu.Unlock()
		session.TempData[key] = value
	} else {
		session := &UserSession{
			UserID:   userID,
			State:    StateIdle,
			TempData: map[string]string{key: value},
		}
		sm.sessions[userID] = session
	}
}

func (sm *StateManager) GetTempData(userID int64, key string) string {
	sm.mu.RLock()
	session, exists := sm.sessions[userID]
	sm.mu.RUnlock()

	if !exists {
		return ""
	}

	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.TempData[key]
}

func (sm *StateManager) ClearSession(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, userID)
}

func (sm *StateManager) ClearState(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[userID]; exists {
		session.State = StateIdle
		session.TempData = make(map[string]string)
	}
}
