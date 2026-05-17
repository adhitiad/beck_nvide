package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
	"nvide-live/pkg/redis"
)

type moderationUseCase struct {
	repo          domain.ModerationRepository
	wsHub         *websocket.Hub
	redis         *redis.Client
	logger        *zap.Logger
	nsfwScanner   domain.NSFWScanner
	wordlistRegex []*compiledWord
	wordlistMu    sync.RWMutex
}

type compiledWord struct {
	ID            domain.UUID
	Word          string
	SeverityLevel int
	Language      string
	IsRegex       bool
	Reg           *regexp.Regexp
}

func NewModerationUseCase(
	repo domain.ModerationRepository,
	wsHub *websocket.Hub,
	redisClient *redis.Client,
	logger *zap.Logger,
	scanner domain.NSFWScanner,
) domain.ModerationUseCase {
	uc := &moderationUseCase{
		repo:        repo,
		wsHub:       wsHub,
		redis:       redisClient,
		logger:      logger,
		nsfwScanner: scanner,
	}
	// Initial wordlist compile
	if err := uc.ReloadWordlist(context.Background()); err != nil {
		logger.Error("Failed to compile toxicity filter wordlist", zap.Error(err))
	}
	return uc
}

func (u *moderationUseCase) ReloadWordlist(ctx context.Context) error {
	u.wordlistMu.Lock()
	defer u.wordlistMu.Unlock()

	words, err := u.repo.GetWordlist(ctx)
	if err != nil {
		return err
	}

	var compiled []*compiledWord
	for _, w := range words {
		var r *regexp.Regexp
		if w.IsRegex {
			r, err = regexp.Compile("(?i)" + w.Word)
			if err != nil {
				u.logger.Warn("Failed to compile regex word", zap.String("word", w.Word), zap.Error(err))
				continue
			}
		} else {
			// exact word border matching
			r, err = regexp.Compile("(?i)\\b" + regexp.QuoteMeta(w.Word) + "\\b")
			if err != nil {
				continue
			}
		}

		compiled = append(compiled, &compiledWord{
			ID:            w.ID,
			Word:          w.Word,
			SeverityLevel: w.SeverityLevel,
			Language:      w.Language,
			IsRegex:       w.IsRegex,
			Reg:           r,
		})
	}

	u.wordlistRegex = compiled
	u.logger.Info("Toxicity wordlist compiled successfully", zap.Int("count", len(compiled)))
	return nil
}

// AWSRekognitionScanner simulator
type AWSRekognitionScanner struct {
	logger *zap.Logger
}

func NewAWSRekognitionScanner(logger *zap.Logger) domain.NSFWScanner {
	return &AWSRekognitionScanner{logger: logger}
}

func (s *AWSRekognitionScanner) Scan(ctx context.Context, imageURL string) (*domain.ScanResult, error) {
	s.logger.Info("Scanning image with AWS Rekognition simulator", zap.String("url", imageURL))
	lower := strings.ToLower(imageURL)
	score := 0.10 // default benign

	if strings.Contains(lower, "nsfw") || strings.Contains(lower, "explicit") || strings.Contains(lower, "nude") {
		score = 0.95
	} else if strings.Contains(lower, "sensitip") || strings.Contains(lower, "bikini") || strings.Contains(lower, "sensual") {
		score = 0.72
	}

	labels := []map[string]interface{}{}
	if score >= 0.80 {
		labels = append(labels, map[string]interface{}{"name": "Explicit Nudity", "confidence": score})
	} else if score >= 0.60 {
		labels = append(labels, map[string]interface{}{"name": "Suggestive Content", "confidence": score})
	}

	return &domain.ScanResult{
		NSFWScore: score,
		IsNSFW:    score >= 0.60,
		Labels:    labels,
	}, nil
}

// GoogleVisionScanner simulator (fallback)
type GoogleVisionScanner struct {
	logger *zap.Logger
}

func NewGoogleVisionScanner(logger *zap.Logger) domain.NSFWScanner {
	return &GoogleVisionScanner{logger: logger}
}

