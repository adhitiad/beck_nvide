package database

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GORM Models for AutoMigrate only (queries still use pgx)

type Role struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string `gorm:"type:varchar(50);unique;not null"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Permission struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Resource    string `gorm:"type:varchar(100);not null"`
	Action      string `gorm:"type:varchar(50);not null"`
	Name        string `gorm:"type:varchar(200);unique;not null"`
	Description string `gorm:"type:text"`
	CreatedAt   time.Time
}

type RolePermission struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoleID       string `gorm:"type:uuid;not null"`
	PermissionID string `gorm:"type:uuid;not null"`
	CreatedAt    time.Time
}

type User struct {
	ID                  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username            string     `gorm:"type:varchar(50);unique"`
	Email               string     `gorm:"type:varchar(255);unique;not null"`
	PasswordHash        string     `gorm:"type:varchar(255)"`
	RoleID              string     `gorm:"type:uuid"`
	Role                string     `gorm:"type:varchar(50);default:'user'"`
	Banned              bool       `gorm:"default:false"`
	BanReason           string     `gorm:"type:text"`
	BanExpires          *time.Time
	AvatarURL           string     `gorm:"type:text"`
	IsVerified          bool       `gorm:"default:false"`
	
	// Better Auth Compatibility
	Name                string     `gorm:"type:varchar(255)"`
	EmailVerified       bool       `gorm:"default:false"`
	Image               string     `gorm:"type:text"`
	VerificationToken   string     `gorm:"type:varchar(255)"`
	ResetToken          string     `gorm:"type:varchar(255)"`
	ResetTokenExpiresAt *time.Time
	LastLoginAt         *time.Time
	UserXP              int64      `gorm:"default:0"`
	UserLevel           int        `gorm:"default:1"`
	HostXP              int64      `gorm:"default:0"`
	HostLevel           int        `gorm:"default:1"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Better Auth Models
type Session struct {
	ID        string    `gorm:"type:varchar(255);primaryKey"`
	ExpiresAt time.Time `gorm:"not null;index"`
	Token     string    `gorm:"type:text;unique;not null;index"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
	IPAddress string    `gorm:"type:text"`
	UserAgent string    `gorm:"type:text"`
	UserID    string    `gorm:"type:uuid;not null"`
}

type Account struct {
	ID                    string    `gorm:"type:varchar(255);primaryKey"`
	AccountID             string    `gorm:"type:text;not null"`
	ProviderID            string    `gorm:"type:text;not null"`
	UserID                string    `gorm:"type:uuid;not null"`
	AccessToken           string    `gorm:"type:text"`
	RefreshToken          string    `gorm:"type:text"`
	IDToken               string    `gorm:"type:text"`
	AccessTokenExpiresAt  *time.Time
	RefreshTokenExpiresAt *time.Time
	Scope                 string    `gorm:"type:text"`
	Password              string    `gorm:"type:text"`
	CreatedAt             time.Time `gorm:"not null"`
	UpdatedAt             time.Time `gorm:"not null"`
}

type Verification struct {
	ID         string    `gorm:"type:varchar(255);primaryKey"`
	Identifier string    `gorm:"type:text;not null"`
	Value      string    `gorm:"type:text;not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type RefreshToken struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string `gorm:"type:uuid;not null;index"`
	TokenHash string `gorm:"type:text;unique;not null"`
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type Story struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string `gorm:"type:uuid;not null;index"`
	MediaURL  string `gorm:"type:text;not null"`
	MediaType string `gorm:"type:varchar(20);not null"`
	Caption   string `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"index"`
	IsActive  bool `gorm:"default:true"`
	ViewCount int  `gorm:"default:0"`
	CreatedAt time.Time
}

type StoryView struct {
	ID       string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoryID  string `gorm:"type:uuid;not null;index"`
	ViewerID string `gorm:"type:uuid;not null"`
	ViewedAt time.Time
}

type Comment struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoryID   string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null;index"`
	ParentID  string `gorm:"type:uuid"`
	Content   string `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentLike struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CommentID string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null"`
	CreatedAt time.Time
}

type Like struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string `gorm:"type:uuid;not null;index"`
	StoryID   string `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
}

type LiveRoom struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string `gorm:"type:varchar(100)"`
	Type      string `gorm:"type:varchar(20);default:'group'"`
	TargetID  string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LiveRoom) TableName() string {
	return "live_rooms"
}

type LiveRoomParticipant struct {
	ID       string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID   string `gorm:"type:uuid;not null;index"`
	UserID   string `gorm:"type:uuid;not null"`
	JoinedAt time.Time
}

func (LiveRoomParticipant) TableName() string {
	return "live_room_participants"
}

type LiveMessage struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID    string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null"`
	Content   string `gorm:"type:text;not null"`
	Type      string `gorm:"type:varchar(20);default:'text'"`
	ReplyToID string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LiveMessage) TableName() string {
	return "live_messages"
}

