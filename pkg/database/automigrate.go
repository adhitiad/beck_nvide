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
	Username            string     `gorm:"type:varchar(50);unique;not null"`
	Email               string     `gorm:"type:varchar(255);unique;not null"`
	PasswordHash        string     `gorm:"type:varchar(255);not null"`
	RoleID              string     `gorm:"type:uuid;not null"`
	AvatarURL           string     `gorm:"type:text"`
	IsVerified          bool       `gorm:"default:false"`
	VerificationToken   string     `gorm:"type:varchar(255)"`
	ResetToken          string     `gorm:"type:varchar(255)"`
	ResetTokenExpiresAt *time.Time
	LastLoginAt         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
	ExpiresAt time.Time
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

type ChatRoom struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string `gorm:"type:varchar(100)"`
	Type      string `gorm:"type:varchar(20);default:'group'"`
	CreatedAt time.Time
}

type ChatRoomMember struct {
	ID       string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID   string `gorm:"type:uuid;not null;index"`
	UserID   string `gorm:"type:uuid;not null"`
	JoinedAt time.Time
}

type Message struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RoomID    string `gorm:"type:uuid;not null;index"`
	UserID    string `gorm:"type:uuid;not null"`
	Content   string `gorm:"type:text;not null"`
	Type      string `gorm:"type:varchar(20);default:'text'"`
	CreatedAt time.Time
}

type Stream struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	HostID      string `gorm:"type:uuid;not null;index"`
	Title       string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	StreamKey   string `gorm:"type:varchar(255);unique"`
	Status      string `gorm:"type:varchar(20);default:'idle'"`
	ViewerCount int    `gorm:"default:0"`
	StartedAt   *time.Time
	EndedAt     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StreamSession struct {
	ID            string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StreamID      string `gorm:"type:uuid;not null;index"`
	ViewerCount   int    `gorm:"default:0"`
	PeakViewers   int    `gorm:"default:0"`
	DurationSecs  int    `gorm:"default:0"`
	GiftsReceived int    `gorm:"default:0"`
	Revenue       float64
	StartedAt     time.Time
	EndedAt       *time.Time
}

type VODMedia struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       string `gorm:"type:uuid;not null;index"`
	Title        string `gorm:"type:varchar(255);not null"`
	Description  string `gorm:"type:text"`
	FilePath     string `gorm:"type:text;not null"`
	ThumbnailURL string `gorm:"type:text"`
	Duration     int
	FileSize     int64
	MimeType     string `gorm:"type:varchar(50)"`
	Status       string `gorm:"type:varchar(20);default:'processing'"`
	ViewCount    int    `gorm:"default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string `gorm:"type:varchar(100);not null"`
	IconURL   string `gorm:"type:text"`
	Price     int64  `gorm:"not null"`
	Category  string `gorm:"type:varchar(50)"`
	IsActive  bool   `gorm:"default:true"`
	CreatedAt time.Time
}

type GiftTransaction struct {
	ID         string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	GiftID     string `gorm:"type:uuid;not null"`
	SenderID   string `gorm:"type:uuid;not null;index"`
	ReceiverID string `gorm:"type:uuid;not null;index"`
	StreamID   string `gorm:"type:uuid"`
	Quantity   int    `gorm:"default:1"`
	TotalPrice int64
	CreatedAt  time.Time
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
	UserID          string `gorm:"type:uuid;not null;index"`
	MerchantOrderID string `gorm:"type:varchar(100);unique"`
	Reference       string `gorm:"type:varchar(100)"`
	Amount          int64
	PaymentMethod   string `gorm:"type:varchar(50)"`
	Status          string `gorm:"type:varchar(20);default:'pending'"`
	PaymentURL      string `gorm:"type:text"`
	CallbackData    string `gorm:"type:text"`
	ExpiresAt       *time.Time
	PaidAt          *time.Time
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

type PrivateChatConversation struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	User1ID   string `gorm:"type:uuid;not null;index"`
	User2ID   string `gorm:"type:uuid;not null;index"`
	Status    string `gorm:"type:varchar(20);default:'active'"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PrivateChatMessage struct {
	ID             string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID string `gorm:"type:uuid;not null;index"`
	SenderID       string `gorm:"type:uuid;not null"`
	Content        string `gorm:"type:text"`
	MessageType    string `gorm:"type:varchar(20);default:'text'"`
	IsRead         bool   `gorm:"default:false"`
	CreatedAt      time.Time
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


// RunAutoMigrate creates all tables using GORM AutoMigrate
func RunAutoMigrate(dsn string) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	return db.AutoMigrate(
		&Role{}, &Permission{}, &RolePermission{},
		&User{}, &RefreshToken{},
		&Story{}, &StoryView{}, &Comment{}, &CommentLike{}, &Like{},
		&ChatRoom{}, &ChatRoomMember{}, &Message{},
		&Stream{}, &StreamSession{}, &VODMedia{},
		&Wallet{}, &Transaction{}, &Gift{}, &GiftTransaction{},
		&Agency{}, &HostApplication{}, &DuitkuPayment{},
		&CryptoWallet{}, &CryptoMasterWallet{}, &CryptoDepositAddress{},
		&CryptoTransaction{}, &CryptoExchangeRate{}, &CryptoWithdrawalWhitelist{},
		&PrivateChatConversation{}, &PrivateChatMessage{}, &MessageAttachment{},
		&PaidCall{},
		&HostSchedule{}, &HostScheduleException{}, &HostBookingType{},
		&Booking{}, &Withdrawal{},
		&HostOffer{}, &UserOffer{}, &BookingLocationLog{},
	)
}
