package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"vaultke-backend/internal/models"

	"github.com/google/uuid"
)

// PollsService handles poll-related business logic
type PollsService struct {
	db *sql.DB
}

// NewPollsService creates a new polls service
func NewPollsService(db *sql.DB) *PollsService {
	return &PollsService{db: db}
}

// CreatePoll creates a new poll
func (s *PollsService) CreatePoll(chamaID, createdBy string, req *models.CreatePollRequest) (*models.Poll, error) {
	// Validate that the user has permission to create polls
	if !s.canCreatePolls(createdBy, chamaID) {
		return nil, fmt.Errorf("user does not have permission to create polls")
	}

	// Validate end date is in the future
	if req.EndDate.Before(time.Now()) {
		return nil, fmt.Errorf("end date must be in the future")
	}

	// Generate unique ID
	pollID := uuid.New().String()
	now := time.Now()

	// Set defaults
	isAnonymous := true
	if req.IsAnonymous != nil {
		isAnonymous = *req.IsAnonymous
	}

	requiresMajority := true
	if req.RequiresMajority != nil {
		requiresMajority = *req.RequiresMajority
	}

	majorityPercentage := 50.0
	if req.MajorityPercentage != nil {
		majorityPercentage = *req.MajorityPercentage
	}

	// Get total eligible voters
	totalVoters, err := s.getTotalEligibleVoters(chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get eligible voters: %w", err)
	}

	poll := &models.Poll{
		ID:                  pollID,
		ChamaID:             chamaID,
		Title:               req.Title,
		Description:         req.Description,
		PollType:            req.PollType,
		CreatedBy:           createdBy,
		StartDate:           now,
		EndDate:             req.EndDate,
		Status:              models.PollStatusActive,
		IsAnonymous:         isAnonymous,
		RequiresMajority:    requiresMajority,
		MajorityPercentage:  majorityPercentage,
		TotalEligibleVoters: totalVoters,
		TotalVotesCast:      0,
		Metadata:            req.Metadata,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// Insert poll into database
	query := `
		INSERT INTO polls (
			id, chama_id, title, description, poll_type, created_by, start_date, end_date,
			status, is_anonymous, requires_majority, majority_percentage, total_eligible_voters,
			total_votes_cast, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(
		query,
		poll.ID, poll.ChamaID, poll.Title, poll.Description, poll.PollType,
		poll.CreatedBy, poll.StartDate, poll.EndDate, poll.Status, poll.IsAnonymous,
		poll.RequiresMajority, poll.MajorityPercentage, poll.TotalEligibleVoters,
		poll.TotalVotesCast, poll.Metadata, poll.CreatedAt, poll.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create poll: %w", err)
	}

	// Create poll options
	for i, optionReq := range req.Options {
		err = s.createPollOption(pollID, &optionReq, i)
		if err != nil {
			log.Printf("Warning: Failed to create poll option: %v", err)
		}
	}

	log.Printf("Created poll %s for chama %s by user %s", pollID, chamaID, createdBy)
	return poll, nil
}

// GetChamaPolls retrieves polls for a chama
func (s *PollsService) GetChamaPolls(chamaID string, limit, offset int) ([]models.PollWithDetails, error) {
	query := `
		SELECT p.id, p.chama_id, p.title, p.description, p.poll_type, p.created_by,
			   p.start_date, p.end_date, p.status, p.is_anonymous, p.requires_majority,
			   p.majority_percentage, p.total_eligible_voters, p.total_votes_cast,
			   p.result, p.result_declared_at, p.metadata, p.created_at, p.updated_at,
			   u.first_name, u.last_name
		FROM polls p
		JOIN users u ON p.created_by = u.id
		WHERE p.chama_id = ?
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama polls: %w", err)
	}
	defer rows.Close()

	var polls []models.PollWithDetails
	for rows.Next() {
		var poll models.PollWithDetails
		var firstName, lastName string

		err := rows.Scan(
			&poll.ID, &poll.ChamaID, &poll.Title, &poll.Description, &poll.PollType,
			&poll.CreatedBy, &poll.StartDate, &poll.EndDate, &poll.Status, &poll.IsAnonymous,
			&poll.RequiresMajority, &poll.MajorityPercentage, &poll.TotalEligibleVoters,
			&poll.TotalVotesCast, &poll.Result, &poll.ResultDeclaredAt, &poll.Metadata,
			&poll.CreatedAt, &poll.UpdatedAt, &firstName, &lastName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan poll: %w", err)
		}

		poll.CreatedByName = firstName + " " + lastName
		poll.TimeRemaining = poll.GetTimeRemaining()

		// Get poll options
		options, err := s.getPollOptions(poll.ID)
		if err != nil {
			log.Printf("Warning: Failed to get options for poll %s: %v", poll.ID, err)
		} else {
			poll.Options = options
		}

		polls = append(polls, poll)
	}

	return polls, nil
}