func (s *GoogleVisionScanner) Scan(ctx context.Context, imageURL string) (*domain.ScanResult, error) {
	s.logger.Info("Scanning image with Google Vision SafeSearch simulator", zap.String("url", imageURL))
	lower := strings.ToLower(imageURL)
	score := 0.05

	if strings.Contains(lower, "nsfw") || strings.Contains(lower, "explicit") || strings.Contains(lower, "nude") {
		score = 0.92
	} else if strings.Contains(lower, "sensitip") || strings.Contains(lower, "bikini") || strings.Contains(lower, "sensual") {
		score = 0.68
	}

	labels := []map[string]interface{}{}
	if score >= 0.60 {
		labels = append(labels, map[string]interface{}{"name": "Racy Content", "confidence": score})
	}

	return &domain.ScanResult{
		NSFWScore: score,
		IsNSFW:    score >= 0.60,
		Labels:    labels,
	}, nil
}

// CRUD Rules
func (u *moderationUseCase) CreateRule(ctx context.Context, rule *domain.ModerationRule) error {
	if rule.ID.IsZero() {
		rule.ID = domain.NewUUID()
	}
	return u.repo.CreateRule(ctx, rule)
}

func (u *moderationUseCase) UpdateRule(ctx context.Context, rule *domain.ModerationRule) error {
	return u.repo.UpdateRule(ctx, rule)
}

func (u *moderationUseCase) ListRules(ctx context.Context) ([]*domain.ModerationRule, error) {
	return u.repo.ListRules(ctx)
}

// General Rules Evaluation Engine
func (u *moderationUseCase) EvaluateEvent(ctx context.Context, eventType string, userID domain.UUID, payload map[string]interface{}, streamID *domain.UUID) (*domain.ModerationDecision, error) {
	rules, err := u.repo.GetActiveRulesOrdered(ctx)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		if rule.AppliesTo != "all" && rule.AppliesTo != eventType {
			continue
		}

		matched := false
		var matchedValue float64

		switch rule.ConditionType {
		case "nsfw_score":
			if val, ok := payload["nsfw_score"].(float64); ok && val >= rule.Threshold {
				matched = true
				matchedValue = val
			}
		case "repeated_message":
			if val, ok := payload["repeated_count"].(int); ok && float64(val) >= rule.Threshold {
				matched = true
				matchedValue = float64(val)
			}
		case "caps_ratio":
			if val, ok := payload["caps_ratio"].(float64); ok && val >= rule.Threshold {
				matched = true
				matchedValue = val
			}
		case "link_count":
			if val, ok := payload["link_count"].(int); ok && float64(val) >= rule.Threshold {
				matched = true
				matchedValue = float64(val)
			}
		case "gift_velocity":
			if val, ok := payload["gift_count"].(int); ok && float64(val) >= rule.Threshold {
				matched = true
				matchedValue = float64(val)
			}
		case "toxicity_score":
			if val, ok := payload["toxicity_score"].(float64); ok && val >= rule.Threshold {
				matched = true
				matchedValue = val
			}
		}

		if matched {
			u.logger.Info("Moderation rule matched!", zap.String("rule_code", rule.RuleCode), zap.Float64("matched_val", matchedValue))
			duration := 0
			if rule.ActionDurationSeconds != nil {
				duration = *rule.ActionDurationSeconds
			}

			decision := &domain.ModerationDecision{
				Blocked:      rule.Action == "block" || rule.Action == "ban_temp" || rule.Action == "ban_perm",
				Muted:        rule.Action == "mute",
				Kicked:       rule.Action == "kick",
				Banned:       rule.Action == "ban_temp" || rule.Action == "ban_perm",
				Blurred:      rule.Action == "blur_image",
				ActionTaken:  rule.Action,
				DurationSecs: duration,
				Reason:       fmt.Sprintf("Memicu aturan %s (%s)", rule.Name, rule.RuleCode),
				RuleID:       rule.ID,
			}

			// Apply strike escalation
			escalatedDecision, err := u.ApplyStrikeEscalation(ctx, userID, rule, decision)
			if err == nil {
				decision = escalatedDecision
			}

			return decision, nil
		}
	}

	return &domain.ModerationDecision{ActionTaken: "pass"}, nil
}

