package delivery

import (
	"time"

	"nvide-live/internal/domain"
)

// RegisterRequest untuk registrasi
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest untuk login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse untuk response auth
type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *UserDTO    `json:"user"`
}

// UserDTO untuk response user
type UserDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	IsVerified bool `json:"is_verified"`
}

// StoryDTO untuk response story
type StoryDTO struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	MediaType string    `json:"media_type"`
	ExpiresAt time.Time `json:"expires_at"`
	ViewCount int       `json:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentDTO untuk response comment
type CommentDTO struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ContentID  string    `json:"content_id"`
	ContentType string   `json:"content_type"`
	ParentID   *string   `json:"parent_id,omitempty"`
	Content    string    `json:"content"`
	LikeCount  int       `json:"like_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MessageDTO untuk response message
type MessageDTO struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *string   `json:"reply_to_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toUserDTO converts domain.User to UserDTO
func toUserDTO(user *domain.User) *UserDTO {
	if user == nil {
		return nil
	}
	
	roleName := ""
	if user.Role != nil {
		roleName = user.Role.Name
	}

	return &UserDTO{
		ID:         user.ID.String(),
		Username:   user.Username,
		Email:      user.Email,
		Role:       roleName,
		AvatarURL:  user.AvatarURL,
		IsVerified: user.IsVerified,
	}
}

// toStoryDTO converts domain.Story to StoryDTO
func toStoryDTO(story *domain.Story) *StoryDTO {
	return &StoryDTO{
		ID:        story.ID.String(),
		UserID:    story.UserID.String(),
		Content:   story.Content,
		MediaType: story.MediaType,
		ExpiresAt: story.ExpiresAt,
		ViewCount: story.ViewCount,
		CreatedAt: story.CreatedAt,
		UpdatedAt: story.UpdatedAt,
	}
}

// toCommentDTO converts domain.Comment to CommentDTO
func toCommentDTO(comment *domain.Comment) *CommentDTO {
	return &CommentDTO{
		ID:         comment.ID.String(),
		UserID:     comment.UserID.String(),
		ContentID:  comment.ContentID.String(),
		ContentType: comment.ContentType,
		ParentID:   func() *string {
			if comment.ParentID != nil {
				s := comment.ParentID.String()
				return &s
			}
			return nil
		}(),
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
	}
}

// toMessageDTO converts domain.Message to MessageDTO
func toMessageDTO(message *domain.Message) *MessageDTO {
	return &MessageDTO{
		ID:        message.ID.String(),
		RoomID:    message.RoomID.String(),
		UserID:    message.UserID.String(),
		Content:   message.Content,
		Type:      message.Type,
		ReplyToID: func() *string {
			if message.ReplyToID != nil {
				s := message.ReplyToID.String()
				return &s
			}
			return nil
		}(),
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}
