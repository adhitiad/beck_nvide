package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"nvide-live/internal/domain"
)

func main() {
	// Definisi semua domain types yang mau di-generate
	types := map[string]interface{}{
		"User":                    domain.User{},
		"Role":                    domain.Role{},
		"Permission":              domain.Permission{},
		"Story":                   domain.Story{},
		"StoryView":               domain.StoryView{},
		"Comment":                 domain.Comment{},
		"Like":                    domain.Like{},
		"Message":                 domain.Message{},
		"Conversation":            domain.Conversation{},
		"ConversationParticipant": domain.ConversationParticipant{},
		"PrivateMessage":          domain.PrivateMessage{},
		"MessageAttachment":       domain.MessageAttachment{},
		"MessageReaction":         domain.MessageReaction{},
		"Stream":                  domain.Stream{},
		"StreamSession":           domain.StreamSession{},
		"VODMedia":                domain.VODMedia{},
		"Gift":                    domain.Gift{},
		"GiftTransaction":         domain.GiftTransaction{},
		"Wallet":                  domain.Wallet{},
		"Transaction":             domain.Transaction{},
		"Withdrawal":              domain.Withdrawal{},
		"WithdrawalFeeAudit":      domain.WithdrawalFeeAudit{},
		"HostCallRate":            domain.HostCallRate{},
		"CallSession":             domain.CallSession{},
		"Booking":                 domain.Booking{},
		"HostSchedule":            domain.HostSchedule{},
		"LiveSchedule":            domain.LiveSchedule{},
		"LiveScheduleOccurrence":  domain.LiveScheduleOccurrence{},
		"HostOffer":               domain.HostOffer{},
		"UserOffer":               domain.UserOffer{},
		"PKBattle":                domain.PKBattle{},
		"CryptoDepositAddress":    domain.CryptoDepositAddress{},
		"CryptoTransaction":       domain.CryptoTransaction{},
		"CryptoExchangeRate":      domain.CryptoExchangeRate{},
		"ModerationRule":          domain.ModerationRule{},
		"ModerationLog":           domain.ModerationLog{},
	}

	var output strings.Builder

	// Header
	output.WriteString("// AUTO-GENERATED from Go domain types\n")
	output.WriteString("// DO NOT EDIT MANUALLY - Run: go run tools/generate_types.go\n")
	output.WriteString("// Generated at: " + time.Now().Format(time.RFC3339) + "\n\n")

	// API Base URL
	output.WriteString("export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';\n\n")

	// Generate setiap type
	for name, v := range types {
		output.WriteString(generateTypeScriptInterface(name, v))
		output.WriteString("\n")
	}

	// Generate API endpoint types
	output.WriteString(generateEndpointTypes())

	// Generate WebSocket message types
	output.WriteString(generateWebSocketTypes())

	// Generate utility types
	output.WriteString(generateUtilityTypes())

	// Write ke file
	outputPath := "../front_nvide/lib/types/api.ts"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	// Ensure parent directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directories: %v\n", err)
		os.Exit(1)
	}

	err := os.WriteFile(outputPath, []byte(output.String()), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d types to %s\n", len(types), outputPath)
}

func generateTypeScriptInterface(name string, v interface{}) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("export interface %s {\n", name))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Parse json tag
		fieldName := field.Name
		isOptional := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					isOptional = true
				}
			}
		}

		// Convert Go type ke TypeScript
		tsType := goTypeToTypeScript(field.Type)
		if isOptional || field.Type.Kind() == reflect.Ptr {
			sb.WriteString(fmt.Sprintf("  %s?: %s;\n", fieldName, tsType))
		} else {
			sb.WriteString(fmt.Sprintf("  %s: %s;\n", fieldName, tsType))
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

func goTypeToTypeScript(t reflect.Type) string {
	// Handle pointer
	if t.Kind() == reflect.Ptr {
		return goTypeToTypeScript(t.Elem()) + " | null"
	}

	// Handle slice/array
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		elemType := goTypeToTypeScript(t.Elem())
		return elemType + "[]"
	}

	// Handle map
	if t.Kind() == reflect.Map {
		keyType := goTypeToTypeScript(t.Key())
		valType := goTypeToTypeScript(t.Elem())
		return fmt.Sprintf("Record<%s, %s>", keyType, valType)
	}

	// Basic types
	switch t.Kind() {
	case reflect.String:
		// Cek kalau enum
		if t.Name() == "RoleType" || strings.HasSuffix(t.Name(), "Status") {
			return "string /* enum */"
		}
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Struct:
		// Time
		if t.String() == "time.Time" {
			return "string /* ISO8601 */"
		}
		// Nested struct - return any untuk simplicity
		// Atau bisa recurse untuk complex types
		return "any"
	case reflect.Interface:
		return "any"
	default:
		return "any"
	}
}