// Escalation Engine
func (u *moderationUseCase) ApplyStrikeEscalation(ctx context.Context, userID domain.UUID, rule *domain.ModerationRule, base *domain.ModerationDecision) (*domain.ModerationDecision, error) {
	state, err := u.repo.GetUserModerationState(ctx, userID)
	if err != nil {
		return base, err
	}

	if state == nil {
		state = &domain.UserModerationState{
			ID:                       domain.NewUUID(),
			UserID:                   userID,
			TotalStrikes:             0,
			CurrentBanLevel:          0,
			ConsecutiveSameRuleCount: 0,
		}
	}

	now := time.Now()
	// Reset consecutive counter if it was triggered > 24 hours ago
	if state.LastStrikeAt != nil && now.Sub(*state.LastStrikeAt) > 24*time.Hour {
		state.ConsecutiveSameRuleCount = 0
	}

	state.TotalStrikes++
	state.LastStrikeAt = &now
	state.LastStrikeRuleID = &rule.ID

	if state.LastStrikeRuleID != nil && *state.LastStrikeRuleID == rule.ID {
		state.ConsecutiveSameRuleCount++
	} else {
		state.ConsecutiveSameRuleCount = 1
	}

	decision := *base

	// Check if repeat offenders match escalation trigger
	if state.ConsecutiveSameRuleCount >= rule.MaxStrikes && rule.EscalationRuleID != nil {
		escRule, err := u.repo.GetRuleByID(ctx, *rule.EscalationRuleID)
		if err == nil && escRule != nil {
			u.logger.Info("Escalating moderation action!", zap.String("from", rule.RuleCode), zap.String("to", escRule.RuleCode))
			duration := 0
			if escRule.ActionDurationSeconds != nil {
				duration = *escRule.ActionDurationSeconds
			}
			decision.ActionTaken = escRule.Action
			decision.DurationSecs = duration
			decision.Reason = fmt.Sprintf("Eskalasi berulang: memicu %s (%s)", escRule.Name, escRule.RuleCode)
			decision.RuleID = escRule.ID
			decision.Blocked = escRule.Action == "block" || escRule.Action == "ban_temp" || escRule.Action == "ban_perm"
			decision.Muted = escRule.Action == "mute"
			decision.Kicked = escRule.Action == "kick"
			decision.Banned = escRule.Action == "ban_temp" || escRule.Action == "ban_perm"
		}
	}

	// Apply final ban states to DB/Redis
	if decision.ActionTaken == "mute" {
		state.CurrentBanLevel = 1
		state.IsMuted = true
		until := now.Add(time.Duration(decision.DurationSecs) * time.Second)
		state.MutedUntil = &until
		_ = u.redis.Set(ctx, fmt.Sprintf("mod:mute:%s", userID), "1", time.Duration(decision.DurationSecs)*time.Second)
	} else if decision.ActionTaken == "kick" {
		state.CurrentBanLevel = 2
	} else if decision.ActionTaken == "ban_temp" {
		state.CurrentBanLevel = 3
		state.IsBanned = true
		until := now.Add(time.Duration(decision.DurationSecs) * time.Second)
		state.BannedUntil = &until
		state.BanReason = decision.Reason
		_ = u.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", userID), "1", time.Duration(decision.DurationSecs)*time.Second)
	} else if decision.ActionTaken == "ban_perm" {
		state.CurrentBanLevel = 4
		state.IsBanned = true
		state.BanReason = decision.Reason
		_ = u.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", userID), "1", 365*24*time.Hour)
	}

	err = u.repo.SaveUserModerationState(ctx, state)
	if err != nil {
		u.logger.Error("Failed to save strike escalation state", zap.Error(err))
	}

	return &decision, nil
}