type Stream struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID        string `gorm:"type:uuid;not null;index"`
	Title         string `gorm:"type:varchar(255);not null"`
	Description   string `gorm:"type:text"`
	ThumbnailURL  string `gorm:"type:varchar(255)"`
	Status        string `gorm:"type:varchar(50);default:'preparing';index"`
	StartedAt     *time.Time `gorm:"index"`
	EndedAt       *time.Time
	ViewerPeak    int `gorm:"default:0"`
	TotalDuration int `gorm:"default:0"`
	RoomID        string `gorm:"type:uuid;not null;unique"`
	
	// Bigo/Mango Room Config & Modes
	RoomMode            string  `gorm:"type:varchar(20);default:'public'"`
	RoomPasswordHash   string  `gorm:"type:varchar(255)"`
	EntryFeeIDR        float64 `gorm:"type:decimal(12,2);default:0"`
	MinLevelToEnter    int     `gorm:"default:0"`
	Category           string  `gorm:"type:varchar(50)"`
	Tags               string  `gorm:"type:text"`
	MaxResolution      string  `gorm:"type:varchar(10);default:'720p'"`
	IsScreenShare      bool    `gorm:"default:false"`
	IsCoHostEnabled    bool    `gorm:"default:false"`
	MaxCoHosts         int     `gorm:"default:3"`
	ViewerCount        int     `gorm:"default:0"`
	TotalGiftValueIDR  float64 `gorm:"type:decimal(15,2);default:0"`
	LikeCount          int     `gorm:"default:0"`
	ShareCount         int     `gorm:"default:0"`
	CurrentPKID        string  `gorm:"type:uuid"`
	IsPKEligible       bool    `gorm:"default:false"`
	ChatMode           string  `gorm:"type:varchar(20);default:'normal'"`
	ChatSlowModeSeconds int     `gorm:"default:0"`
	CountryCode        string  `gorm:"type:varchar(5)"`
	Language           string  `gorm:"type:varchar(10)"`

	// Mux Integration
	StreamKey    string `gorm:"type:text"`
	PlaybackID   string `gorm:"type:text"`
	MuxAssetID   string `gorm:"type:text"`

	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StreamSession struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID  string `gorm:"type:uuid;not null;index"`
	ViewerID  string `gorm:"type:uuid;not null;index"`
	JoinedAt  time.Time
	LeftAt    *time.Time
	Duration  int `gorm:"default:0"`
	IPAddress string `gorm:"type:varchar(45)"`
}

type VODMedia struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       string `gorm:"type:uuid;not null;index"`
	Title        string `gorm:"type:varchar(255);not null"`
	Description  string `gorm:"type:text"`
	OriginalURL  string `gorm:"type:varchar(255);not null"`
	HLSURL       string `gorm:"type:varchar(255)"`
	ThumbnailURL string `gorm:"type:varchar(255)"`
	Duration     int `gorm:"default:0"`
	FileSize     int64 `gorm:"default:0"`
	Status       string `gorm:"type:varchar(50);default:'processing'"`
	Visibility   string `gorm:"type:varchar(50);default:'public'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type Wallet struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string `gorm:"type:uuid;unique;not null"`
	Balance   int64  `gorm:"default:0"`
	Currency  string `gorm:"type:varchar(10);default:'IDR'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Transaction struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID      string `gorm:"type:uuid;not null;index"`
	Type          string `gorm:"type:varchar(30);not null"`
	Amount        int64  `gorm:"not null"`
	BalanceBefore int64
	BalanceAfter  int64
	Reference     string `gorm:"type:varchar(255)"`
	Description   string `gorm:"type:text"`
	Status        string `gorm:"type:varchar(20);default:'completed'"`
	CreatedAt     time.Time
}

type Gift struct {
	ID                       string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name                     string  `gorm:"type:varchar(100);not null"`
	Code                     string  `gorm:"type:varchar(20);unique;not null"`
	Price                    float64 `gorm:"type:decimal(12,2);not null"`
	PriceCoins               int     `gorm:"not null"`
	IconURL                  string  `gorm:"type:text"`
	AnimationURL             string  `gorm:"type:text"`
	SoundURL                 string  `gorm:"type:text"`
	ComboTiers               string  `gorm:"type:text;default:'10,66,188,520,1314,3344,9999'"` // comma-separated values
	ComboAnimationURL        string  `gorm:"type:text"`
	Type                     string  `gorm:"type:varchar(20);default:'normal'"`
	Rarity                   string  `gorm:"type:varchar(20);default:'common'"`
	IsLuckyBox               bool    `gorm:"default:false"`
	RoomEffect               string  `gorm:"type:varchar(50)"`
	EffectDurationSeconds    int     `gorm:"default:5"`
	HostCommissionPercent    float64 `gorm:"type:decimal(5,2);default:60.0"`
	AgencyCommissionPercent  float64 `gorm:"type:decimal(5,2);default:20.0"`
	PlatformPercent          float64 `gorm:"type:decimal(5,2);default:20.0"`
	IsActive                 bool    `gorm:"default:true"`
	CreatedAt                time.Time
}

type GiftTransaction struct {
	ID                 string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GiftID             string    `gorm:"type:uuid;not null"`
	SenderID           string    `gorm:"type:uuid;not null;index"`
	ReceiverID         string    `gorm:"type:uuid;not null;index"`
	StreamID           string    `gorm:"type:uuid;not null;index"`
	AgencyID           string    `gorm:"type:uuid"`
	Quantity           int       `gorm:"default:1"`
	ComboCount         int       `gorm:"default:1"`
	TotalPrice         int64
	AgencyCommission   int64
	HostEarning        int64
	PlatformFee        int64
	IsDuringPK         bool      `gorm:"default:false"`
	PKBattleID         *string   `gorm:"type:uuid"`
	PKSide             string    `gorm:"type:varchar(1)"` // 'a' or 'b'
	AnimationTriggered string    `gorm:"type:varchar(50)"`
	HostEarningIDR     float64   `gorm:"type:decimal(15,2)"`
	AgencyEarningIDR   float64   `gorm:"type:decimal(15,2)"`
	PlatformEarningIDR float64   `gorm:"type:decimal(15,2)"`
	Status             string    `gorm:"type:varchar(20);default:'completed'"`
	CreatedAt          time.Time
}

