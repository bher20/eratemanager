package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bher20/eratemanager/internal/notification"
	"github.com/bher20/eratemanager/internal/storage"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	storage   storage.Storage
	enforcer  *casbin.Enforcer
	adapter   *Adapter
	notifier  *notification.Service
	publicURL string
}

func NewService(s storage.Storage, n *notification.Service, publicURL string) (*Service, error) {
	// Initialize Casbin model
	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && (r.obj == p.obj || p.obj == "*") && (r.act == p.act || p.act == "*")
`)
	if err != nil {
		return nil, err
	}

	// Create adapter for database persistence
	adapter := NewAdapter(s)

	// Create enforcer with adapter for persistence
	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}

	// Enable auto-save so policy changes are persisted immediately
	e.EnableAutoSave(true)

	// Load policies from database
	if err := e.LoadPolicy(); err != nil {
		log.Printf("auth: warning: failed to load policies from database: %v", err)
		// Continue anyway - we'll add defaults below
	}

	// Check if we have any policies loaded, if not, add defaults
	policies, _ := e.GetPolicy()
	if len(policies) == 0 {
		log.Println("auth: no policies found in database, adding defaults")
		
		// Admin can do everything
		e.AddPolicy("admin", "*", "*")
		// Editor can read and write rates/providers
		e.AddPolicy("editor", "rates", "read")
		e.AddPolicy("editor", "rates", "write")
		e.AddPolicy("editor", "providers", "read")
		e.AddPolicy("editor", "providers", "write")
		// Viewer can only read
		e.AddPolicy("viewer", "rates", "read")
		e.AddPolicy("viewer", "providers", "read")
	} else {
		log.Printf("auth: loaded %d policies from database", len(policies))
	}

	// Load existing users and ensure their role mappings exist
	ctx := context.Background()
	users, err := s.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	log.Printf("auth: found %d users to sync roles", len(users))
	for _, u := range users {
		log.Printf("auth: syncing user %s role=%q", u.ID, u.Role)
		if u.Role != "" {
			// AddGroupingPolicy is idempotent - won't duplicate
			added, err := e.AddGroupingPolicy(u.ID, u.Role)
			if err != nil {
				log.Printf("auth: error adding policy for user %s: %v", u.ID, err)
			} else if added {
				log.Printf("auth: added policy for user %s -> %s", u.ID, u.Role)
			}
		}
	}

	return &Service{
		storage:   s,
		enforcer:  e,
		adapter:   adapter,
		notifier:  n,
		publicURL: publicURL,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (*storage.User, error) {
	u, err := s.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return u, nil
}

func (s *Service) Register(ctx context.Context, username, password, email, role string) (*storage.User, error) {
	return s.register(ctx, username, "", "", password, email, role, false)
}

func (s *Service) RegisterInvitedUser(ctx context.Context, username, firstName, lastName, email, role string) (*storage.User, error) {
	// Generate a random password for invited users
	randomPassword := uuid.New().String()
	return s.register(ctx, username, firstName, lastName, randomPassword, email, role, true)
}

func (s *Service) register(ctx context.Context, username, firstName, lastName, password, email, role string, isInvite bool) (*storage.User, error) {
	// Check if user exists
	existing, err := s.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user already exists")
	}

	// Check if email exists
	if email != "" {
		existingEmail, err := s.storage.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if existingEmail != nil {
			return nil, errors.New("email already in use")
		}
	} else {
		return nil, errors.New("email is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := storage.User{
		ID:            uuid.New().String(),
		Username:      username,
		FirstName:     firstName,
		LastName:      lastName,
		Email:         email,
		EmailVerified: false,
		PasswordHash:  string(hash),
		Role:          role,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.storage.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	// Add user to role in Casbin
	s.enforcer.AddGroupingPolicy(u.ID, role)

	// Send appropriate email
	go func() {
		if isInvite {
			if err := s.SendInvitationEmail(context.Background(), u.ID, u.Email, role); err != nil {
				log.Printf("failed to send invitation email to %s: %v", u.Email, err)
			}
		} else {
			if err := s.SendVerificationEmail(context.Background(), u.ID, u.Email); err != nil {
				log.Printf("failed to send verification email to %s: %v", u.Email, err)
			}
		}
	}()

	return &u, nil
}

func (s *Service) CreateToken(ctx context.Context, userID, name, role string, expiresAt *time.Time) (*storage.Token, string, error) {
	// Generate token
	rawToken := uuid.New().String() + uuid.New().String()

	// Hash token for storage
	hasher := sha256.New()
	hasher.Write([]byte(rawToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	t := storage.Token{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		TokenHash: tokenHash,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if err := s.storage.CreateToken(ctx, t); err != nil {
		return nil, "", err
	}

	return &t, rawToken, nil
}

func (s *Service) ValidateToken(ctx context.Context, rawToken string) (*storage.Token, error) {
	hasher := sha256.New()
	hasher.Write([]byte(rawToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	t, err := s.storage.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("invalid token")
	}

	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	// Update last used
	go s.storage.UpdateTokenLastUsed(context.Background(), t.ID)

	return t, nil
}

func (s *Service) Enforce(sub, obj, act string) (bool, error) {
	return s.enforcer.Enforce(sub, obj, act)
}

func (s *Service) LoadPolicy() error {
	return s.enforcer.LoadPolicy()
}

func (s *Service) GetAllRoles() ([]string, error) {
	return s.enforcer.GetAllSubjects()
}

func (s *Service) GetAllPolicies() ([][]string, error) {
	return s.enforcer.GetPolicy()
}

func (s *Service) AddPolicy(role, resource, action string) (bool, error) {
	return s.enforcer.AddPolicy(role, resource, action)
}

func (s *Service) RemovePolicy(role, resource, action string) (bool, error) {
	return s.enforcer.RemovePolicy(role, resource, action)
}

func (s *Service) AddGroupingPolicy(user, role string) (bool, error) {
	return s.enforcer.AddGroupingPolicy(user, role)
}

func (s *Service) RemoveGroupingPolicy(user, role string) (bool, error) {
	return s.enforcer.RemoveGroupingPolicy(user, role)
}

type Policy struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

func (s *Service) CreateRole(role string, policies []Policy) (bool, error) {
	// If no policies provided, add a default one to ensure role exists
	if len(policies) == 0 {
		return s.enforcer.AddPolicy(role, "system", "init")
	}

	// Add all policies
	for _, p := range policies {
		if _, err := s.enforcer.AddPolicy(role, p.Resource, p.Action); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Service) UpdateUser(ctx context.Context, id string, email, role string, skipVerification *bool, onboardingCompleted *bool) (*storage.User, error) {
	user, err := s.storage.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	changed := false
	emailChanged := false

	if email != "" && email != user.Email {
		// Check if email is taken
		existing, err := s.storage.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, errors.New("email already in use")
		}
		user.Email = email
		user.EmailVerified = false
		emailChanged = true
		changed = true
	}

	if role != "" && role != user.Role {
		// Remove old role policy
		s.enforcer.RemoveGroupingPolicy(user.ID, user.Role)
		user.Role = role
		// Add new role policy
		s.enforcer.AddGroupingPolicy(user.ID, role)
		changed = true
	}

	if skipVerification != nil && *skipVerification != user.SkipEmailVerification {
		user.SkipEmailVerification = *skipVerification
		changed = true
	}

	if onboardingCompleted != nil && *onboardingCompleted != user.OnboardingCompleted {
		user.OnboardingCompleted = *onboardingCompleted
		changed = true
	}

	if changed {
		user.UpdatedAt = time.Now()
		if err := s.storage.UpdateUser(ctx, *user); err != nil {
			return nil, err
		}
	}

	if emailChanged {
		go func() {
			if err := s.SendVerificationEmail(context.Background(), user.ID, user.Email); err != nil {
				log.Printf("failed to send verification email to %s: %v", user.Email, err)
			}
		}()
	}

	return user, nil
}

func (s *Service) SendVerificationEmail(ctx context.Context, userID, email string) error {
	expiresAt := time.Now().Add(24 * time.Hour)
	_, rawToken, err := s.CreateToken(ctx, userID, "email-verification", "verification", &expiresAt)
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/ui/verify-email?token=%s", s.publicURL, rawToken)
	return s.sendTemplateEmail(ctx, email, "Verify your email address", "Verify Email Address", "verify your email address", link)
}

func (s *Service) SendInvitationEmail(ctx context.Context, userID, email, role string) error {
	expiresAt := time.Now().Add(72 * time.Hour) // 3 days for invitations
	_, rawToken, err := s.CreateToken(ctx, userID, "account-setup", "verification", &expiresAt)
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/ui/setup-account?token=%s", s.publicURL, rawToken)
	
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f4f4f4; margin: 0; padding: 0; }
  .container { max-width: 600px; margin: 20px auto; background: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
  .header { background-color: #2563eb; color: #ffffff; padding: 20px; text-align: center; }
  .content { padding: 30px 20px; }
  .button { display: inline-block; padding: 12px 24px; background-color: #2563eb; color: #ffffff !important; text-decoration: none; border-radius: 6px; font-weight: bold; margin: 20px 0; }
  .button:visited { color: #ffffff !important; }
  .button:hover { background-color: #1d4ed8; color: #ffffff !important; }
  .footer { padding: 20px; text-align: center; font-size: 0.8em; color: #666; background-color: #f9fafb; }
  .link-text { word-break: break-all; color: #2563eb; }
  .info-box { background-color: #eff6ff; border-left: 4px solid #2563eb; padding: 15px; margin: 20px 0; border-radius: 4px; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1 style="margin:0; font-size: 24px;">eRateManager</h1>
  </div>
  <div class="content">
    <h2 style="margin-top:0; color: #2563eb;">You've Been Invited!</h2>
    <p>You have been invited to join <strong>eRateManager</strong> as a <strong>%s</strong>.</p>
    <div class="info-box">
      <p style="margin:0;"><strong>What's next?</strong></p>
      <p style="margin:5px 0 0 0;">Click the button below to set up your account and create your password. This link will expire in 72 hours.</p>
    </div>
    <div style="text-align: center;">
      <a href="%s" class="button" style="background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; display: inline-block; font-weight: bold;">Set Up My Account</a>
    </div>
    <p>Or copy and paste this link into your browser:</p>
    <p><a href="%s" class="link-text">%s</a></p>
  </div>
  <div class="footer">
    <p>If you did not expect this invitation, please ignore this email or contact your administrator.</p>
  </div>
</div>
</body>
</html>
`, role, link, link, link)

	return s.notifier.SendEmail(ctx, email, "You've been invited to eRateManager", htmlBody)
}


