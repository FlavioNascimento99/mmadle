package httpapi

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/store"
)

// Infinity (survival) mode: each round hides one random fighter behind an
// opaque round id. Five lives per round: a miss costs one, a solve restores
// (the next round starts full), zero kills the round and reveals the answer.
// Lives and streaks are server-authoritative so account records stay
// meaningful; guests play the same rules with browser-kept bests.

// roundResponse opens a round. It carries no target data: only the opaque id.
type roundResponse struct {
	RoundID string `json:"round_id"`
	Lives   int    `json:"lives"`
	Pool    string `json:"pool"`
}

// answerCard reveals a dead round's fighter (name + photo only, no attributes).
type answerCard struct {
	Name        string  `json:"name"`
	PhotoURL    *string `json:"photo_url"`
	PhotoCredit *string `json:"photo_credit"`
}

// infiniteGuessResponse extends the standard guess outcome with round state.
// Answer is present only when the round just died; streak fields only for
// signed-in players (guests track their best in the browser).
type infiniteGuessResponse struct {
	domain.GuessOutcome
	LivesLeft int         `json:"lives_left"`
	Solved    bool        `json:"solved"`
	RoundOver bool        `json:"round_over"`
	Answer    *answerCard `json:"answer,omitempty"`
	Streak    *int        `json:"streak,omitempty"`
	Best      *int        `json:"best,omitempty"`
	NewBest   *bool       `json:"new_best,omitempty"`
}

type roundRequest struct {
	Pool string `json:"pool"`
}

// POST /api/infinite/rounds — open a survival round in a pool.
func (s *Server) handleCreateRound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.infiniteUnavailable(w) {
		return
	}
	var req roundRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	pool, ok := parsePool(w, req.Pool)
	if !ok {
		return
	}
	if !s.roundIPLimiter.Allow("rounds:ip:" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	ids, err := s.Store.GameFighterIDs(ctx, pool)
	if err != nil || len(ids) == 0 {
		s.Logger.Error("round target pool failed", "pool", pool)
		writeError(w, http.StatusInternalServerError, "game_unavailable", "could not start a round")
		return
	}
	pick, err := rand.Int(rand.Reader, big.NewInt(int64(len(ids))))
	if err != nil {
		s.Logger.Error("round target draw failed")
		writeError(w, http.StatusInternalServerError, "game_unavailable", "could not start a round")
		return
	}
	roundID, err := domain.NewRoundID()
	if err != nil {
		s.Logger.Error("round id generation failed")
		writeError(w, http.StatusInternalServerError, "game_unavailable", "could not start a round")
		return
	}
	var userID *int64
	if user, ok := s.currentUser(r); ok {
		userID = &user.ID
	}
	if err := s.Infinite.CreateRound(ctx, roundID, userID, pool, ids[pick.Int64()], s.Clock.Now().Add(24*time.Hour)); err != nil {
		s.Logger.Error("round creation failed")
		writeError(w, http.StatusInternalServerError, "game_unavailable", "could not start a round")
		return
	}
	writeJSON(w, http.StatusCreated, roundResponse{RoundID: roundID, Lives: domain.MaxLives, Pool: string(pool)})
}

type infiniteGuessRequest struct {
	RoundID   string `json:"round_id"`
	FighterID int    `json:"fighter_id"`
}