// CastVote casts a vote in a poll
func (s *PollsService) CastVote(pollID, voterID string, req *models.CastVoteRequest) error {
	// Get poll
	poll, err := s.getPollByID(pollID)
	if err != nil {
		return err
	}

	// Check if poll is active and can accept votes
	if !poll.CanVote() {
		return fmt.Errorf("poll is not accepting votes")
	}

	// Check if user is eligible to vote
	if !s.isEligibleToVote(voterID, poll.ChamaID) {
		return fmt.Errorf("user is not eligible to vote in this chama")
	}

	// Check if user has already voted (using hash for anonymity)
	voterHash := models.GenerateVoterHash(voterID, pollID)
	if s.hasUserVoted(pollID, voterHash) {
		return fmt.Errorf("user has already voted in this poll")
	}

	// Validate option exists
	if !s.isValidOption(pollID, req.OptionID) {
		return fmt.Errorf("invalid option selected")
	}

	// Create vote record
	voteID := uuid.New().String()
	now := time.Now()

	voteQuery := `
		INSERT INTO votes (id, poll_id, option_id, voter_hash, vote_timestamp, is_valid)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(voteQuery, voteID, pollID, req.OptionID, voterHash, now, true)
	if err != nil {
		return fmt.Errorf("failed to cast vote: %w", err)
	}

	// Update option vote count
	err = s.incrementOptionVoteCount(req.OptionID)
	if err != nil {
		log.Printf("Warning: Failed to update option vote count: %v", err)
	}

	// Update poll total votes cast
	err = s.incrementPollVoteCount(pollID)
	if err != nil {
		log.Printf("Warning: Failed to update poll vote count: %v", err)
	}

	// Check if result should be declared immediately
	options, err := s.getPollOptions(pollID)
	if err == nil && poll.ShouldDeclareResult(options) {
		result := poll.CalculateResult(options)
		err = s.declarePollResult(pollID, result)
		if err != nil {
			log.Printf("Warning: Failed to declare poll result: %v", err)
		} else {
			// If this is a role escalation poll, process the role change
			if poll.PollType == models.PollTypeRoleEscalation {
				err = s.ProcessRoleEscalationResult(pollID, result)
				if err != nil {
					log.Printf("Warning: Failed to process role escalation result: %v", err)
				}
			}
		}
	}

	log.Printf("Vote cast in poll %s by user %s", pollID, voterID)
	return nil
}

// GetPollDetails retrieves detailed information about a poll
func (s *PollsService) GetPollDetails(pollID, userID string) (*models.PollWithDetails, error) {
	poll, err := s.getPollByID(pollID)
	if err != nil {
		return nil, err
	}

	// Get creator name
	creatorName, err := s.getUserName(poll.CreatedBy)
	if err != nil {
		log.Printf("Warning: Failed to get creator name: %v", err)
		creatorName = "Unknown"
	}

	// Get poll options
	options, err := s.getPollOptions(pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to get poll options: %w", err)
	}

	// Check if user has voted
	voterHash := models.GenerateVoterHash(userID, pollID)
	userVoted := s.hasUserVoted(pollID, voterHash)

	// Check if user can vote
	userCanVote := poll.CanVote() && s.isEligibleToVote(userID, poll.ChamaID) && !userVoted

	pollDetails := &models.PollWithDetails{
		Poll:          *poll,
		CreatedByName: creatorName,
		Options:       options,
		UserVoted:     userVoted,
		UserCanVote:   userCanVote,
		TimeRemaining: poll.GetTimeRemaining(),
	}

	return pollDetails, nil
}

// CreateRoleEscalationPoll creates a poll for role escalation
func (s *PollsService) CreateRoleEscalationPoll(chamaID, requestedBy string, req *models.CreateRoleEscalationRequest) (*models.RoleEscalationRequest, error) {
	// Validate that the candidate exists and is a member
	if !s.isEligibleToVote(req.CandidateID, chamaID) {
		return nil, fmt.Errorf("candidate is not a member of this chama")
	}

	// Get current role of candidate
	currentRole, err := s.getMemberRole(req.CandidateID, chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate's current role: %w", err)
	}

	// Create role escalation request
	requestID := uuid.New().String()
	now := time.Now()

	escalationReq := &models.RoleEscalationRequest{
		ID:            requestID,
		ChamaID:       chamaID,
		CandidateID:   req.CandidateID,
		CurrentRole:   currentRole,
		RequestedRole: req.RequestedRole,
		RequestedBy:   requestedBy,
		Status:        "voting",
		Justification: req.Justification,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Get candidate name for poll title
	candidateName, err := s.getUserName(req.CandidateID)
	if err != nil {
		candidateName = "Unknown"
	}

	// Create poll for role escalation
	pollReq := &models.CreatePollRequest{
		Title:       fmt.Sprintf("Role Change: %s to %s", candidateName, req.RequestedRole),
		Description: req.Justification,
		PollType:    models.PollTypeRoleEscalation,
		EndDate:     time.Now().Add(7 * 24 * time.Hour), // 7 days voting period
		Options: []models.PollOptionRequest{
			{OptionText: "Approve"},
			{OptionText: "Reject"},
		},
	}

	poll, err := s.CreatePoll(chamaID, requestedBy, pollReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create role escalation poll: %w", err)
	}

	escalationReq.PollID = &poll.ID

	// Insert role escalation request
	query := `
		INSERT INTO Election / Voting_requests (
			id, chama_id, candidate_id, current_role, requested_role, requested_by,
			poll_id, status, justification, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(
		query,
		escalationReq.ID, escalationReq.ChamaID, escalationReq.CandidateID,
		escalationReq.CurrentRole, escalationReq.RequestedRole, escalationReq.RequestedBy,
		escalationReq.PollID, escalationReq.Status, escalationReq.Justification,
		escalationReq.CreatedAt, escalationReq.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create role escalation request: %w", err)
	}

	log.Printf("Created role escalation request %s for candidate %s", requestID, req.CandidateID)
	return escalationReq, nil
}

// Helper methods
func (s *PollsService) getPollByID(pollID string) (*models.Poll, error) {
	query := `
		SELECT id, chama_id, title, description, poll_type, created_by, start_date, end_date,
			   status, is_anonymous, requires_majority, majority_percentage, total_eligible_voters,
			   total_votes_cast, result, result_declared_at, metadata, created_at, updated_at
		FROM polls WHERE id = ?
	`

	var poll models.Poll
	err := s.db.QueryRow(query, pollID).Scan(
		&poll.ID, &poll.ChamaID, &poll.Title, &poll.Description, &poll.PollType,
		&poll.CreatedBy, &poll.StartDate, &poll.EndDate, &poll.Status, &poll.IsAnonymous,
		&poll.RequiresMajority, &poll.MajorityPercentage, &poll.TotalEligibleVoters,
		&poll.TotalVotesCast, &poll.Result, &poll.ResultDeclaredAt, &poll.Metadata,
		&poll.CreatedAt, &poll.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("poll not found")
		}
		return nil, fmt.Errorf("failed to get poll: %w", err)
	}

	return &poll, nil
}

