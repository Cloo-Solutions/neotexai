package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/cloo-solutions/neotexai/internal/domain"
)

const apiKeyPrefix = "ntx_"

type OrgRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id string) (*domain.Organization, error)
	GetByName(ctx context.Context, name string) (*domain.Organization, error)
	List(ctx context.Context) ([]*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id string) error
}

type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	GetByID(ctx context.Context, id string) (*domain.APIKey, error)
	GetByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	GetByOrgID(ctx context.Context, orgID string) ([]*domain.APIKey, error)
	Revoke(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type AuthService struct {
	orgRepo OrgRepository
	keyRepo APIKeyRepository
	uuidGen UUIDGenerator
}

func NewAuthService(orgRepo OrgRepository, keyRepo APIKeyRepository, uuidGen UUIDGenerator) *AuthService {
	return &AuthService{
		orgRepo: orgRepo,
		keyRepo: keyRepo,
		uuidGen: uuidGen,
	}
}

func (s *AuthService) CreateOrg(ctx context.Context, name string) (*domain.Organization, error) {
	if name == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "organization name is required")
	}

	org := &domain.Organization{
		ID:        s.uuidGen.NewString(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	if err := domain.ValidateOrganization(org); err != nil {
		return nil, err
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *AuthService) CreateAPIKey(ctx context.Context, orgID, name string) (string, error) {
	if orgID == "" {
		return "", domain.NewDomainError(domain.ErrCodeValidation, "organization ID is required")
	}
	if name == "" {
		return "", domain.NewDomainError(domain.ErrCodeValidation, "API key name is required")
	}

	_, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return "", err
	}

	token, err := generateAPIToken()
	if err != nil {
		return "", domain.NewDomainErrorWithCause(domain.ErrCodeInternalError, "failed to generate API key", err)
	}

	hash := hashToken(token)

	key := &domain.APIKey{
		ID:        s.uuidGen.NewString(),
		OrgID:     orgID,
		Name:      name,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
		RevokedAt: nil,
	}

	if err := domain.ValidateAPIKey(key); err != nil {
		return "", err
	}

	if err := s.keyRepo.Create(ctx, key); err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) CreateAPIKeyWithToken(ctx context.Context, orgID, name, token string) error {
	if orgID == "" {
		return domain.NewDomainError(domain.ErrCodeValidation, "organization ID is required")
	}
	if name == "" {
		return domain.NewDomainError(domain.ErrCodeValidation, "API key name is required")
	}
	if !IsValidAPIToken(token) {
		return domain.NewDomainError(domain.ErrCodeValidation, "invalid API key format (expected ntx_<64 hex chars>)")
	}

	_, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	hash := hashToken(token)

	key := &domain.APIKey{
		ID:        s.uuidGen.NewString(),
		OrgID:     orgID,
		Name:      name,
		KeyHash:   hash,
		CreatedAt: time.Now().UTC(),
		RevokedAt: nil,
	}

	if err := domain.ValidateAPIKey(key); err != nil {
		return err
	}

	return s.keyRepo.Create(ctx, key)
}

func (s *AuthService) ValidateAPIKey(ctx context.Context, token string) (string, error) {
	if !IsValidAPIToken(token) {
		return "", domain.ErrInvalidAPIKey
	}

	hash := hashToken(token)

	key, err := s.keyRepo.GetByHash(ctx, hash)
	if err != nil {
		if err == domain.ErrAPIKeyNotFound {
			return "", domain.ErrInvalidAPIKey
		}
		return "", err
	}

	if key.IsRevoked() {
		return "", domain.ErrAPIKeyRevoked
	}

	return key.OrgID, nil
}

func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		return domain.NewDomainError(domain.ErrCodeValidation, "API key ID is required")
	}

	return s.keyRepo.Revoke(ctx, keyID)
}

func (s *AuthService) ListAPIKeys(ctx context.Context, orgID string) ([]*domain.APIKey, error) {
	if orgID == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "organization ID is required")
	}

	return s.keyRepo.GetByOrgID(ctx, orgID)
}

func (s *AuthService) GetAPIKeyByHash(ctx context.Context, token string) (*domain.APIKey, error) {
	if !IsValidAPIToken(token) {
		return nil, domain.ErrInvalidAPIKey
	}
	hash := hashToken(token)
	return s.keyRepo.GetByHash(ctx, hash)
}

func generateAPIToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return apiKeyPrefix + hex.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func IsValidAPIToken(token string) bool {
	if !strings.HasPrefix(token, apiKeyPrefix) {
		return false
	}
	hexPart := token[len(apiKeyPrefix):]
	if len(hexPart) != 64 {
		return false
	}
	for _, c := range hexPart {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
