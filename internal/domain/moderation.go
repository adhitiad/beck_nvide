package domain

import (
	"context"
	"time"
)

// ModerationRule represents a single configurable security filter/rule
type ModerationRule struct {
	ID                    UUID       `json:"id"`
	RuleCode              string     `json:"rule_code"`
	Name                  string     `json:"name"`
	Category              string     `json:"category"` // 'content', 'behavior', 'fraud', 'spam'
	ConditionType         string     `json:"condition_type"` // 'nsfw_score', 'repeated_message', 'gift_velocity', 'toxicity_score', 'ip_cluster', 'caps_ratio', 'link_count'
	Threshold             float64    `json:"threshold"`
	TimeWindowSeconds     int        `json:"time_window_seconds"`
	Action                string     `json:"action"` // 'warn', 'mute', 'kick', 'ban_temp', 'ban_perm', 'blur_image', 'flag_review'
	ActionDurationSeconds *int       `json:"action_duration_seconds,omitempty"`
	EscalationRuleID      *UUID      `json:"escalation_rule_id,omitempty"`
	MaxStrikes            int        `json:"max_strikes"`
	AppliesTo             string     `json:"applies_to"` // 'all', 'chat', 'story', 'stream', 'gift'
	IsActive              bool       `json:"is_active"`
	Priority              int        `json:"priority"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// UserModerationState tracks global moderation strikes and active restrictions per user
type UserModerationState struct {
	ID                       UUID       `json:"id"`
	UserID                   UUID       `json:"user_id"`
	TotalStrikes             int        `json:"total_strikes"`
	CurrentBanLevel          int        `json:"current_ban_level"` // 0=none, 1=mute, 2=kick, 3=tempban, 4=permaban
	IsMuted                  bool       `json:"is_muted"`
	MutedUntil               *time.Time `json:"muted_until,omitempty"`
	IsBanned                 bool       `json:"is_banned"`
	BannedUntil              *time.Time `json:"banned_until,omitempty"`
	BanReason                string     `json:"ban_reason,omitempty"`
	LastStrikeAt             *time.Time `json:"last_strike_at,omitempty"`
	LastStrikeRuleID         *UUID      `json:"last_strike_rule_id,omitempty"`
	ConsecutiveSameRuleCount int        `json:"consecutive_same_rule_count"`
	SuspectedBotScore        float64    `json:"suspected_bot_score"`
	DeviceFingerprintHash    string     `json:"device_fingerprint_hash,omitempty"`
	IPClusterID              *UUID      `json:"ip_cluster_id,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

// ModerationLog represents immutable audit records of moderation decisions
type ModerationLog struct {
	ID                    UUID      `json:"id"`
	UserID                UUID      `json:"user_id"`
	StreamID              *UUID     `json:"stream_id,omitempty"`
	ConversationID        *UUID     `json:"conversation_id,omitempty"`
	RuleID                *UUID     `json:"rule_id,omitempty"`
	TriggerType           string    `json:"trigger_type"` // 'auto', 'manual', 'appeal', 'system'
	EvidenceType          string    `json:"evidence_type"` // 'chat_message', 'image', 'gift_pattern', 'ip_cluster', 'device_fingerprint'
	EvidenceContent       string    `json:"evidence_content,omitempty"`
	EvidenceMetadata      []byte    `json:"evidence_metadata,omitempty"`
	ActionTaken           string    `json:"action_taken"`
	ActionDurationSeconds *int      `json:"action_duration_seconds,omitempty"`
	ActionExecutedAt      time.Time `json:"action_executed_at"`
	ActionExecutedBy      *UUID     `json:"action_executed_by,omitempty"`
	RelatedMessageID      *UUID     `json:"related_message_id,omitempty"`
	RelatedImageURL       string    `json:"related_image_url,omitempty"`
	IsAppealed            bool      `json:"is_appealed"`
	AppealStatus          string    `json:"appeal_status,omitempty"` // 'pending', 'approved', 'rejected'
	CreatedAt             time.Time `json:"created_at"`
}

// ImageModerationQueue represents queued and processed image moderation requests
type ImageModerationQueue struct {
	ID               UUID       `json:"id"`
	ImageURL         string     `json:"image_url"`
	SourceType       string     `json:"source_type"` // 'chat', 'story', 'avatar', 'stream_thumbnail', 'clip'
	SourceID         UUID       `json:"source_id"`
	Status           string     `json:"status"` // queued, scanning, completed, failed
	Provider         string     `json:"provider,omitempty"`
	NSFWScore        *float64   `json:"nsfw_score,omitempty"`
	IsNSFW           *bool      `json:"is_nsfw,omitempty"`
	ModerationLabels []byte     `json:"moderation_labels,omitempty"`
	ActionTaken      string     `json:"action_taken,omitempty"`
	BlurredURL       string     `json:"blurred_url,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// ChatPatternSnapshot contains rolling window stats of a user's messaging pattern
type ChatPatternSnapshot struct {
	ID                   UUID      `json:"id"`
	UserID               UUID      `json:"user_id"`
	StreamID             *UUID     `json:"stream_id,omitempty"`
	WindowStart          time.Time `json:"window_start"`
	WindowEnd            time.Time `json:"window_end"`
	MessageCount         int       `json:"message_count"`
	UniqueContentCount   int       `json:"unique_content_count"`
	RepeatedContentCount int       `json:"repeated_content_count"`
	CapsRatio            float64   `json:"caps_ratio"`
	LinkCount            int       `json:"link_count"`
	EmojiCount           int       `json:"emoji_count"`
	IsSpamDetected       bool      `json:"is_spam_detected"`
	SpamScore            float64   `json:"spam_score"`
	CreatedAt            time.Time `json:"created_at"`
}

// GiftFraudAlert logs suspicious gift activities (velocity, IP clusters, circular gifting)
type GiftFraudAlert struct {
	ID                UUID       `json:"id"`
	AlertType         string     `json:"alert_type"` // 'velocity_spike', 'circular_gifting', 'new_account_burst', 'ip_cluster'
	PrimaryUserID     UUID       `json:"primary_user_id"`
	SecondaryUserID   *UUID      `json:"secondary_user_id,omitempty"`
	StreamID          *UUID      `json:"stream_id,omitempty"`
	TotalGiftValueIDR float64    `json:"total_gift_value_idr,omitempty"`
	TransactionCount  int        `json:"transaction_count"`
	TimeWindowSeconds int        `json:"time_window_seconds"`
	IPAddress         string     `json:"ip_address,omitempty"`
	DeviceFingerprint string     `json:"device_fingerprint,omitempty"`
	AccountClusterID  *UUID      `json:"account_cluster_id,omitempty"`
	Status            string     `json:"status"` // open, under_review, confirmed_fraud, false_positive
	ReviewedBy        *UUID      `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// ModerationWordlist represents registered keywords for toxic content filtering
type ModerationWordlist struct {
	ID            UUID      `json:"id"`
	Word          string    `json:"word"`
	SeverityLevel int       `json:"severity_level"` // 1=mild, 2=moderate, 3=severe
	Language      string    `json:"language"`
	IsRegex       bool      `json:"is_regex"`
	CreatedAt     time.Time `json:"created_at"`
}

// ModerationDecision represents the outcome of a rule execution
type ModerationDecision struct {
	Blocked      bool   `json:"blocked"`
	Muted        bool   `json:"muted"`
	Kicked       bool   `json:"kicked"`
	Banned       bool   `json:"banned"`
	Blurred      bool   `json:"blurred"`
	ActionTaken  string `json:"action_taken"` // 'warn', 'mute', 'kick', 'ban_temp', 'ban_perm', 'blur_image'
	DurationSecs int    `json:"duration_seconds"`
	Reason       string `json:"reason"`
	RuleID       UUID   `json:"rule_id"`
}

// ScanResult is the standard structure returned by external NSFW scanners
type ScanResult struct {
	NSFWScore float64                  `json:"nsfw_score"`
	IsNSFW    bool                     `json:"is_nsfw"`
	Labels    []map[string]interface{} `json:"labels"`
}

// NSFWScanner defines the interface for scanning upload files
type NSFWScanner interface {
	Scan(ctx context.Context, imageURL string) (*ScanResult, error)
}

// ModerationRepository defines repository operations
type ModerationRepository interface {
	// Rules CRUD
	CreateRule(ctx context.Context, rule *ModerationRule) error
	UpdateRule(ctx context.Context, rule *ModerationRule) error
	GetRuleByID(ctx context.Context, id UUID) (*ModerationRule, error)
	GetRuleByCode(ctx context.Context, code string) (*ModerationRule, error)
	ListRules(ctx context.Context) ([]*ModerationRule, error)
	GetActiveRulesOrdered(ctx context.Context) ([]*ModerationRule, error)

	// User State
	GetUserModerationState(ctx context.Context, userID UUID) (*UserModerationState, error)
	SaveUserModerationState(ctx context.Context, state *UserModerationState) error

	// Logs
	LogModerationAction(ctx context.Context, log *ModerationLog) error
	ListModerationLogs(ctx context.Context, userID *UUID, streamID *UUID, action *string, limit, offset int) ([]*ModerationLog, error)
	GetActiveBans(ctx context.Context) ([]*UserModerationState, error)
	GetModerationLogByID(ctx context.Context, id UUID) (*ModerationLog, error)

	// Image Moderation Queue
	EnqueueImage(ctx context.Context, q *ImageModerationQueue) error
	GetPendingImages(ctx context.Context) ([]*ImageModerationQueue, error)
	GetImageJobByID(ctx context.Context, id UUID) (*ImageModerationQueue, error)
	UpdateImageJob(ctx context.Context, q *ImageModerationQueue) error

	// Wordlist
	GetWordlist(ctx context.Context) ([]*ModerationWordlist, error)
	AddWord(ctx context.Context, w *ModerationWordlist) error
	DeleteWord(ctx context.Context, word string) error

	// Gift hold logic
	HoldGiftTransaction(ctx context.Context, txID UUID) error
	ReleaseGiftTransaction(ctx context.Context, txID UUID) error
	GetStreamGiftTransactionsInWindow(ctx context.Context, userID UUID, window time.Duration) ([]*GiftTransaction, error)

	// User info lookup
	GetUsernameByID(ctx context.Context, userID UUID) (string, error)
	ExistsUsername(ctx context.Context, username string) (bool, error)

	// Save circular fraud alert
	SaveGiftFraudAlert(ctx context.Context, alert *GiftFraudAlert) error

	// Submit appeal update helper
	SubmitAppealUpdate(ctx context.Context, logID UUID, updatedEvidenceContent string) error
}

// ModerationUseCase defines core moderation engine logic
type ModerationUseCase interface {
	// Rules CRUD & Evaluation
	CreateRule(ctx context.Context, rule *ModerationRule) error
	UpdateRule(ctx context.Context, rule *ModerationRule) error
	ListRules(ctx context.Context) ([]*ModerationRule, error)
	EvaluateEvent(ctx context.Context, eventType string, userID UUID, payload map[string]interface{}, streamID *UUID) (*ModerationDecision, error)

	// Chat Behavior Detection
	EvaluateChatMessage(ctx context.Context, userID UUID, streamID UUID, text string) (*ModerationDecision, error)

	// Image Scan Pipeline
	EnqueueImageScan(ctx context.Context, imageURL string, sourceType string, sourceID UUID) error
	GetPendingImages(ctx context.Context) ([]*ImageModerationQueue, error)
	ApproveImage(ctx context.Context, jobID UUID) error
	RejectImage(ctx context.Context, jobID UUID) error

	// Wordlist CRUD
	GetWordlist(ctx context.Context) ([]*ModerationWordlist, error)
	AddWord(ctx context.Context, word string, severity int, lang string, isRegex bool) error
	DeleteWord(ctx context.Context, word string) error

	// Appeals & Administration
	SubmitAppeal(ctx context.Context, logID UUID, reason string) error
	ListLogs(ctx context.Context, userID *UUID, streamID *UUID, action *string, limit, offset int) ([]*ModerationLog, error)
	GetActiveBans(ctx context.Context) ([]*UserModerationState, error)
	ManualOverride(ctx context.Context, adminID UUID, userID UUID, actionType string, reason string) error

	// Async Workers API
	ProcessNextImageInQueue(ctx context.Context) error
	AnalyzeCircularGifting(ctx context.Context, userID UUID) error
	SyncUserModerationStates(ctx context.Context) error
}