type Agency struct {
	ID            string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string  `gorm:"type:varchar(100);not null"`
	OwnerID       string  `gorm:"type:uuid;not null"`
	RevenueShare  float64 `gorm:"default:0.067"`
	TotalHosts    int     `gorm:"default:0"`
	TotalRevenue  float64 `gorm:"default:0"`
	IsActive      bool    `gorm:"default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type HostApplication struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      string `gorm:"type:uuid;not null;index"`
	AgencyID    string `gorm:"type:uuid"`
	Status      string `gorm:"type:varchar(20);default:'pending'"`
	RealName    string `gorm:"type:varchar(100)"`
	Phone       string `gorm:"type:varchar(20)"`
	IDCardURL   string `gorm:"type:text"`
	ReviewedBy  string `gorm:"type:uuid"`
	ReviewNotes string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DuitkuPayment struct {
	ID              string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TransactionID   string `gorm:"type:uuid;not null;index"`
	MerchantOrderID string `gorm:"type:varchar(100);unique"`
	DuitkuReference string `gorm:"type:varchar(100)"`
	PaymentURL      string `gorm:"type:text"`
	VANumber        string `gorm:"type:varchar(50)"`
	PaymentMethod   string `gorm:"type:varchar(50)"`
	Status          string `gorm:"type:varchar(20);default:'pending'"`
	Amount          int64
	ExpiryAt        *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CryptoWallet struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     string `gorm:"type:uuid;not null;index"`
	Chain      string `gorm:"type:varchar(20);not null"`
	Address    string `gorm:"type:varchar(255);not null"`
	PrivateKey string `gorm:"type:text"`
	CreatedAt  time.Time
}

type CryptoMasterWallet struct {
	ID                  string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Chain               string  `gorm:"type:varchar(20);not null"`
	PublicKey           string  `gorm:"type:varchar(255);not null"`
	EncryptedPrivateKey string  `gorm:"type:text"`
	DerivationPath      string  `gorm:"type:varchar(100)"`
	Balance             float64 `gorm:"default:0"`
	Status              string  `gorm:"type:varchar(20);default:'active'"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type CryptoDepositAddress struct {
	ID              string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          string `gorm:"type:uuid;not null;index"`
	Chain           string `gorm:"type:varchar(20);not null"`
	Address         string `gorm:"type:varchar(255);not null"`
	DerivationIndex int
	MasterWalletID  string `gorm:"type:uuid;not null"`
	IsActive        bool   `gorm:"default:true"`
	CreatedAt       time.Time
}

type CryptoTransaction struct {
	ID                    string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                string     `gorm:"type:uuid;not null;index"`
	Type                  string     `gorm:"type:varchar(20);not null;column:type"`
	Chain                 string     `gorm:"type:varchar(20);not null"`
	Asset                 string     `gorm:"type:varchar(20)"`
	AmountCrypto          float64    `gorm:"column:amount_crypto"`
	AmountIDR             float64    `gorm:"column:amount_idr"`
	ExchangeRate          float64    `gorm:"column:exchange_rate"`
	TxHash                string     `gorm:"type:varchar(255);column:tx_hash"`
	FromAddress           string     `gorm:"type:varchar(255);column:from_address"`
	ToAddress             string     `gorm:"type:varchar(255);column:to_address"`
	Confirmations         int        `gorm:"default:0;column:confirmations"`
	RequiredConfirmations int        `gorm:"default:1;column:required_confirmations"`
	Status                string     `gorm:"type:varchar(20);default:'pending';column:status"`
	FeeCrypto             float64    `gorm:"default:0;column:fee_crypto"`
	FeeIDR                float64    `gorm:"default:0;column:fee_idr"`
	Metadata              string     `gorm:"type:jsonb"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CompletedAt           *time.Time
}

type CryptoExchangeRate struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Asset     string    `gorm:"type:varchar(20);not null;unique:idx_asset_currency"`
	Currency  string    `gorm:"type:varchar(10);not null;unique:idx_asset_currency"`
	Rate      float64
	Source    string    `gorm:"type:varchar(50)"`
	FetchedAt time.Time
}

type CryptoWithdrawalWhitelist struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     string `gorm:"type:uuid;not null;index"`
	Chain      string `gorm:"type:varchar(20);not null"`
	Address    string `gorm:"type:varchar(255);not null"`
	Label      string `gorm:"type:varchar(100)"`
	IsVerified bool   `gorm:"default:false"`
	CreatedAt  time.Time
}

type BookingLocationLog struct {
	ID         string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BookingID  string  `gorm:"type:uuid;not null;index"`
	UserID     string  `gorm:"type:uuid;not null"`
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

type Conversation struct {
	ID            string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Type          string     `gorm:"type:varchar(20);default:'direct'"`
	InitiatorID   string     `gorm:"type:uuid;not null;index"`
	RecipientID   string     `gorm:"type:uuid;not null;index"`
	LastMessageID *string    `gorm:"type:uuid"`
	LastMessageAt *time.Time `gorm:"index"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Conversation) TableName() string {
	return "conversations"
}

type ConversationParticipant struct {
	ID                string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID    string `gorm:"type:uuid;not null;index"`
	UserID            string `gorm:"type:uuid;not null;index"`
	UnreadCount       int    `gorm:"default:0"`
	IsMuted           bool   `gorm:"default:false"`
	IsArchived        bool   `gorm:"default:false"`
	IsPinned          bool   `gorm:"default:false"`
	IsDeleted         bool   `gorm:"default:false"`
	LastReadMessageID *string `gorm:"type:uuid"`
	JoinedAt          time.Time
	UpdatedAt         time.Time
}

func (ConversationParticipant) TableName() string {
	return "conversation_participants"
}

type PrivateMessage struct {
	ID               string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID   string `gorm:"type:uuid;not null;index"`
	SenderID         string `gorm:"type:uuid;not null"`
	Type             string `gorm:"type:varchar(20);default:'text'"`
	Content          *string `gorm:"type:text"`
	Metadata         []byte `gorm:"type:jsonb"`
	ReplyToMessageID *string `gorm:"type:uuid"`
	IsEdited         bool   `gorm:"default:false"`
	IsDeleted        bool   `gorm:"default:false"`
	IsExpired        bool   `gorm:"default:false"`
	DisappearMode    string `gorm:"type:varchar(20);default:'none'"`
	DisappearAt      *time.Time
	ViewedAt         *time.Time
	IsScreenshot     bool `gorm:"default:false"`
	IsForwarded      bool `gorm:"default:false"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (PrivateMessage) TableName() string {
	return "messages" 
}

type MessageReaction struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null"`
	Emoji     string `gorm:"type:varchar(50);not null"`
	CreatedAt time.Time
}

func (MessageReaction) TableName() string {
	return "message_reactions"
}

type MessageView struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID string `gorm:"type:uuid;not null;index"`
	ViewerID  string `gorm:"type:uuid;not null"`
	ViewedAt  time.Time
}