func (s *PollsService) createPollOption(pollID string, req *models.PollOptionRequest, order int) error {
	optionID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO poll_options (id, poll_id, option_text, option_order, vote_count, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, optionID, pollID, req.OptionText, order, 0, req.Metadata, now)
	return err
}

func (s *PollsService) getPollOptions(pollID string) ([]models.PollOption, error) {
	query := `
		SELECT id, poll_id, option_text, option_order, vote_count, metadata, created_at
		FROM poll_options
		WHERE poll_id = ?
		ORDER BY option_order ASC
	`

	rows, err := s.db.Query(query, pollID)
	if err != nil {
		return nil, fmt.Errorf("failed to get poll options: %w", err)
	}
	defer rows.Close()

	var options []models.PollOption
	for rows.Next() {
		var option models.PollOption
		err := rows.Scan(
			&option.ID, &option.PollID, &option.OptionText, &option.OptionOrder,
			&option.VoteCount, &option.Metadata, &option.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan poll option: %w", err)
		}
		options = append(options, option)
	}

	return options, nil
}

func (s *PollsService) getTotalEligibleVoters(chamaID string) (int, error) {
	query := `SELECT COUNT(*) FROM chama_members WHERE chama_id = ? AND is_active = TRUE`
	var count int
	err := s.db.QueryRow(query, chamaID).Scan(&count)
	return count, err
}