func generateEndpointTypes() string {
	return `
// ==========================================
// API REQUEST/RESPONSE TYPES
// ==========================================

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
  details?: Record<string, any>;
  retry_after?: number;
}

export interface PaginationMeta {
  cursor?: string;
  has_more: boolean;
  total_count?: number;
}

export interface PaginatedResponse<T> extends ApiResponse<T[]> {
  meta: PaginationMeta;
}

// Auth
export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
  referral_code?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

// Stories
export interface CreateStoryRequest {
  media_url: string;
  media_type: "image" | "video";
  caption?: string;
}

// Comments
export interface CreateCommentRequest {
  content_id: string;
  content_type: "stream" | "story" | "vod";
  text: string;
  parent_id?: string;
}

// Chat
export interface SendMessageRequest {
  room_id: string;
  type: "text" | "gift";
  content: string;
  metadata?: Record<string, any>;
}

export interface SendPrivateMessageRequest {
  content: string;
  type?: "text" | "image" | "voice" | "gift";
  disappear_mode?: "none" | "view_once" | "7s" | "1m" | "1h" | "24h" | "7d";
  reply_to_message_id?: string;
  attachments?: Array<{
    file_url: string;
    file_type: string;
  }>;
}

export interface ToggleReactionRequest {
  emoji: string;
}

// Streaming
export interface CreateStreamRequest {
  title: string;
  description?: string;
  category?: string;
  room_mode?: "public" | "password" | "paid" | "members" | "private";
  room_password?: string;
  entry_fee_idr?: number;
  min_level_to_enter?: number;
  tags?: string[];
  max_resolution?: "480p" | "720p" | "1080p";
}

export interface SwitchRoomModeRequest {
  room_mode: "public" | "password" | "paid" | "members" | "private";
  room_password?: string;
  entry_fee_idr?: number;
  min_level_to_enter?: number;
}

// Gifts
export interface SendGiftRequest {
  stream_id: string;
  gift_id: string;
  quantity: number;
  combo_type?: "x1" | "x10" | "x66" | "x188" | "x520" | "x1314" | "x3344" | "x9999";
}

// Wallet
export interface WithdrawalPreviewRequest {
  amount: number;
}

export interface SubmitWithdrawalRequest {
  amount: number;
  payment_method: "bank_transfer" | "crypto" | "e_wallet";
  bank_account_info?: {
    bank_name: string;
    account_number: string;
    account_holder: string;
  };
  idempotency_key: string;
}

// Booking
export interface RequestBookingRequest {
  host_id: string;
  scheduled_at: string;
  duration_minutes: number;
  user_notes?: string;
}

export interface SetCallRatesRequest {
  voice_call_rate_idr: number;
  video_call_rate_idr: number;
  is_enabled?: boolean;
}

// PK Battle
export interface InvitePKRequest {
  opponent_host_id: string;
  duration?: 180 | 300 | 600;
  theme?: "gift_war" | "popularity" | "task_challenge";
}

// Crypto
export interface CryptoWithdrawalRequest {
  chain: "SOL" | "BTC" | "USDT_ERC20" | "USDT_TRC20" | "USDT_BEP20";
  asset: "SOL" | "BTC" | "USDT";
  amount_crypto: number;
  to_address: string;
}

// Moderation
export interface AppealRequest {
  log_id: string;
  reason: string;
}

`
}