func (MessageView) TableName() string {
	return "message_views"
}

type MessageStatus struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null;index"`
	Status    string `gorm:"type:varchar(20);not null"`
	UpdatedAt time.Time
}

func (MessageStatus) TableName() string {
	return "message_status"
}

type UserBlock struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BlockerID string `gorm:"type:uuid;not null;index"`
	BlockedID string `gorm:"type:uuid;not null;index"`
	Reason    string `gorm:"type:text"`
	CreatedAt time.Time
}

func (UserBlock) TableName() string {
	return "user_blocks"
}

type MessageAttachment struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MessageID string `gorm:"type:uuid;not null;index"`
	FileURL   string `gorm:"type:text;not null"`
	FileType  string `gorm:"type:varchar(20)"`
	FileSize  int64
	CreatedAt time.Time
}

type PaidCall struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CallerID      string `gorm:"type:uuid;not null;index"`
	ReceiverID    string `gorm:"type:uuid;not null;index"`
	CallType      string `gorm:"type:varchar(20);not null"`
	Status        string `gorm:"type:varchar(20);default:'ringing'"`
	PricePerMin   float64
	DurationSecs  int
	TotalCharged  float64
	StartedAt     *time.Time
	EndedAt       *time.Time
	CreatedAt     time.Time
}

type HostSchedule struct {
	ID                  string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID              string `gorm:"type:uuid;not null;index"`
	DayOfWeek           int
	StartTime           string `gorm:"type:varchar(10)"`
	EndTime             string `gorm:"type:varchar(10)"`
	SlotDurationMinutes int
	IsActive            bool `gorm:"default:true"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type HostScheduleException struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID        string    `gorm:"type:uuid;not null;index"`
	ExceptionDate time.Time
	Type          string    `gorm:"type:varchar(20)"`
	StartTime     string    `gorm:"type:varchar(10)"`
	EndTime       string    `gorm:"type:varchar(10)"`
	Reason        string    `gorm:"type:text"`
	CreatedAt     time.Time
}

type HostBookingType struct {
	ID                   string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID               string  `gorm:"type:uuid;not null;index"`
	Type                 string  `gorm:"type:varchar(30);not null"`
	Name                 string  `gorm:"type:varchar(100);not null"`
	Description          string  `gorm:"type:text"`
	PricePerMinute       float64
	MinDuration          int
	MaxDuration          int
	IsActive             bool    `gorm:"default:true"`
	AllowExtend          bool    `gorm:"default:false"`
	ExtendPricePerMinute *float64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Booking struct {
	ID                       string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BookingCode              string  `gorm:"type:varchar(50);unique"`
	HostID                   string  `gorm:"type:uuid;not null;index"`
	UserID                   string  `gorm:"type:uuid;not null;index"`
	BookingTypeID            string  `gorm:"type:uuid;not null"`
	ScheduledAt              time.Time
	DurationMinutes          int
	EndedAt                  *time.Time
	BasePrice                float64
	PlatformFee              float64
	ProcessingFee            float64
	TaxFee                   float64
	AgencyFee                float64
	TotalPrice               float64
	HostEarning              float64
	Status                   string  `gorm:"type:varchar(30);default:'pending'"`
	PaymentStatus            string  `gorm:"type:varchar(20)"`
	RoomID                   string  `gorm:"type:uuid"`
	JoinToken                string  `gorm:"type:varchar(255)"`
	UserNotes                string  `gorm:"type:text"`
	MeetingLatitude          *float64
	MeetingLongitude         *float64
	MeetingLocationName      string  `gorm:"type:varchar(255)"`
	IsRealtimeTrackingActive bool    `gorm:"default:false"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type Withdrawal struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string `gorm:"type:uuid;not null;index"`
	Amount    int64
	Fee       int64
	NetAmount int64
	Status    string `gorm:"type:varchar(20);default:'pending'"`
	Method    string `gorm:"type:varchar(30)"`
	AccountNo string `gorm:"type:varchar(50)"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type HostOffer struct {
	ID                  string   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID              string   `gorm:"type:uuid;not null;index"`
	OfferCode           string   `gorm:"type:varchar(50);unique"`
	Title               string   `gorm:"type:varchar(255);not null"`
	Description         string   `gorm:"type:text"`
	BookingTypeID       string   `gorm:"type:uuid"`
	OfferMode           string   `gorm:"type:varchar(20);default:'specific'"`
	SpecificAt          *time.Time
	SlotDurationMinutes int
	BasePricePerMinute  float64
	DiscountPercentage  float64
	FinalPricePerMinute float64
	MaxBookings         int      `gorm:"default:1"`
	BookingsMade        int      `gorm:"default:0"`
	MaxBookingsPerUser  int      `gorm:"default:1"`
	Status              string   `gorm:"type:varchar(20);default:'active'"`
	ExpiresAt           *time.Time
	IsAutoConfirm       bool     `gorm:"default:false"`
	Latitude            *float64
	Longitude           *float64
	LocationName        string   `gorm:"type:varchar(255)"`
	ShareLocationType   string   `gorm:"type:varchar(20);default:'none'"`
	ThumbnailURL        string   `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UserOffer struct {
	ID                      string   `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OfferCode               string   `gorm:"type:varchar(50);unique"`
	UserID                  string   `gorm:"type:uuid;not null;index"`
	HostID                  string   `gorm:"type:uuid;not null;index"`
	BookingTypeID           string   `gorm:"type:uuid"`
	OfferType               string   `gorm:"type:varchar(20);default:'standard'"`
	ProposedAt              time.Time
	ProposedDurationMinutes int
	ProposedPricePerMinute  float64
	TotalOfferAmount        float64
	Message                 string   `gorm:"type:text"`
	Status                  string   `gorm:"type:varchar(20);default:'pending'"`
	HostResponseAt          *time.Time
	HostMessage             string   `gorm:"type:text"`
	HostCounterPrice        *float64
	HostCounterAt           *time.Time
	ConvertedBookingID      string   `gorm:"type:uuid"`
	IsPrepaid               bool     `gorm:"default:false"`
	PrepaidAmount           float64  `gorm:"default:0"`
	Latitude                *float64
	Longitude               *float64
	LocationName            string   `gorm:"type:varchar(255)"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type StreamCoHost struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID   string     `gorm:"type:uuid;not null;index;uniqueIndex:idx_stream_seat"`
	UserID     string     `gorm:"type:uuid;not null;index"`
	SeatNumber int        `gorm:"not null;uniqueIndex:idx_stream_seat"`
	JoinedAt   time.Time  `gorm:"default:now()"`
	LeftAt     *time.Time
	IsMuted    bool       `gorm:"default:false"`
	IsVideoOn  bool       `gorm:"default:true"`
}

func (StreamCoHost) TableName() string {
	return "stream_co_hosts"
}

type StreamModerator struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID       string    `gorm:"type:uuid;not null;index"`
	UserID         string    `gorm:"type:uuid;not null;index"`
	Role           string    `gorm:"type:varchar(20);default:'moderator'"`
	GrantedBy      string    `gorm:"type:uuid;not null"`
	CanKick        bool      `gorm:"default:false"`
	CanMute        bool      `gorm:"default:true"`
	CanPinMessage  bool      `gorm:"default:true"`
	GrantedAt      time.Time `gorm:"default:now()"`
}

