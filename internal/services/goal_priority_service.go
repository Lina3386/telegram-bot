package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
)

func (s *FinanceService) SwapGoalPriorities(ctx context.Context, telegramID int64, goalID int64, newPriority int) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return fmt.Errorf("goal not found: %w", err)
	}

	if goal.UserID != user.ID {
		return fmt.Errorf("goal does not belong to user")
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return err
	}

	if newPriority < 1 || newPriority > len(goals) {
		return fmt.Errorf("invalid priority: must be between 1 and %d", len(goals))
	}

	var targetGoal *models.SavingsGoal
	for i := range goals {
		if goals[i].Priority == newPriority && goals[i].ID != goal.ID {
			targetGoal = &goals[i]
			break
		}
	}

	oldPriority := goal.Priority
	if targetGoal != nil {
		targetGoal.Priority = oldPriority
		err = s.goalRepo.UpdateGoal(ctx, targetGoal)
		if err != nil {
			return fmt.Errorf("failed to update target goal: %w", err)
		}
	}
	goal.Priority = newPriority

	if err := s.goalRepo.UpdateGoal(ctx, goal); err != nil {
		return fmt.Errorf("failed to update goal: %w", err)
	}

	log.Printf("Swapped priorities: goal %d (priority %d <-> %d)", goalID, oldPriority, newPriority)

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	return err
}

func (s *FinanceService) GetGoalPriorityInfo(ctx context.Context, telegramID int64) ([]map[string]interface{}, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(goals)-1; i++ {
		for j := i + 1; j < len(goals); j++ {
			if goals[i].Priority > goals[j].Priority {
				goals[i], goals[j] = goals[j], goals[i]
			}
		}
	}

	result := make([]map[string]interface{}, len(goals))
	for i, goal := range goals {
		result[i] = map[string]interface{}{
			"id":       goal.ID,
			"name":     goal.GoalName,
			"priority": goal.Priority,
			"target":   goal.TargetAmount,
			"current":  goal.CurrentAmount,
			"monthly":  goal.MonthlyContrib,
		}
	}

	return result, nil
}

func (s *FinanceService) CalculateMonthlyBudgetDistribution(ctx context.Context, telegramID int64) (map[int64]int64, int64, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, 0, fmt.Errorf("user not found: %w", err)
	}

	availableForSavings, err := s.CalculateAvailableForSavings(ctx, telegramID)
	if err != nil {
		return nil, 0, err
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return nil, 0, err
	}

	if len(goals) == 0 {
		return make(map[int64]int64), 0, nil
	}

	for i := 0; i < len(goals)-1; i++ {
		for j := i + 1; j < len(goals); j++ {
			if goals[i].Priority > goals[j].Priority {
				goals[i], goals[j] = goals[j], goals[i]
			}
		}
	}

	distribution := make(map[int64]int64)
	n := int64(len(goals))
	summaryFactorial := (n * (n + 1)) / 2

	for _, goal := range goals {
		if goal.Status == "active" {
			priorityWeight := n - int64(goal.Priority) + 1
			budgetShare := (availableForSavings * priorityWeight) / summaryFactorial
			distribution[goal.ID] = budgetShare
		}
	}

	return distribution, availableForSavings, nil
}

func (s *FinanceService) ResetMonthlyContributions(ctx context.Context, userID int64) error {
	_, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, userID)
	if err != nil {
		return err
	}

	todayDate := time.Now()
	monthStartDate := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.Local)

	for _, goal := range goals {
		goal.MonthStarted.Valid = true
		goal.MonthStarted.Time = monthStartDate
		goal.MonthlyAccumulated = 0

		if err := s.goalRepo.UpdateGoal(ctx, &goal); err != nil {
			log.Printf("Failed to reset monthly contributions for goal %d: %v", goal.ID, err)
		}
	}

	return nil
}

func (s *FinanceService) UpdateGoalMonthlyBudget(ctx context.Context, goalID int64, budgetAmount int64) error {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return err
	}

	goal.MonthlyBudgetLimit = budgetAmount
	goal.MonthlyAccumulated = 0

	todayDate := time.Now()
	monthStartDate := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.Local)
	goal.MonthStarted.Valid = true
	goal.MonthStarted.Time = monthStartDate

	return s.goalRepo.UpdateGoal(ctx, goal)
}