func (s *PollsService) canCreatePolls(userID, chamaID string) bool {
	// Check if user is a chama member (any member can create general polls)
	query := `
		SELECT 1 FROM chama_members
		WHERE user_id = ? AND chama_id = ? AND is_active = TRUE
	`
	var exists int
	err := s.db.QueryRow(query, userID, chamaID).Scan(&exists)
	return err == nil
}

func (s *PollsService) isEligibleToVote(userID, chamaID string) bool {
	query := `
		SELECT 1 FROM chama_members
		WHERE user_id = ? AND chama_id = ? AND is_active = TRUE
	`
	var exists int
	err := s.db.QueryRow(query, userID, chamaID).Scan(&exists)
	return err == nil
}

func (s *PollsService) hasUserVoted(pollID, voterHash string) bool {
	query := `SELECT 1 FROM votes WHERE poll_id = ? AND voter_hash = ? AND is_valid = TRUE`
	var exists int
	err := s.db.QueryRow(query, pollID, voterHash).Scan(&exists)
	return err == nil
}

func (s *PollsService) isValidOption(pollID, optionID string) bool {
	query := `SELECT 1 FROM poll_options WHERE poll_id = ? AND id = ?`
	var exists int
	err := s.db.QueryRow(query, pollID, optionID).Scan(&exists)
	return err == nil
}

func (s *PollsService) incrementOptionVoteCount(optionID string) error {
	query := `UPDATE poll_options SET vote_count = vote_count + 1 WHERE id = ?`
	_, err := s.db.Exec(query, optionID)
	return err
}

func (s *PollsService) incrementPollVoteCount(pollID string) error {
	query := `UPDATE polls SET total_votes_cast = total_votes_cast + 1, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), pollID)
	return err
}

func (s *PollsService) declarePollResult(pollID string, result models.PollResult) error {
	now := time.Now()
	query := `
		UPDATE polls
		SET result = ?, result_declared_at = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query, result, now, models.PollStatusCompleted, now, pollID)
	return err
}

func (s *PollsService) getUserName(userID string) (string, error) {
	query := `SELECT first_name, last_name FROM users WHERE id = ?`
	var firstName, lastName string
	err := s.db.QueryRow(query, userID).Scan(&firstName, &lastName)
	if err != nil {
		return "", err
	}
	return firstName + " " + lastName, nil
}

func (s *PollsService) getMemberRole(userID, chamaID string) (string, error) {
	query := `SELECT role FROM chama_members WHERE user_id = ? AND chama_id = ? AND is_active = TRUE`
	var role string
	err := s.db.QueryRow(query, userID, chamaID).Scan(&role)
	return role, err
}