func (StreamModerator) TableName() string {
	return "stream_moderators"
}

type PKBattle struct {
	ID                  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BattleCode          string     `gorm:"type:varchar(50);unique;not null"`
	Mode                string     `gorm:"type:varchar(20);default:'1v1'"`
	HostAID             string     `gorm:"type:uuid;not null;index"`
	HostBID             string     `gorm:"type:uuid;not null;index"`
	TeamAName           string     `gorm:"type:varchar(50)"`
	TeamBName           string     `gorm:"type:varchar(50)"`
	DurationSeconds     int        `gorm:"default:300"`
	Theme               string     `gorm:"type:varchar(50)"`
	ScoreA              float64    `gorm:"type:decimal(15,2);default:0"`
	ScoreB              float64    `gorm:"type:decimal(15,2);default:0"`
	TotalGiftValueA     float64    `gorm:"type:decimal(15,2);default:0"`
	TotalGiftValueB     float64    `gorm:"type:decimal(15,2);default:0"`
	WinnerID            *string    `gorm:"type:uuid"`
	IsDraw              bool       `gorm:"default:false"`
	WinMargin           float64    `gorm:"type:decimal(15,2);default:0"`
	PunishmentType      string     `gorm:"type:varchar(50)"`
	RewardType          string     `gorm:"type:varchar(50)"`
	Status              string     `gorm:"type:varchar(20);default:'invited'"`
	InvitedAt           time.Time  `gorm:"default:now()"`
	StartedAt           *time.Time
	EndedAt             *time.Time
	CreatedAt           time.Time
}

func (PKBattle) TableName() string {
	return "pk_battles"
}

type PKBattleVote struct {
	ID               string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PKBattleID       string    `gorm:"type:uuid;not null;index"`
	SenderID         string    `gorm:"type:uuid;not null;index"`
	RecipientSide    string    `gorm:"type:varchar(1);not null"` // 'a' or 'b'
	GiftID           string    `gorm:"type:uuid;not null"`
	Quantity         int       `gorm:"default:1"`
	TotalValueIDR    float64   `gorm:"type:decimal(15,2);not null"`
	ComboCount       int       `gorm:"default:1"`
	IsCriticalHit    bool      `gorm:"default:false"`
	SentAt           time.Time `gorm:"default:now()"`
}

func (PKBattleVote) TableName() string {
	return "pk_battle_votes"
}

type LuckyBoxDrop struct {
	ID              string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID        string     `gorm:"type:uuid;not null;index"`
	HostID          string     `gorm:"type:uuid;not null;index"`
	BoxType         string     `gorm:"type:varchar(20);not null"` // 'bronze', 'silver', 'gold', 'diamond'
	PriceIDR        float64    `gorm:"type:decimal(12,2);not null"`
	RewardGiftID    *string    `gorm:"type:uuid"`
	RewardQuantity  int        `gorm:"default:0"`
	IsJackpot       bool       `gorm:"default:false"`
	OpenedBy        *string    `gorm:"type:uuid"`
	OpenedAt        *time.Time
	CreatedAt       time.Time
}

func (LuckyBoxDrop) TableName() string {
	return "lucky_box_drops"
}

type HostLevel struct {
	Level                    int     `gorm:"primaryKey"`
	Name                     string  `gorm:"type:varchar(50);not null"`
	MinXP                    int64   `gorm:"not null"`
	MaxResolution            string  `gorm:"type:varchar(10)"`
	CanPK                    bool    `gorm:"default:false"`
	MaxCoHosts               int     `gorm:"default:0"`
	CanGoLiveMobile          bool    `gorm:"default:true"`
	CanGoLiveOBS             bool    `gorm:"default:false"`
	DailyPKLimit             int     `gorm:"default:3"`
	CommissionBoostPercent   float64 `gorm:"type:decimal(5,2);default:0"`
	BadgeURL                 string  `gorm:"type:text"`
}

func (HostLevel) TableName() string {
	return "host_levels"
}

type UserLevel struct {
	Level                     int     `gorm:"primaryKey"`
	Name                      string  `gorm:"type:varchar(50);not null"`
	MinXP                     int64   `gorm:"not null"`
	BadgeURL                  string  `gorm:"type:text"`
	ChatColor                 string  `gorm:"type:varchar(7)"`
	CanEnterPaidRoom          bool    `gorm:"default:false"`
	DailyGiftDiscountPercent float64 `gorm:"type:decimal(5,2);default:0"`
}

func (UserLevel) TableName() string {
	return "user_levels"
}