// Chat Behavior Detection (SYNC Middleware Integration)
func (u *moderationUseCase) EvaluateChatMessage(ctx context.Context, userID domain.UUID, streamID domain.UUID, text string) (*domain.ModerationDecision, error) {
	// First check if user is globally muted
	isMuted, err := u.redis.Exists(ctx, fmt.Sprintf("mod:mute:%s", userID))
	if err == nil && isMuted > 0 {
		return &domain.ModerationDecision{
			Muted:       true,
			ActionTaken: "mute",
			Reason:      "Anda sedang dibisukan secara global.",
		}, nil
	}

	cleanText := strings.TrimSpace(text)
	textLen := len(cleanText)
	if textLen == 0 {
		return &domain.ModerationDecision{ActionTaken: "pass"}, nil
	}

	// Calculate metrics
	var capsCount int
	for _, c := range cleanText {
		if c >= 'A' && c <= 'Z' {
			capsCount++
		}
	}
	capsRatio := 0.0
	if textLen > 0 {
		capsRatio = float64(capsCount) / float64(textLen) * 100.0
	}

	// Count link urls
	linkRegex := regexp.MustCompile(`(?i)https?://[^\s]+`)
	linksCount := len(linkRegex.FindAllString(cleanText, -1))

	// Count mentions
	mentionRegex := regexp.MustCompile(`@[a-zA-Z0-9_]+`)
	mentions := mentionRegex.FindAllString(cleanText, -1)
	mentionsCount := len(mentions)

	// Count emojis
	emojiRegex := regexp.MustCompile(`[\x{1F300}-\x{1F9FF}]|[\x{2600}-\x{26FF}]`)
	emojisCount := len(emojiRegex.FindAllString(cleanText, -1))

	// Spam check windows using Redis
	nowUnix := time.Now().Unix()
	window60 := nowUnix / 60
	window30 := nowUnix / 30

	redisKeyPrefix := fmt.Sprintf("mod:chat:%s:%s", userID, streamID)
	rClient := u.redis.GetClient()

	// Repeated message hash check
	hasher := sha256.New()
	hasher.Write([]byte(cleanText))
	textHash := hex.EncodeToString(hasher.Sum(nil))[:16]

	hashKey := fmt.Sprintf("%s:hash:%d", redisKeyPrefix, window60)
	prevHash, _ := rClient.Get(ctx, hashKey).Result()
	repeatedCount := 1
	if prevHash == textHash {
		countKey := fmt.Sprintf("%s:rep_count:%d", redisKeyPrefix, window60)
		rClient.Incr(ctx, countKey)
		rClient.Expire(ctx, countKey, 60*time.Second)
		countStr, _ := rClient.Get(ctx, countKey).Result()
		repeatedCount, _ = strconv.Atoi(countStr)
		if repeatedCount == 0 {
			repeatedCount = 2
		}
	} else {
		rClient.Set(ctx, hashKey, textHash, 60*time.Second)
	}

	// Toxicity filters
	toxicityScore := 0.0
	var matchedWord string
	severity := 0

	u.wordlistMu.RLock()
	for _, w := range u.wordlistRegex {
		if w.Reg.MatchString(cleanText) {
			// Context bypass verify: jika kata kasar adalah username valid yang di-mention
			isMentionBypass := false
			for _, m := range mentions {
				mName := strings.TrimPrefix(m, "@")
				if strings.EqualFold(mName, w.Word) {
					// Check if user username exists
					exists, _ := u.repo.ExistsUsername(ctx, mName)
					if exists {
						isMentionBypass = true
						break
					}
				}
			}

			if !isMentionBypass {
				toxicityScore = 1.0
				matchedWord = w.Word
				severity = w.SeverityLevel
				break
			}
		}
	}
	u.wordlistMu.RUnlock()

	// Evaluate conditions via rules engine
	evalPayload := map[string]interface{}{
		"repeated_count": repeatedCount,
		"caps_ratio":     capsRatio,
		"link_count":     linksCount,
		"toxicity_score": toxicityScore,
	}

	decision, err := u.EvaluateEvent(ctx, "chat", userID, evalPayload, &streamID)
	if err == nil && decision.ActionTaken != "pass" {
		u.ExecuteSyncChatAction(ctx, userID, streamID, decision)
		return decision, nil
	}

	// Fallback/Legacy toxicity severity levels (if not explicitly covered by DB rules)
	if toxicityScore > 0 {
		decision = &domain.ModerationDecision{
			RuleID:      domain.NewUUID(),
			ActionTaken: "warn",
			Reason:      fmt.Sprintf("Kata sensitif terdeteksi: %s", matchedWord),
		}

		if severity == 1 {
			// Mild: Message hidden (sender sees warning only)
			decision.ActionTaken = "warn"
			decision.Blocked = true
			decision.Reason = "Pesan Anda disembunyikan karena mengandung kata kasar ringan."
		} else if severity == 2 {
			// Moderate: Mute 10m
			duration := 600
			decision.ActionTaken = "mute"
			decision.Muted = true
			decision.DurationSecs = duration
			decision.Reason = fmt.Sprintf("Mute otomatis karena kata kasar sedang: %s", matchedWord)
			_ = u.repo.LogModerationAction(ctx, &domain.ModerationLog{
				ID:               domain.NewUUID(),
				UserID:           userID,
				StreamID:         &streamID,
				EvidenceType:     "chat_message",
				EvidenceContent:  cleanText,
				ActionTaken:      "mute",
				ActionDurationSeconds: &duration,
				TriggerType:      "auto",
			})
			_ = u.redis.Set(ctx, fmt.Sprintf("mod:mute:%s", userID), "1", 10*time.Minute)
			u.ExecuteSyncChatAction(ctx, userID, streamID, decision)
		} else if severity == 3 {
			// Severe: Kick + Ban 24h
			duration := 86400
			decision.ActionTaken = "ban_temp"
			decision.Banned = true
			decision.DurationSecs = duration
			decision.Reason = fmt.Sprintf("Ban 24 jam otomatis karena kata kasar berat: %s", matchedWord)
			_ = u.repo.LogModerationAction(ctx, &domain.ModerationLog{
				ID:               domain.NewUUID(),
				UserID:           userID,
				StreamID:         &streamID,
				EvidenceType:     "chat_message",
				EvidenceContent:  cleanText,
				ActionTaken:      "ban_temp",
				ActionDurationSeconds: &duration,
				TriggerType:      "auto",
			})
			_ = u.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", userID), "1", 24*time.Hour)
			u.ExecuteSyncChatAction(ctx, userID, streamID, decision)
		}
		return decision, nil
	}

	// Rolling triggers count on Redis for emoji/mention spam (legacy heuristic safety)
	if emojisCount >= 10 {
		emojiSpamKey := fmt.Sprintf("%s:emojispam:%d", redisKeyPrefix, window30)
		rClient.Incr(ctx, emojiSpamKey)
		rClient.Expire(ctx, emojiSpamKey, 30*time.Second)
		spamCountStr, _ := rClient.Get(ctx, emojiSpamKey).Result()
		if count, _ := strconv.Atoi(spamCountStr); count >= 3 {
			decision = &domain.ModerationDecision{
				ActionTaken:  "warn",
				Reason:       "Peringatan: Hindari membombardir emoji (emoji flood)!",
				DurationSecs: 0,
			}
			u.ExecuteSyncChatAction(ctx, userID, streamID, decision)
			return decision, nil
		}
	}

	if mentionsCount >= 5 {
		duration := 600
		decision = &domain.ModerationDecision{
			ActionTaken:  "mute",
			Muted:        true,
			DurationSecs: duration,
			Reason:       "Mute otomatis 10 menit karena penyebaran mention berlebih (mention spam).",
		}
		_ = u.redis.Set(ctx, fmt.Sprintf("mod:mute:%s", userID), "1", 10*time.Minute)
		u.ExecuteSyncChatAction(ctx, userID, streamID, decision)
		return decision, nil
	}

	return &domain.ModerationDecision{ActionTaken: "pass"}, nil
}