// ProcessRoleEscalationResult processes the result of a role escalation poll
func (s *PollsService) ProcessRoleEscalationResult(pollID string, result models.PollResult) error {
	// Get the role escalation request associated with this poll
	query := `
		SELECT id, chama_id, candidate_id, current_role, requested_role, requested_by
		FROM Election / Voting_requests
		WHERE poll_id = ?
	`

	var req models.RoleEscalationRequest
	err := s.db.QueryRow(query, pollID).Scan(
		&req.ID, &req.ChamaID, &req.CandidateID,
		&req.CurrentRole, &req.RequestedRole, &req.RequestedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to get role escalation request: %w", err)
	}

	now := time.Now()

	if result == models.PollResultPassed {
		// Role change approved - update member roles
		err = s.executeRoleChange(&req)
		if err != nil {
			return fmt.Errorf("failed to execute role change: %w", err)
		}

		// Update escalation request status
		updateQuery := `
			UPDATE Election / Voting_requests
			SET status = 'approved', updated_at = ?
			WHERE id = ?
		`
		_, err = s.db.Exec(updateQuery, now, req.ID)
		if err != nil {
			return fmt.Errorf("failed to update escalation request status: %w", err)
		}

		log.Printf("Role escalation approved: %s changed from %s to %s in chama %s",
			req.CandidateID, req.CurrentRole, req.RequestedRole, req.ChamaID)
	} else {
		// Role change rejected
		updateQuery := `
			UPDATE Election / Voting_requests
			SET status = 'rejected', updated_at = ?
			WHERE id = ?
		`
		_, err = s.db.Exec(updateQuery, now, req.ID)
		if err != nil {
			return fmt.Errorf("failed to update escalation request status: %w", err)
		}

		log.Printf("Role escalation rejected for %s in chama %s", req.CandidateID, req.ChamaID)
	}

	return nil
}

// executeRoleChange handles the actual role change process
func (s *PollsService) executeRoleChange(req *models.RoleEscalationRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// If someone else currently holds the requested role, demote them to member
	if req.RequestedRole != "member" {
		demoteQuery := `
			UPDATE chama_members
			SET role = 'member', updated_at = ?
			WHERE chama_id = ? AND role = ? AND user_id != ? AND is_active = TRUE
		`
		_, err = tx.Exec(demoteQuery, now, req.ChamaID, req.RequestedRole, req.CandidateID)
		if err != nil {
			return fmt.Errorf("failed to demote current role holder: %w", err)
		}
	}

	// Update the candidate's role
	promoteQuery := `
		UPDATE chama_members
		SET role = ?, updated_at = ?
		WHERE chama_id = ? AND user_id = ? AND is_active = TRUE
	`
	_, err = tx.Exec(promoteQuery, req.RequestedRole, now, req.ChamaID, req.CandidateID)
	if err != nil {
		return fmt.Errorf("failed to update candidate role: %w", err)
	}

	// Create role change log entry
	logQuery := `
		INSERT INTO role_change_logs (
			id, chama_id, user_id, old_role, new_role, changed_by, change_reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	logID := uuid.New().String()
	_, err = tx.Exec(logQuery, logID, req.ChamaID, req.CandidateID,
		req.CurrentRole, req.RequestedRole, req.RequestedBy,
		"Role escalation poll approved", now)
	if err != nil {
		return fmt.Errorf("failed to create role change log: %w", err)
	}

	return tx.Commit()
}

// GetChamaMembers retrieves all active members of a chama for role voting
func (s *PollsService) GetChamaMembers(chamaID string) ([]models.ChamaMemberInfo, error) {
	query := `
		SELECT cm.user_id, cm.role, u.first_name, u.last_name, u.email, u.phone
		FROM chama_members cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.chama_id = ? AND cm.is_active = TRUE
		ORDER BY u.first_name, u.last_name
	`

	rows, err := s.db.Query(query, chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama members: %w", err)
	}
	defer rows.Close()

	var members []models.ChamaMemberInfo
	for rows.Next() {
		var member models.ChamaMemberInfo
		err := rows.Scan(
			&member.UserID, &member.Role, &member.FirstName,
			&member.LastName, &member.Email, &member.Phone,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		member.FullName = member.FirstName + " " + member.LastName
		members = append(members, member)
	}

	return members, nil
}