func (s *FinanceService) DistributeFundsToGoalsV2(ctx context.Context, telegramID int64) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	availableForSavings, err := s.CalculateAvailableForSavings(ctx, telegramID)
	if err != nil {
		return err
	}

	if availableForSavings <= 0 {
		goals, _ := s.goalRepo.GetUserGoals(ctx, user.ID)
		for i := range goals {
			goals[i].MonthlyContrib = 0
			s.goalRepo.UpdateGoal(ctx, &goals[i])
		}
		return nil
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return err
	}

	if len(goals) == 0 {
		return nil
	}

	for i := 0; i < len(goals)-1; i++ {
		for j := i + 1; j < len(goals); j++ {
			if goals[i].Priority > goals[j].Priority {
				goals[i], goals[j] = goals[j], goals[i]
			}
		}
	}

	n := int64(len(goals))
	summaryFactorial := (n * (n + 1)) / 2
	if summaryFactorial == 0 {
		summaryFactorial = 1
	}

	allocated := make([]int64, len(goals))
	totalAllocated := int64(0)

	log.Printf("[DISTRIBUTE] Starting with availableForSavings=%d₽ for %d goals", availableForSavings, n)

	weights := make([]int64, len(goals))
	initialAllocated := make([]int64, len(goals))

	for i, goal := range goals {
		weights[i] = n - int64(goal.Priority) + 1
		initialAllocated[i] = (availableForSavings * weights[i]) / summaryFactorial
	}

	copy(allocated, initialAllocated)

	for i, goal := range goals {
		remainingToTarget := goal.TargetAmount - goal.CurrentAmount
		if remainingToTarget < 0 {
			remainingToTarget = 0
		}

		if allocated[i] > remainingToTarget {
			excess := allocated[i] - remainingToTarget
			allocated[i] = remainingToTarget
			totalAllocated += allocated[i]

			log.Printf("[DISTRIBUTE] Goal %d (%s) excess: %d₽, allocated reduced to %d₽", goal.ID, goal.GoalName, excess, remainingToTarget)

			sumLowerWeights := int64(0)
			for j := i + 1; j < len(goals); j++ {
				sumLowerWeights += weights[j]
			}

			if sumLowerWeights > 0 {
				for j := i + 1; j < len(goals); j++ {
					added := (excess * weights[j]) / sumLowerWeights
					allocated[j] += added
					log.Printf("[DISTRIBUTE] Goal %d (%s): +%d₽ from excess, new allocated %d₽", goals[j].ID, goals[j].GoalName, added, allocated[j])
				}
			}
		} else {
			totalAllocated += allocated[i]
		}
	}

	log.Printf("[DISTRIBUTE] Final allocation total: %d₽ (available: %d₽)", totalAllocated, availableForSavings)

	for i := range goals {
		if goals[i].Status != "active" {
			continue
		}

		contrib := allocated[i]
		if contrib == 0 {
			contrib = 1
		}

		goals[i].MonthlyContrib = contrib
		goals[i].MonthlyBudgetLimit = contrib

		currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

		if !goals[i].MonthStarted.Valid || goals[i].MonthStarted.Time != currentMonth {
			log.Printf("[DISTRIBUTE] Month changed for goal %d, resetting accumulated", goals[i].ID)
			goals[i].MonthStarted.Valid = true
			goals[i].MonthStarted.Time = currentMonth

			monthlyContribRecord, _ := s.monthlyContribRepo.GetContributionByUserGoalMonth(ctx, goals[i].UserID, goals[i].ID, currentMonth)
			if monthlyContribRecord == nil || monthlyContribRecord.AmountContributed <= 0 {
				goals[i].MonthlyAccumulated = 0
				log.Printf("[DISTRIBUTE] No monthly contributions found for goal %d, resetting to 0", goals[i].ID)
			} else {
				goals[i].MonthlyAccumulated = monthlyContribRecord.AmountContributed
				log.Printf("[DISTRIBUTE] Found monthly contributions for goal %d, setting MonthlyAccumulated=%d", goals[i].ID, goals[i].MonthlyAccumulated)
			}
		}

		remaining := goals[i].TargetAmount - goals[i].CurrentAmount
		if remaining <= 0 {
			remaining = 0
		}

		monthsNeeded := (remaining / goals[i].MonthlyContrib)
		if remaining%goals[i].MonthlyContrib != 0 {
			monthsNeeded++
		}
		if monthsNeeded <= 0 {
			monthsNeeded = 1
		}

		goals[i].TargetDate = time.Now().AddDate(0, int(monthsNeeded), 0)

		err := s.goalRepo.UpdateGoal(ctx, &goals[i])
		if err != nil {
			log.Printf("Failed to update goal: %v", err)
		}
	}

	return nil
}