func (u *moderationUseCase) ExecuteSyncChatAction(ctx context.Context, userID domain.UUID, streamID domain.UUID, d *domain.ModerationDecision) {
	username, _ := u.repo.GetUsernameByID(ctx, userID)
	if username == "" {
		username = "User"
	}

	if d.ActionTaken == "warn" {
		// Private websocket warning to target user only
		if u.wsHub != nil {
			msg := &websocket.WSMessage{
				Type:   "moderation_warning",
				RoomID: string(streamID),
				UserID: string(userID),
				Payload: map[string]interface{}{
					"message": d.Reason,
				},
			}
			msgBytes, _ := json.Marshal(msg)
			u.wsHub.BroadcastToRoom(string(streamID), msgBytes)
		}
	} else if d.ActionTaken == "mute" {
		// Mute system message to all users in stream
		if u.wsHub != nil {
			msg := &websocket.WSMessage{
				Type:   "system_announcement",
				RoomID: string(streamID),
				Payload: map[string]interface{}{
					"message": fmt.Sprintf("%s telah dibisukan otomatis selama %d menit. Alasan: %s", username, d.DurationSecs/60, d.Reason),
				},
			}
			msgBytes, _ := json.Marshal(msg)
			u.wsHub.BroadcastToRoom(string(streamID), msgBytes)
		}
	} else if d.ActionTaken == "kick" || d.ActionTaken == "ban_temp" || d.ActionTaken == "ban_perm" {
		// Kick/Ban system message and force WS connection drop
		if u.wsHub != nil {
			msg := &websocket.WSMessage{
				Type:   "system_announcement",
				RoomID: string(streamID),
				Payload: map[string]interface{}{
					"message": fmt.Sprintf("%s telah dikeluarkan dari stream. Alasan: %s", username, d.Reason),
				},
			}
			msgBytes, _ := json.Marshal(msg)
			u.wsHub.BroadcastToRoom(string(streamID), msgBytes)

			// Kick user out of room
			kickKey := fmt.Sprintf("room:kick:%s:%s", streamID, userID)
			_ = u.redis.Set(ctx, kickKey, "1", 10*time.Minute)

			// Broadcast action to trigger client reconnection drop
			dropMsg := &websocket.WSMessage{
				Type:   "force_disconnect",
				RoomID: string(streamID),
				UserID: string(userID),
				Payload: map[string]interface{}{
					"target_user_id": string(userID),
					"reason":         d.Reason,
				},
			}
			dropBytes, _ := json.Marshal(dropMsg)
			u.wsHub.BroadcastToRoom(string(streamID), dropBytes)
		}
	}
}