func generateWebSocketTypes() string {
	return `
// ==========================================
// WEBSOCKET MESSAGE TYPES
// ==========================================

export interface WSMessage {
  event: string;
  room_id?: string;
  user_id?: string;
  payload: any;
  timestamp: number;
  node_id?: string;
}

// Stream Chat Events
export type StreamChatEvent = 
  | { event: "message:new"; payload: Message }
  | { event: "message:status"; payload: { message_id: string; status: "delivered" | "read" } }
  | { event: "gift:received"; payload: GiftTransaction }
  | { event: "viewer:join"; payload: { user_id: string; viewer_count: number } }
  | { event: "viewer:leave"; payload: { user_id: string; viewer_count: number } }
  | { event: "stream:ended"; payload: { stream_id: string; reason: string } };

// Private Chat Events
export type PrivateChatEvent =
  | { event: "message:new"; payload: PrivateMessage }
  | { event: "message:status"; payload: { message_id: string; status: "delivered" | "read" } }
  | { event: "message:expired"; payload: { message_id: string } }
  | { event: "typing:start"; payload: { user_id: string } }
  | { event: "typing:stop"; payload: { user_id: string } }
  | { event: "screenshot:alert"; payload: { by_user_id: string } }
  | { event: "reaction:update"; payload: MessageReaction };

// Call Events
export type CallEvent =
  | { event: "call:request"; payload: { host_id: string; type: "voice" | "video" } }
  | { event: "call:incoming"; payload: { session_id: string; caller: any; rate_idr: number; type: string } }
  | { event: "call:accept"; payload: { session_id: string } }
  | { event: "call:start"; payload: { session_id: string; started_at: string } }
  | { event: "call:tick"; payload: { minute: number; charged: number; remaining_balance: number } }
  | { event: "call:end"; payload: { session_id: string } }
  | { event: "call:ended"; payload: CallSession };

// PK Battle Events
export type PKBattleEvent =
  | { event: "pk:invite"; payload: { battle_id: string; inviter: any } }
  | { event: "pk:started"; payload: PKBattle }
  | { event: "pk:score_update"; payload: { score_a: number; score_b: number; time_remaining: number } }
  | { event: "pk:combo_bonus"; payload: { side: "a" | "b"; multiplier: number; duration: number } }
  | { event: "pk:ended"; payload: { winner_id?: string; is_draw: boolean; punishment?: string } };

// WebRTC Signaling
export type WebRTCSignal =
  | { type: "offer"; sdp: string; peer_id: string }
  | { type: "answer"; sdp: string; peer_id: string }
  | { type: "ice_candidate"; candidate: string; sdp_mline_index: number; sdp_mid: string; peer_id: string }
  | { type: "join"; room_id: string; token: string }
  | { type: "leave"; room_id: string };

`
}

func generateUtilityTypes() string {
	return `
// ==========================================
// UTILITY TYPES
// ==========================================

export type Role = "guest" | "user" | "host" | "agency" | "admin";

export type ContentType = "stream" | "story" | "vod" | "comment";

export type RoomMode = "public" | "password" | "paid" | "members" | "private" | "official";

export type StreamStatus = "preparing" | "live" | "paused" | "ended" | "banned";

export type BookingStatus = "pending" | "host_accepted" | "confirmed" | "active" | "completed" | "cancelled" | "host_rejected" | "user_cancelled";

export type PKMode = "1v1" | "team2v2" | "team3v3" | "mvp";

export type GiftRarity = "common" | "rare" | "epic" | "legendary";

export type GiftType = "normal" | "lucky_box" | "privilege" | "entry_ticket";

export type TransactionType = "deposit" | "withdrawal" | "gift_sent" | "gift_received" | "agency_commission" | "host_earning" | "booking_payment" | "call_payment" | "refund";

export type TransactionStatus = "pending" | "success" | "failed" | "cancelled";

export type CryptoChain = "SOL" | "BTC" | "USDT_ERC20" | "USDT_TRC20" | "USDT_BEP20";

export type ModerationAction = "warn" | "mute" | "kick" | "ban_temp" | "ban_perm" | "blur_image" | "flag_review";

// API Error Codes
export type ApiErrorCode =
  | "UNAUTHORIZED"
  | "FORBIDDEN"
  | "INVALID_TOKEN"
  | "TOKEN_EXPIRED"
  | "TOKEN_REVOKED"
  | "VALIDATION_ERROR"
  | "CONFLICT"
  | "NOT_FOUND"
  | "RATE_LIMIT_EXCEEDED"
  | "INTERNAL_ERROR"
  | "INSUFFICIENT_BALANCE"
  | "SLOT_UNAVAILABLE"
  | "SCHEDULE_OVERLAP"
  | "CLIP_DURATION_INVALID"
  | "MESSAGE_BLOCKED"
  | "OFFER_EXPIRED";

// Helper type untuk API calls
export type ApiMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export interface ApiEndpoint<TRequest = any, TResponse = any> {
  path: string;
  method: ApiMethod;
  requiresAuth: boolean;
  requestType?: TRequest;
  responseType?: TResponse;
}

`
}
