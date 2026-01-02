package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bher20/eratemanager/internal/storage"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	storage  storage.Storage
	enforcer *casbin.Enforcer
}

func NewService(s storage.Storage) (*Service, error) {
	// Initialize Casbin
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

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, err
	}

	// Add default policies
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

	return &Service{storage: s, enforcer: e}, nil
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

func (s *Service) Register(ctx context.Context, username, password, role string) (*storage.User, error) {
	// Check if user exists
	existing, err := s.storage.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u := storage.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.storage.CreateUser(ctx, u); err != nil {
		return nil, err
	}

	// Add user to role in Casbin
	s.enforcer.AddGroupingPolicy(u.ID, role)

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
	return nil
}