func (s *FinanceService) ContributeToGoalWithMonthlyTracking(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	log.Printf("[CONTRIBUTION] Adding %d to goal %d, current MonthlyAccumulated: %d", amount, goalID, goal.MonthlyAccumulated)

	todayDate := time.Now()
	monthStartDate := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	var goalMonthStart time.Time
	if goal.MonthStarted.Valid {
		goalMonthStart = time.Date(goal.MonthStarted.Time.Year(), goal.MonthStarted.Time.Month(), goal.MonthStarted.Time.Day(), 0, 0, 0, 0, time.UTC)
	}

	if !goal.MonthStarted.Valid || goalMonthStart != monthStartDate {
		log.Printf("[CONTRIBUTION] Month changed for goal %d, resetting MonthlyAccumulated", goalID)
		goal.MonthlyAccumulated = 0
		goal.MonthStarted.Valid = true
		goal.MonthStarted.Time = monthStartDate
	}

	goal.CurrentAmount += amount
	goal.MonthlyAccumulated += amount

	currentMonth := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthlyContribRecord, err := s.monthlyContribRepo.GetContributionByUserGoalMonth(ctx, goal.UserID, goalID, currentMonth)

	if err != nil || monthlyContribRecord == nil {
		_, createErr := s.monthlyContribRepo.CreateContribution(ctx, goal.UserID, goalID, currentMonth, goal.MonthlyAccumulated)
		if createErr != nil {
			log.Printf("[CONTRIBUTION] Failed to create monthly contribution: %v", createErr)
		}
	} else {
		monthlyContribRecord.AmountContributed = goal.MonthlyAccumulated
		if updateErr := s.monthlyContribRepo.UpdateContribution(ctx, monthlyContribRecord); updateErr != nil {
			log.Printf("[CONTRIBUTION] Failed to update monthly contribution: %v", updateErr)
		}
	}

	log.Printf("[CONTRIBUTION] Goal %d updated: CurrentAmount=%d, MonthlyAccumulated=%d", goalID, goal.CurrentAmount, goal.MonthlyAccumulated)

	if goal.MonthlyAccumulated > goal.MonthlyBudgetLimit && goal.MonthlyBudgetLimit > 0 {
		log.Printf("Goal %d exceeded monthly budget: accumulated %d, limit %d",
			goalID, goal.MonthlyAccumulated, goal.MonthlyBudgetLimit)
	}

	if goal.CurrentAmount >= goal.TargetAmount {
		goal.Status = "completed"
		goal.CurrentAmount = goal.TargetAmount
		log.Printf("Goal %d completed!", goalID)
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}
	log.Printf("[CONTRIBUTION] Goal %d saved successfully", goalID)

	goalUser, err := s.userRepo.GetUserByID(ctx, goal.UserID)
	if err == nil {
		log.Printf("[CONTRIBUTION] Redistributing funds for user %d", goalUser.TelegramID)
		s.DistributeFundsToGoalsV2(ctx, goalUser.TelegramID)
	}

	return goal, nil
}