type UserXPLog struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	Action    string    `gorm:"type:varchar(50);not null"` // 'watch', 'gift_sent', 'gift_received', 'chat', 'share', 'pk_win'
	XPAmount  int       `gorm:"not null"`
	ContextID string    `gorm:"type:uuid"`
	CreatedAt time.Time `gorm:"default:now()"`
}

func (UserXPLog) TableName() string {
	return "user_xp_logs"
}

type Family struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string    `gorm:"type:varchar(100);not null"`
	Code           string    `gorm:"type:varchar(20);unique;not null"`
	LeaderID       string    `gorm:"type:uuid;not null;index"`
	Description    string    `gorm:"type:text"`
	LogoURL        string    `gorm:"type:text"`
	BannerURL      string    `gorm:"type:text"`
	TotalMembers   int       `gorm:"default:0"`
	TotalXP        int64     `gorm:"default:0"`
	WeeklyPKWins   int       `gorm:"default:0"`
	MinHostLevel   int       `gorm:"default:5"`
	EntryRequirement string  `gorm:"type:varchar(50);default:'open'"`
	CreatedAt      time.Time
}

func (Family) TableName() string {
	return "families"
}

type FamilyMember struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	FamilyID       string    `gorm:"type:uuid;not null;index"`
	UserID         string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_family_member"`
	Role           string    `gorm:"type:varchar(20);default:'member'"` // 'member', 'elder', 'co_leader'
	JoinedAt       time.Time `gorm:"default:now()"`
	ContributionXP int64     `gorm:"default:0"`
}

func (FamilyMember) TableName() string {
	return "family_members"
}

type TrendingStream struct {
	ID                 string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID           string    `gorm:"type:uuid;not null;index"`
	Category           string    `gorm:"type:varchar(50)"`
	ScoreViewerCount   float64   `gorm:"type:decimal(10,2);default:0"`
	ScoreGiftVelocity  float64   `gorm:"type:decimal(10,2);default:0"`
	ScoreNewHostBoost  float64   `gorm:"type:decimal(10,2);default:0"`
	ScorePKActive      float64   `gorm:"type:decimal(10,2);default:0"`
	ScoreTotal         float64   `gorm:"type:decimal(10,2);default:0"`
	Rank               int       `gorm:"type:integer"`
	Bucket             string    `gorm:"type:varchar(20)"`
	CalculatedAt       time.Time `gorm:"default:now()"`
	ExpiresAt          time.Time
}

func (TrendingStream) TableName() string {
	return "trending_streams"
}

type HostTopFan struct {
	ID                string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID            string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_host_fan"`
	UserID            string    `gorm:"type:uuid;not null;index;uniqueIndex:idx_host_fan"`
	Period            string    `gorm:"type:varchar(20);not null;uniqueIndex:idx_host_fan"` // 'daily', 'weekly', 'monthly', 'all_time'
	TotalGiftValueIDR float64   `gorm:"type:decimal(15,2);default:0"`
	Rank              int       `gorm:"type:integer;not null"`
	BadgeType         string    `gorm:"type:varchar(20)"`
	CalculatedAt      time.Time `gorm:"default:now()"`
}

func (HostTopFan) TableName() string {
	return "host_top_fans"
}