// Image Moderation Async Queue Upload Scan Pipeline
func (u *moderationUseCase) EnqueueImageScan(ctx context.Context, imageURL string, sourceType string, sourceID domain.UUID) error {
	q := &domain.ImageModerationQueue{
		ID:         domain.NewUUID(),
		ImageURL:   imageURL,
		SourceType: sourceType,
		SourceID:   sourceID,
		Status:     "queued",
	}

	err := u.repo.EnqueueImage(ctx, q)
	if err != nil {
		return err
	}

	u.logger.Info("Enqueued image scan request successfully", zap.String("url", imageURL))
	return nil
}

func (u *moderationUseCase) GetPendingImages(ctx context.Context) ([]*domain.ImageModerationQueue, error) {
	return u.repo.GetPendingImages(ctx)
}

func (u *moderationUseCase) ApproveImage(ctx context.Context, jobID domain.UUID) error {
	job, err := u.repo.GetImageJobByID(ctx, jobID)
	if err != nil || job == nil {
		return fmt.Errorf("job not found")
	}

	job.Status = "completed"
	job.ActionTaken = "pass"
	now := time.Now()
	job.CompletedAt = &now
	return u.repo.UpdateImageJob(ctx, job)
}

func (u *moderationUseCase) RejectImage(ctx context.Context, jobID domain.UUID) error {
	job, err := u.repo.GetImageJobByID(ctx, jobID)
	if err != nil || job == nil {
		return fmt.Errorf("job not found")
	}

	job.Status = "completed"
	job.ActionTaken = "block"
	now := time.Now()
	job.CompletedAt = &now
	return u.repo.UpdateImageJob(ctx, job)
}

// Queue async poll scanned worker execution
func (u *moderationUseCase) ProcessNextImageInQueue(ctx context.Context) error {
	pending, err := u.repo.GetPendingImages(ctx)
	if err != nil || len(pending) == 0 {
		return nil
	}

	job := pending[0]
	u.logger.Info("Processing next image in queue", zap.String("id", string(job.ID)), zap.String("url", job.ImageURL))

	job.Status = "scanning"
	_ = u.repo.UpdateImageJob(ctx, job)

	// AWS primary scan with Vision fallback
	job.Provider = "aws_rekognition"
	result, err := u.nsfwScanner.Scan(ctx, job.ImageURL)
	if err != nil {
		u.logger.Warn("AWS Rekognition failed, trying fallback Google Vision scanner", zap.Error(err))
		job.Provider = "google_vision"
		fallback := NewGoogleVisionScanner(u.logger)
		result, err = fallback.Scan(ctx, job.ImageURL)
	}

	now := time.Now()
	job.CompletedAt = &now

	if err != nil {
		u.logger.Error("Both primary and fallback NSFW scanners failed", zap.Error(err))
		job.Status = "failed"
		job.ActionTaken = "flag_review" // flag for manual review
		// temporary blur for extreme safety
		job.BlurredURL = job.ImageURL + "?blurred=true"
		_ = u.repo.UpdateImageJob(ctx, job)
		return err
	}

	job.Status = "completed"
	job.NSFWScore = &result.NSFWScore
	job.IsNSFW = &result.IsNSFW
	labelsBytes, _ := json.Marshal(result.Labels)
	job.ModerationLabels = labelsBytes

	// Score threshold action Taken
	if result.NSFWScore < 0.60 {
		job.ActionTaken = "pass"
	} else if result.NSFWScore >= 0.60 && result.NSFWScore < 0.80 {
		job.ActionTaken = "blur_image"
		// Generate blurred version (Simulasi Gaussian blur CDN url wrapper)
		job.BlurredURL = job.ImageURL + "?blur=20"
		u.logger.Warn("Image NSFW detected, auto-blur applied", zap.String("url", job.ImageURL), zap.Float64("score", result.NSFWScore))
	} else {
		job.ActionTaken = "block"
		job.BlurredURL = "" // block entirely
		u.logger.Error("Image highly NSFW detected, blocked entirely", zap.String("url", job.ImageURL), zap.Float64("score", result.NSFWScore))

		// Trigger strike to the creator (we map creator userID from sourceID or log context)
		_ = u.repo.LogModerationAction(ctx, &domain.ModerationLog{
			ID:              domain.NewUUID(),
			UserID:          job.SourceID, // shortcut using sourceID as target
			EvidenceType:    "image",
			EvidenceContent: job.ImageURL,
			ActionTaken:     "block",
			TriggerType:     "auto",
		})
	}

	return u.repo.UpdateImageJob(ctx, job)
}