// POST /api/infinite/guess — evaluate one guess against a round's target.
func (s *Server) handleInfiniteGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
		return
	}
	if !requireJSONContentType(w, r) || s.infiniteUnavailable(w) {
		return
	}
	var req infiniteGuessRequest
	if !decodeStrict(w, r, &req) {
		return
	}
	if err := domain.ValidateRoundID(req.RoundID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_round", "unknown round")
		return
	}
	if req.FighterID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_fighter_id", "fighter_id must be a positive integer")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	round, err := s.Infinite.FindRound(ctx, req.RoundID, s.Clock.Now())
	if err != nil {
		if errors.Is(err, store.ErrNoRound) {
			writeError(w, http.StatusNotFound, "round_not_found", "unknown or expired round")
			return
		}
		s.Logger.Error("round lookup failed")
		writeError(w, http.StatusInternalServerError, "guess_failed", "could not evaluate guess")
		return
	}
	if round.Status != "playing" {
		writeError(w, http.StatusGone, "round_over", "this round already finished")
		return
	}

	guess, err := s.Store.FighterView(ctx, req.FighterID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "fighter_not_found", "unknown fighter id")
			return
		}
		s.Logger.Error("infinite guess lookup failed")
		writeError(w, http.StatusInternalServerError, "guess_failed", "could not evaluate guess")
		return
	}
	target, err := s.Store.FighterView(ctx, round.TargetID)
	if err != nil {
		s.Logger.Error("infinite target lookup failed")
		writeError(w, http.StatusInternalServerError, "game_unavailable", "round unavailable")
		return
	}
	outcome := domain.EvaluateGuess(target, guess, s.gameDate())

	updated, err := s.Infinite.ApplyRoundGuess(ctx, req.RoundID, outcome.Correct, s.Clock.Now())
	if err != nil {
		if errors.Is(err, store.ErrRoundOver) {
			writeError(w, http.StatusGone, "round_over", "this round already finished")
			return
		}
		if errors.Is(err, store.ErrNoRound) {
			writeError(w, http.StatusNotFound, "round_not_found", "unknown or expired round")
			return
		}
		s.Logger.Error("round guess apply failed")
		writeError(w, http.StatusInternalServerError, "guess_failed", "could not evaluate guess")
		return
	}

	resp := infiniteGuessResponse{
		GuessOutcome: outcome,
		LivesLeft:    updated.LivesLeft,
		Solved:       updated.Status == "solved",
		RoundOver:    updated.Status != "playing",
	}
	if updated.Status == "dead" {
		resp.Answer = &answerCard{Name: target.Name, PhotoURL: target.PhotoURL, PhotoCredit: target.PhotoCredit}
	}
	if user, ok := s.currentUser(r); ok && updated.Status != "playing" {
		if streak, best, newBest, ok := s.noteFinishedRound(ctx, user.ID, round.Pool, updated.Status == "solved"); ok {
			resp.Streak, resp.Best, resp.NewBest = &streak, &best, &newBest
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// noteFinishedRound folds a finished round into the account run: solves
// extend it (reporting whether a new best was set), deaths reset it.
// Best-effort like daily guess recording: the guess response never fails
// because the streak write did.
func (s *Server) noteFinishedRound(ctx context.Context, userID int64, pool domain.Pool, solved bool) (streak, best int, newBest, ok bool) {
	if !solved {
		if err := s.Infinite.NoteDeath(ctx, userID, pool); err != nil {
			s.Logger.Error("infinite record reset failed", "user_id", userID)
			return 0, 0, false, false
		}
		rec, err := s.Infinite.FetchRecord(ctx, userID, pool)
		if err != nil {
			s.Logger.Error("infinite record fetch failed", "user_id", userID)
			return 0, 0, false, false
		}
		return rec.CurrentStreak, rec.BestStreak, false, true
	}
	prev, err := s.Infinite.FetchRecord(ctx, userID, pool)
	if err != nil {
		s.Logger.Error("infinite record fetch failed", "user_id", userID)
		return 0, 0, false, false
	}
	rec, err := s.Infinite.NoteSolve(ctx, userID, pool)
	if err != nil {
		s.Logger.Error("infinite record update failed", "user_id", userID)
		return 0, 0, false, false
	}
	return rec.CurrentStreak, rec.BestStreak, rec.CurrentStreak > prev.BestStreak, true
}

// GET /api/me/infinite/record?pool= — the account's survival run.
func (s *Server) handleInfiniteRecord(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
		return
	}
	if s.infiniteUnavailable(w) {
		return
	}
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	pool, ok := parsePool(w, r.URL.Query().Get("pool"))
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rec, err := s.Infinite.FetchRecord(ctx, user.ID, pool)
	if err != nil {
		s.Logger.Error("infinite record fetch failed", "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "record_failed", "could not load record")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

// infiniteUnavailable guards routes when built without an infinite store.
func (s *Server) infiniteUnavailable(w http.ResponseWriter) bool {
	if s.Infinite == nil {
		writeError(w, http.StatusInternalServerError, "infinite_unavailable", "infinity mode is not configured")
		return true
	}
	return false
}
