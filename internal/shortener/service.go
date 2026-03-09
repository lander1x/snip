package shortener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const codeLength = 6
const maxCodeRetries = 5

var (
	ErrInvalidURL   = errors.New("invalid URL: must be an absolute HTTP(S) URL")
	ErrInvalidAlias = errors.New("invalid alias: must be 1-12 alphanumeric characters")
)

type ClickEvent struct {
	Code      string `json:"code"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
	Timestamp int64  `json:"timestamp"`
}

type Service struct {
	repo    *Repository
	cache   *Cache
	js      nats.JetStreamContext
	baseURL string
	log     *slog.Logger
}

func NewService(repo *Repository, cache *Cache, js nats.JetStreamContext, baseURL string, log *slog.Logger) *Service {
	return &Service{
		repo:    repo,
		cache:   cache,
		js:      js,
		baseURL: baseURL,
		log:     log,
	}
}

var aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9]{1,12}$`)

func (s *Service) CreateLink(ctx context.Context, rawURL, customAlias string) (*Link, error) {
	if err := validateURL(rawURL); err != nil {
		return nil, err
	}

	if customAlias != "" {
		if !aliasRegex.MatchString(customAlias) {
			return nil, ErrInvalidAlias
		}
		link := &Link{
			ID:        uuid.New(),
			Code:      customAlias,
			URL:       rawURL,
			CreatedAt: time.Now().UTC(),
		}
		if err := s.repo.Create(ctx, link); err != nil {
			return nil, err
		}
		return link, nil
	}

	// Auto-generate code with retry on duplicate
	for range maxCodeRetries {
		link := &Link{
			ID:        uuid.New(),
			Code:      generateCode(),
			URL:       rawURL,
			CreatedAt: time.Now().UTC(),
		}
		err := s.repo.Create(ctx, link)
		if err == nil {
			return link, nil
		}
		if !isDuplicateKeyErr(err) {
			return nil, err
		}
		s.log.Warn("duplicate code generated, retrying", "code", link.Code)
	}

	return nil, fmt.Errorf("failed to generate unique code after %d attempts", maxCodeRetries)
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrInvalidURL
	}
	if u.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func isDuplicateKeyErr(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "23505")
}

func (s *Service) ResolveLink(ctx context.Context, code string) (*Link, error) {
	return s.repo.GetByCode(ctx, code)
}

func (s *Service) ResolveLinkCached(ctx context.Context, code string) (string, error) {
	// Check cache first
	url, err := s.cache.Get(ctx, code)
	if err != nil {
		s.log.Warn("cache get error", "error", err)
	}
	if url != "" {
		return url, nil
	}

	// Fallback to DB
	link, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	// Cache the result
	if err := s.cache.Set(ctx, code, link.URL); err != nil {
		s.log.Warn("cache set error", "error", err)
	}

	return link.URL, nil
}

func (s *Service) IncrementClicks(ctx context.Context, code string) {
	if err := s.repo.IncrementClicks(ctx, code); err != nil {
		s.log.Error("failed to increment clicks", "code", code, "error", err)
	}
}

func (s *Service) DeleteLink(ctx context.Context, code string) error {
	if err := s.repo.Delete(ctx, code); err != nil {
		return err
	}
	_ = s.cache.Delete(ctx, code)
	return nil
}

func (s *Service) PublishClick(event ClickEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		s.log.Error("failed to marshal click event", "error", err)
		return
	}
	if _, err := s.js.Publish("clicks.created", data); err != nil {
		s.log.Error("failed to publish click event", "error", err)
	}
}

func (s *Service) ShortURL(code string) string {
	return s.baseURL + "/" + code
}

func generateCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = base62Chars[rand.IntN(len(base62Chars))]
	}
	return string(b)
}