// Gift Fraud Detection Analyzer (Async)
func (u *moderationUseCase) AnalyzeCircularGifting(ctx context.Context, userID domain.UUID) error {
	// Circular wash trading check A -> B -> A within 1 hour
	txs, err := u.repo.GetStreamGiftTransactionsInWindow(ctx, userID, 1*time.Hour)
	if err != nil || len(txs) == 0 {
		return nil
	}

	u.logger.Info("Analyzing circular wash trading patterns", zap.String("user_id", string(userID)), zap.Int("tx_count", len(txs)))

	cycleCount := 0
	var totalValue float64
	var secondaryUserID domain.UUID

	for _, tx := range txs {
		// Get transactions in the opposite direction
		oppTxs, err := u.repo.GetStreamGiftTransactionsInWindow(ctx, tx.ReceiverID, 1*time.Hour)
		if err != nil {
			continue
		}

		for _, oppTx := range oppTxs {
			if oppTx.ReceiverID == userID {
				cycleCount++
				totalValue += float64(tx.TotalPrice + oppTx.TotalPrice)
				secondaryUserID = tx.ReceiverID
			}
		}
	}

	if cycleCount >= 3 && totalValue >= 200000 {
		u.logger.Warn("Circular gifting fraud detected!", zap.String("sender", string(userID)), zap.String("receiver", string(secondaryUserID)))
		// Insert fraud alert
		alert := &domain.GiftFraudAlert{
			ID:                domain.NewUUID(),
			AlertType:         "circular_gifting",
			PrimaryUserID:     userID,
			SecondaryUserID:   &secondaryUserID,
			TotalGiftValueIDR: totalValue,
			TransactionCount:  cycleCount,
			TimeWindowSeconds: 3600,
			Status:            "open",
		}

		// Hold gift transactions for these users
		for _, tx := range txs {
			_ = u.repo.HoldGiftTransaction(ctx, tx.ID)
		}

		// Save alert to database
		_ = u.repo.SaveGiftFraudAlert(ctx, alert)
	}

	return nil
}