func (s *Service) VerifyEmail(ctx context.Context, rawToken string) error {
	token, err := s.ValidateToken(ctx, rawToken)
	if err != nil {
		return err
	}

	if token.Name != "email-verification" && token.Name != "account-setup" {
		return errors.New("invalid token type")
	}

	user, err := s.storage.GetUser(ctx, token.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	
	if err := s.storage.UpdateUser(ctx, *user); err != nil {
		return err
	}

	// Delete token
	return s.storage.DeleteToken(ctx, token.ID)
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		// Return nil to avoid enumerating users
		return nil
	}

	if !user.EmailVerified {
		return errors.New("email not verified")
	}

	// Create a reset token
	expiresAt := time.Now().Add(1 * time.Hour)
	_, rawToken, err := s.CreateToken(ctx, user.ID, "password-reset", "reset", &expiresAt)
	if err != nil {
		return err
	}

	link := fmt.Sprintf("%s/ui/reset-password?token=%s", s.publicURL, rawToken)
	return s.sendTemplateEmail(ctx, user.Email, "Password Reset Request", "Reset Password", "reset your password", link)
}

func (s *Service) sendTemplateEmail(ctx context.Context, to, subject, title, actionText, link string) error {
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; background-color: #f4f4f4; margin: 0; padding: 0; }
  .container { max-width: 600px; margin: 20px auto; background: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
  .header { background-color: #2563eb; color: #ffffff; padding: 20px; text-align: center; }
  .content { padding: 30px 20px; }
  .button { display: inline-block; padding: 12px 24px; background-color: #2563eb; color: #ffffff !important; text-decoration: none; border-radius: 6px; font-weight: bold; margin: 20px 0; }
  .button:visited { color: #ffffff !important; }
  .button:hover { background-color: #1d4ed8; color: #ffffff !important; }
  .footer { padding: 20px; text-align: center; font-size: 0.8em; color: #666; background-color: #f9fafb; }
  .link-text { word-break: break-all; color: #2563eb; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1 style="margin:0; font-size: 24px;">eRateManager</h1>
  </div>
  <div class="content">
    <h2 style="margin-top:0; color: #2563eb; text-align: center;">%s</h2>
    <p style="text-align: center;">Please click the button below to %s:</p>
    <div style="text-align: center;">
      <a href="%s" class="button" style="background-color: #2563eb; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 6px; display: inline-block; font-weight: bold;">%s</a>
    </div>
    <p>Or copy and paste this link into your browser:</p>
    <p><a href="%s" class="link-text">%s</a></p>
  </div>
  <div class="footer">
    <p>If you did not request this, please ignore this email.</p>
  </div>
</div>
</body>
</html>
`, title, actionText, link, title, link, link)

	return s.notifier.SendEmail(ctx, to, subject, htmlBody)
}

func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	// Validate token
	token, err := s.ValidateToken(ctx, rawToken)
	if err != nil {
		return err
	}
	
	if token.Name != "password-reset" && token.Name != "account-setup" {
		return errors.New("invalid token type")
	}

	// Get user
	user, err := s.storage.GetUser(ctx, token.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update user
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()
	
	// If this is an account setup token, also verify the email
	if token.Name == "account-setup" {
		user.EmailVerified = true
	}
	
	if err := s.storage.UpdateUser(ctx, *user); err != nil {
		return err
	}

	// Delete the used token
	if err := s.storage.DeleteToken(ctx, token.ID); err != nil {
		log.Printf("failed to delete used reset token: %v", err)
	}

	return nil
}

func (s *Service) ValidateSetupToken(ctx context.Context, rawToken string) (*storage.User, error) {
	token, err := s.ValidateToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	
	if token.Name != "account-setup" {
		return nil, errors.New("invalid token type")
	}

	user, err := s.storage.GetUser(ctx, token.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (s *Service) SetupInvitedAccount(ctx context.Context, rawToken, username, firstName, lastName, newPassword string) error {
	// Validate token
	token, err := s.ValidateToken(ctx, rawToken)
	if err != nil {
		return err
	}
	
	if token.Name != "account-setup" {
		return errors.New("invalid token type")
	}

	// Get user
	user, err := s.storage.GetUser(ctx, token.UserID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	// Check if new username is already taken (if different from current)
	if username != user.Username {
		existing, err := s.storage.GetUserByUsername(ctx, username)
		if err != nil {
			return err
		}
		if existing != nil && existing.ID != user.ID {
			return errors.New("username already taken")
		}
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update user with new username, first/last name, password, and verify email
	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.PasswordHash = string(hash)
	user.EmailVerified = true
	user.UpdatedAt = time.Now()
	
	if err := s.storage.UpdateUser(ctx, *user); err != nil {
		return err
	}

	// Delete the used token
	if err := s.storage.DeleteToken(ctx, token.ID); err != nil {
		log.Printf("failed to delete used setup token: %v", err)
	}

	return nil
}