func (s *FinanceService) GetGoalMonthlyStats(ctx context.Context, goalID int64) (map[string]interface{}, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

	monthlyContribRecord, err := s.monthlyContribRepo.GetContributionByUserGoalMonth(ctx, goal.UserID, goalID, currentMonth)
	monthlyAccumulated := int64(0)

	if err == nil && monthlyContribRecord != nil {
		monthlyAccumulated = monthlyContribRecord.AmountContributed
		log.Printf("[MONTHLY_STATS] Using monthly_contributions: goal %d, accumulated=%d", goalID, monthlyAccumulated)
	} else {
		monthlyAccumulated = goal.MonthlyAccumulated
		log.Printf("[MONTHLY_STATS] Using goal.MonthlyAccumulated: goal %d, accumulated=%d", goalID, monthlyAccumulated)
	}

	if goal.MonthlyAccumulated != monthlyAccumulated {
		goal.MonthlyAccumulated = monthlyAccumulated
		log.Printf("[MONTHLY_STATS] Syncing goal.MonthlyAccumulated for goal %d to %d", goalID, monthlyAccumulated)
		s.goalRepo.UpdateGoal(ctx, goal)
	}

	monthStartedStr := "NULL"
	if goal.MonthStarted.Valid {
		monthStartedStr = goal.MonthStarted.Time.Format("2006-01-02")
	}
	log.Printf("[DEBUG] GetGoalMonthlyStats for goal %d: MonthlyAccumulated=%d, MonthlyBudgetLimit=%d, MonthStarted=%s",
		goalID, goal.MonthlyAccumulated, goal.MonthlyBudgetLimit, monthStartedStr)

	todayDate := time.Now()
	monthStartDate := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	var goalMonthStart time.Time
	if goal.MonthStarted.Valid {
		goalMonthStart = time.Date(goal.MonthStarted.Time.Year(), goal.MonthStarted.Time.Month(), goal.MonthStarted.Time.Day(), 0, 0, 0, 0, time.UTC)
	}

	goalMonthStartStr := "NULL"
	if goal.MonthStarted.Valid {
		goalMonthStartStr = goalMonthStart.Format("2006-01-02")
	}
	log.Printf("[DEBUG] Current month start: %s, Goal month start: %s",
		monthStartDate.Format("2006-01-02"), goalMonthStartStr)

	if !goal.MonthStarted.Valid || goalMonthStart != monthStartDate {
		log.Printf("[DEBUG] Month changed for goal %d, resetting accumulated", goalID)

		goal.MonthStarted.Valid = true
		goal.MonthStarted.Time = monthStartDate

		goal.MonthlyAccumulated = 0

		err = s.goalRepo.UpdateGoal(ctx, goal)
		if err != nil {
			log.Printf("Failed to reset monthly accumulated for goal %d: %v", goalID, err)
		}

		goal, err = s.goalRepo.GetGoalByID(ctx, goalID)
		if err != nil {
			return nil, err
		}
		monthlyAccumulated = 0
		log.Printf("[DEBUG] After reset for goal %d: MonthlyAccumulated=%d", goalID, monthlyAccumulated)
	}

	monthlyBudgetLimit := goal.MonthlyBudgetLimit
	if monthlyBudgetLimit == 0 {
		monthlyBudgetLimit = goal.MonthlyContrib
		log.Printf("[DEBUG] Using MonthlyContrib as budget limit: %d", monthlyBudgetLimit)
	}

	remainingToTarget := goal.TargetAmount - goal.CurrentAmount
	if remainingToTarget < 0 {
		remainingToTarget = 0
	}

	displayMonthlyBudgetLimit := monthlyBudgetLimit
	if remainingToTarget > 0 && remainingToTarget <= displayMonthlyBudgetLimit {
		displayMonthlyBudgetLimit = remainingToTarget
		log.Printf("[DEBUG] Goal %d (%s) can be closed within monthly limit %d₽, showing remaining %d₽",
			goal.ID, goal.GoalName, monthlyBudgetLimit, remainingToTarget)
	}

	remaining := monthlyBudgetLimit - monthlyAccumulated
	if remaining < 0 {
		remaining = 0
	}

	log.Printf("[DEBUG] Final stats for goal %d: accumulated=%d, budget_limit=%d, progress=%d",
		goalID, monthlyAccumulated, monthlyBudgetLimit,
		func() int64 {
			if monthlyBudgetLimit > 0 {
				return (monthlyAccumulated * 100) / monthlyBudgetLimit
			}
			return 0
		}())

	stats := map[string]interface{}{
		"goal_id":              goal.ID,
		"goal_name":            goal.GoalName,
		"priority":             goal.Priority,
		"target_amount":        goal.TargetAmount,
		"current_amount":       goal.CurrentAmount,
		"progress_percent":     0,
		"monthly_budget_limit": monthlyBudgetLimit,
		"monthly_accumulated":  monthlyAccumulated,
		"monthly_remaining":    remaining,
		"monthly_progress":     0,
		"target_date":          goal.TargetDate.Format("02.01.2006"),
	}

	if goal.TargetAmount > 0 {
		stats["progress_percent"] = (goal.CurrentAmount * 100) / goal.TargetAmount
	}

	if monthlyBudgetLimit > 0 {
		stats["monthly_progress"] = (monthlyAccumulated * 100) / monthlyBudgetLimit
	}

	return stats, nil
}