// Sync bans/mutes periodically (Background Worker)
func (u *moderationUseCase) SyncUserModerationStates(ctx context.Context) error {
	bans, err := u.repo.GetActiveBans(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, ban := range bans {
		dirty := false

		// Expire global mute
		if ban.IsMuted && ban.MutedUntil != nil && now.After(*ban.MutedUntil) {
			ban.IsMuted = false
			ban.MutedUntil = nil
			_ = u.redis.Del(ctx, fmt.Sprintf("mod:mute:%s", ban.UserID))
			dirty = true
			u.logger.Info("Auto-lifting expired mute status", zap.String("user_id", string(ban.UserID)))
		}

		// Expire global ban
		if ban.IsBanned && ban.BannedUntil != nil && now.After(*ban.BannedUntil) {
			ban.IsBanned = false
			ban.BannedUntil = nil
			_ = u.redis.Del(ctx, fmt.Sprintf("jwt:blacklist:%s", ban.UserID))
			dirty = true
			u.logger.Info("Auto-lifting expired ban status", zap.String("user_id", string(ban.UserID)))
		}

		if dirty {
			_ = u.repo.SaveUserModerationState(ctx, ban)
		}
	}

	return nil
}

// Wordlist CRUD
func (u *moderationUseCase) GetWordlist(ctx context.Context) ([]*domain.ModerationWordlist, error) {
	return u.repo.GetWordlist(ctx)
}

func (u *moderationUseCase) AddWord(ctx context.Context, word string, severity int, lang string, isRegex bool) error {
	w := &domain.ModerationWordlist{
		ID:            domain.NewUUID(),
		Word:          word,
		SeverityLevel: severity,
		Language:      lang,
		IsRegex:       isRegex,
	}
	err := u.repo.AddWord(ctx, w)
	if err != nil {
		return err
	}
	return u.ReloadWordlist(ctx)
}

func (u *moderationUseCase) DeleteWord(ctx context.Context, word string) error {
	err := u.repo.DeleteWord(ctx, word)
	if err != nil {
		return err
	}
	return u.ReloadWordlist(ctx)
}

// Appeals & Support
func (u *moderationUseCase) SubmitAppeal(ctx context.Context, logID domain.UUID, reason string) error {
	log, err := u.repo.GetModerationLogByID(ctx, logID)
	if err != nil || log == nil {
		return fmt.Errorf("log not found")
	}

	now := time.Now()
	// Mute/kick appeal window: 15 minutes. Ban appeal window: 7 days
	limit := 15 * time.Minute
	if log.ActionTaken == "ban_temp" || log.ActionTaken == "ban_perm" {
		limit = 7 * 24 * time.Hour
	}

	if now.Sub(log.ActionExecutedAt) > limit {
		return fmt.Errorf("appeal window has expired for this action")
	}

	log.IsAppealed = true
	log.AppealStatus = "pending"
	log.EvidenceContent = log.EvidenceContent + "\n[APPEAL REASON]: " + reason

	return u.repo.SubmitAppealUpdate(ctx, logID, log.EvidenceContent)
}

func (u *moderationUseCase) ListLogs(ctx context.Context, userID *domain.UUID, streamID *domain.UUID, action *string, limit, offset int) ([]*domain.ModerationLog, error) {
	return u.repo.ListModerationLogs(ctx, userID, streamID, action, limit, offset)
}

func (u *moderationUseCase) GetActiveBans(ctx context.Context) ([]*domain.UserModerationState, error) {
	return u.repo.GetActiveBans(ctx)
}

func (u *moderationUseCase) ManualOverride(ctx context.Context, adminID domain.UUID, userID domain.UUID, actionType string, reason string) error {
	state, err := u.repo.GetUserModerationState(ctx, userID)
	if err != nil {
		return err
	}

	if state == nil {
		state = &domain.UserModerationState{
			ID:     domain.NewUUID(),
			UserID: userID,
		}
	}

	now := time.Now()

	if actionType == "unmute" {
		state.IsMuted = false
		state.MutedUntil = nil
		_ = u.redis.Del(ctx, fmt.Sprintf("mod:mute:%s", userID))
	} else if actionType == "unban" {
		state.IsBanned = false
		state.BannedUntil = nil
		_ = u.redis.Del(ctx, fmt.Sprintf("jwt:blacklist:%s", userID))
	} else if actionType == "mute" {
		state.IsMuted = true
		until := now.Add(24 * time.Hour) // manual default 24h
		state.MutedUntil = &until
		_ = u.redis.Set(ctx, fmt.Sprintf("mod:mute:%s", userID), "1", 24*time.Hour)
	} else if actionType == "ban_perm" {
		state.IsBanned = true
		state.BanReason = reason
		_ = u.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", userID), "1", 365*24*time.Hour)
	}

	err = u.repo.SaveUserModerationState(ctx, state)
	if err != nil {
		return err
	}

	// Immutable log
	return u.repo.LogModerationAction(ctx, &domain.ModerationLog{
		ID:               domain.NewUUID(),
		UserID:           userID,
		TriggerType:      "manual",
		EvidenceType:     "admin_override",
		EvidenceContent:  reason,
		ActionTaken:      actionType,
		ActionExecutedBy: &adminID,
	})
}