type LiveSchedule struct {
	ID                     string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID                 string     `gorm:"type:uuid;not null;index"`
	Title                  string     `gorm:"type:varchar(200);not null"`
	Description            string     `gorm:"type:text"`
	Category               string     `gorm:"type:varchar(50)"`
	ThumbnailURL           string     `gorm:"type:text"`
	ScheduleType           string     `gorm:"type:varchar(20);default:'one_time'"` // 'one_time', 'recurring'
	ScheduledAt            *time.Time `gorm:"index"`
	RecurrenceRule         string     `gorm:"type:text"`
	RecurrenceStartDate    *time.Time `gorm:"type:date"`
	RecurrenceEndDate      *time.Time `gorm:"type:date"`
	RecurrenceTime         string     `gorm:"type:time"`
	Timezone               string     `gorm:"type:varchar(50);default:'Asia/Jakarta'"`
	ExpectedDurationMinutes int        `gorm:"default:60"`
	Status                 string     `gorm:"type:varchar(20);default:'scheduled';index"` // scheduled, live, completed, cancelled, expired
	IsCancelled            bool       `gorm:"default:false"`
	CancelledAt            *time.Time
	CancellationReason     string     `gorm:"type:text"`
	ActualStreamID         *string    `gorm:"type:uuid"`
	WentLiveAt             *time.Time
	MaxWaitRoomUsers       int        `gorm:"default:500"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (LiveSchedule) TableName() string {
	return "live_schedules"
}

type LiveScheduleOccurrence struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ScheduleID       string     `gorm:"type:uuid;not null;index"`
	HostID           string     `gorm:"type:uuid;not null;index"`
	OccurrenceDate   time.Time  `gorm:"type:date;not null;uniqueIndex:idx_sched_occ"`
	OccurrenceStartAt time.Time  `gorm:"not null;uniqueIndex:idx_sched_occ;index"`
	OccurrenceEndAt   *time.Time
	Status           string     `gorm:"type:varchar(20);default:'upcoming';index"` // upcoming, live, completed, missed, cancelled
	ActualStreamID   *string    `gorm:"type:uuid"`
	WaitRoomOpenedAt *time.Time
	CreatedAt        time.Time
}

func (LiveScheduleOccurrence) TableName() string {
	return "schedule_occurrences"
}

type UserScheduleReminder struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           string     `gorm:"type:uuid;not null;uniqueIndex:idx_user_sched"`
	ScheduleID       string     `gorm:"type:uuid;not null;uniqueIndex:idx_user_sched"`
	Remind24h        bool       `gorm:"default:true"`
	Remind1h         bool       `gorm:"default:true"`
	Remind15m        bool       `gorm:"default:true"`
	RemindLiveStart  bool       `gorm:"default:true"`
	PushEnabled      bool       `gorm:"default:true"`
	EmailEnabled     bool       `gorm:"default:false"`
	SMSEnabled       bool       `gorm:"default:false"`
	IsActive         bool       `gorm:"default:true;index"`
	UnsubscribedAt   *time.Time
	JoinedWaitRoomAt *time.Time
	LeftWaitRoomAt   *time.Time
	CreatedAt        time.Time
}

func (UserScheduleReminder) TableName() string {
	return "user_schedule_reminders"
}

type ReminderLog struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReminderID   string    `gorm:"type:uuid;not null;index"`
	ReminderType string    `gorm:"type:varchar(20);not null"` // '24h', '1h', '15m', 'live_start'
	Channel      string    `gorm:"type:varchar(20);not null"` // 'push', 'ws', 'email', 'sms'
	SentAt       time.Time `gorm:"default:now()"`
	DeliveredAt  *time.Time
	OpenedAt     *time.Time
	IsSuccess    bool      `gorm:"default:true"`
	ErrorMessage string    `gorm:"type:text"`
}

func (ReminderLog) TableName() string {
	return "reminder_logs"
}

type WaitRoom struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OccurrenceID     string     `gorm:"type:uuid;not null;unique;index"`
	HostID           string     `gorm:"type:uuid;not null;index"`
	Status           string     `gorm:"type:varchar(20);default:'waiting'"` // waiting, opening_soon, live_started, closed
	OpenedAt         *time.Time
	ClosedAt         *time.Time
	CurrentUserCount int        `gorm:"default:0"`
	PeakUserCount    int        `gorm:"default:0"`
	CreatedAt        time.Time
}

func (WaitRoom) TableName() string {
	return "wait_rooms"
}

type WaitRoomMessage struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WaitRoomID string    `gorm:"type:uuid;not null;index"`
	UserID     string    `gorm:"type:uuid;not null"`
	Content    string    `gorm:"type:text;not null"`
	MessageType string   `gorm:"type:varchar(20);default:'text'"`
	CreatedAt  time.Time `gorm:"default:now()"`
}

func (WaitRoomMessage) TableName() string {
	return "wait_room_messages"
}

type HostScheduleStat struct {
	ID                        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID                    string     `gorm:"type:uuid;not null;index"`
	ScheduleID                *string    `gorm:"type:uuid"`
	OccurrenceID              *string    `gorm:"type:uuid"`
	TotalRemindersSent        int        `gorm:"default:0"`
	TotalRemindersOpened      int        `gorm:"default:0"`
	WaitRoomJoined            int        `gorm:"default:0"`
	WaitRoomToLiveConversion  int        `gorm:"default:0"`
	LiveStartViewers          int        `gorm:"default:0"`
	ScheduledAt               *time.Time
	CreatedAt                 time.Time
}

// Fitur 6: Auto-Moderation & Safety Engine models
type ModerationRule struct {
	ID                    string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RuleCode              string `gorm:"type:varchar(50);unique;not null"`
	Name                  string `gorm:"type:varchar(100);not null"`
	Category              string `gorm:"type:varchar(50);not null"` // 'content', 'behavior', 'fraud', 'spam'
	ConditionType         string `gorm:"type:varchar(50);not null"` // 'nsfw_score', 'repeated_message', 'gift_velocity', 'toxicity_score', 'ip_cluster', 'caps_ratio', 'link_count'
	Threshold             float64 `gorm:"type:decimal(10,4);not null"`
	TimeWindowSeconds     int    `gorm:"default:0"`
	Action                string `gorm:"type:varchar(50);not null"` // 'warn', 'mute', 'kick', 'ban_temp', 'ban_perm', 'blur_image', 'flag_review'
	ActionDurationSeconds *int
	EscalationRuleID      *string `gorm:"type:uuid"`
	MaxStrikes            int    `gorm:"default:3"`
	AppliesTo             string `gorm:"type:varchar(50);default:'all'"`
	IsActive              bool   `gorm:"default:true"`
	Priority              int    `gorm:"default:0"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (ModerationRule) TableName() string {
	return "moderation_rules"
}

type UserModerationState struct {
	ID                       string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                   string     `gorm:"type:uuid;unique;not null"`
	TotalStrikes             int        `gorm:"default:0"`
	CurrentBanLevel          int        `gorm:"default:0"` // 0=none, 1=mute, 2=kick, 3=tempban, 4=permaban
	IsMuted                  bool       `gorm:"default:false"`
	MutedUntil               *time.Time
	IsBanned                 bool       `gorm:"default:false"`
	BannedUntil              *time.Time
	BanReason                string     `gorm:"type:text"`
	LastStrikeAt             *time.Time
	LastStrikeRuleID         *string    `gorm:"type:uuid"`
	ConsecutiveSameRuleCount int        `gorm:"default:0"`
	SuspectedBotScore        float64    `gorm:"type:decimal(5,2);default:0"`
	DeviceFingerprintHash    string     `gorm:"type:varchar(255)"`
	IPClusterID              *string    `gorm:"type:uuid"`
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (UserModerationState) TableName() string {
	return "user_moderation_state"
}

type ModerationLog struct {
	ID                    string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                string `gorm:"type:uuid;not null"`
	StreamID              *string `gorm:"type:uuid"`
	ConversationID        *string `gorm:"type:uuid"`
	RuleID                *string `gorm:"type:uuid"`
	TriggerType           string `gorm:"type:varchar(50);not null"` // 'auto', 'manual', 'appeal', 'system'
	EvidenceType          string `gorm:"type:varchar(50);not null"` // 'chat_message', 'image', 'gift_pattern', 'ip_cluster', 'device_fingerprint'
	EvidenceContent       string `gorm:"type:text"`
	EvidenceMetadata      []byte `gorm:"type:jsonb"`
	ActionTaken           string `gorm:"type:varchar(50);not null"`
	ActionDurationSeconds *int
	ActionExecutedAt      time.Time `gorm:"default:now()"`
	ActionExecutedBy      *string `gorm:"type:uuid"`
	RelatedMessageID      *string `gorm:"type:uuid"`
	RelatedImageURL       string `gorm:"type:text"`
	IsAppealed            bool   `gorm:"default:false"`
	AppealStatus          string `gorm:"type:varchar(20)"` // 'pending', 'approved', 'rejected'
	CreatedAt             time.Time
}

func (ModerationLog) TableName() string {
	return "moderation_logs"
}

type ImageModerationQueue struct {
	ID               string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ImageURL         string     `gorm:"type:text;not null"`
	SourceType       string     `gorm:"type:varchar(50);not null"` // 'chat', 'story', 'avatar', 'stream_thumbnail', 'clip'
	SourceID         string     `gorm:"type:uuid;not null"`
	Status           string     `gorm:"type:varchar(20);default:'queued'"` // queued, scanning, completed, failed
	Provider         string     `gorm:"type:varchar(50)"`
	NSFWScore        *float64   `gorm:"type:decimal(5,4)"`
	IsNSFW           *bool
	ModerationLabels []byte     `gorm:"type:jsonb"`
	ActionTaken      string     `gorm:"type:varchar(50)"`
	BlurredURL       string     `gorm:"type:text"`
	CreatedAt        time.Time
	CompletedAt      *time.Time
}

func (ImageModerationQueue) TableName() string {
	return "image_moderation_queue"
}

type ChatPatternSnapshot struct {
	ID                    string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID                string `gorm:"type:uuid;not null"`
	StreamID              *string `gorm:"type:uuid"`
	WindowStart           time.Time `gorm:"not null"`
	WindowEnd             time.Time `gorm:"not null"`
	MessageCount          int    `gorm:"default:0"`
	UniqueContentCount    int    `gorm:"default:0"`
	RepeatedContentCount  int    `gorm:"default:0"`
	CapsRatio             float64 `gorm:"type:decimal(5,2)"`
	LinkCount             int    `gorm:"default:0"`
	EmojiCount            int    `gorm:"default:0"`
	IsSpamDetected        bool   `gorm:"default:false"`
	SpamScore             float64 `gorm:"type:decimal(5,2);default:0"`
	CreatedAt             time.Time
}

func (ChatPatternSnapshot) TableName() string {
	return "chat_pattern_snapshots"
}

type GiftFraudAlert struct {
	ID                 string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AlertType          string    `gorm:"type:varchar(50);not null"` // 'velocity_spike', 'circular_gifting', 'new_account_burst', 'ip_cluster'
	PrimaryUserID      string    `gorm:"type:uuid;not null"`
	SecondaryUserID    *string   `gorm:"type:uuid"`
	StreamID           *string   `gorm:"type:uuid"`
	TotalGiftValueIDR  float64   `gorm:"type:decimal(15,2)"`
	TransactionCount   int
	TimeWindowSeconds  int
	IPAddress          string    `gorm:"type:varchar(45)"`
	DeviceFingerprint  string    `gorm:"type:varchar(255)"`
	AccountClusterID   *string   `gorm:"type:uuid"`
	Status             string    `gorm:"type:varchar(20);default:'open'"` // open, under_review, confirmed_fraud, false_positive
	ReviewedBy         *string   `gorm:"type:uuid"`
	ReviewedAt         *time.Time
	Notes              string    `gorm:"type:text"`
	CreatedAt          time.Time
}

func (GiftFraudAlert) TableName() string {
	return "gift_fraud_alerts"
}

type ModerationWordlist struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Word          string `gorm:"type:varchar(100);unique;not null"`
	SeverityLevel int    `gorm:"default:1"` // 1=mild, 2=moderate, 3=severe
	Language      string `gorm:"type:varchar(10);default:'id'"`
	IsRegex       bool   `gorm:"default:false"`
	CreatedAt     time.Time
}

func (ModerationWordlist) TableName() string {
	return "moderation_wordlist"
}

// RunAutoMigrate creates all tables using GORM AutoMigrate
func RunAutoMigrate(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&Role{}, &Permission{}, &RolePermission{}, &User{},
		&Session{}, &Account{}, &Verification{},
		&Story{}, &StoryView{}, &Comment{}, &CommentLike{}, &Like{},
		&LiveRoom{}, &LiveRoomParticipant{}, &LiveMessage{},
		&Stream{}, &StreamSession{}, &VODMedia{},
		&Wallet{}, &Transaction{}, &Gift{}, &GiftTransaction{},
		&Agency{}, &HostApplication{}, &DuitkuPayment{},
		&CryptoWallet{}, &CryptoMasterWallet{}, &CryptoDepositAddress{},
		&CryptoTransaction{}, &CryptoExchangeRate{}, &CryptoWithdrawalWhitelist{},
		&Conversation{}, &ConversationParticipant{}, &PrivateMessage{},
		&MessageReaction{}, &MessageView{}, &MessageStatus{}, &UserBlock{}, &MessageAttachment{},
		&PaidCall{},
		&HostSchedule{}, &HostScheduleException{}, &HostBookingType{},
		&Booking{}, &Withdrawal{},
		&HostOffer{}, &UserOffer{}, &BookingLocationLog{},
		
		// Bigo/Mango style Stream models
		&StreamCoHost{}, &StreamModerator{},
		&PKBattle{}, &PKBattleVote{}, &LuckyBoxDrop{},
		&HostLevel{}, &UserLevel{}, &UserXPLog{},
		&Family{}, &FamilyMember{}, &TrendingStream{}, &HostTopFan{},

		// Fitur 7: Schedule & Wait Room models
		&LiveSchedule{}, &LiveScheduleOccurrence{}, &UserScheduleReminder{},
		&ReminderLog{}, &WaitRoom{}, &WaitRoomMessage{}, &HostScheduleStat{},

		// Fitur 6: Auto-Moderation & Safety Engine
		&ModerationRule{}, &UserModerationState{}, &ModerationLog{},
		&ImageModerationQueue{}, &ChatPatternSnapshot{}, &GiftFraudAlert{},
		&ModerationWordlist{},
	)
}
