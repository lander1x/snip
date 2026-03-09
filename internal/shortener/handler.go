package shortener

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type createLinkRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias"`
}

type linkResponse struct {
	ID        string  `json:"id"`
	Code      string  `json:"code"`
	URL       string  `json:"url"`
	ShortURL  string  `json:"short_url"`
	Clicks    int64   `json:"clicks"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func (h *Handler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Post("/links", h.CreateLink)
	api.Get("/links/:code", h.GetLink)
	api.Delete("/links/:code", h.DeleteLink)
	api.Get("/health", h.Health)

	// Redirect route — only match base62 codes (1-12 chars), skip favicon.ico etc.
	app.Get("/:code<regex(^[a-zA-Z0-9]{1,12}$)>", h.Redirect)
}

func (h *Handler) CreateLink(c *fiber.Ctx) error {
	var req createLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}

	link, err := h.svc.CreateLink(c.Context(), req.URL, req.CustomAlias)
	if err != nil {
		if errors.Is(err, ErrInvalidURL) || errors.Is(err, ErrInvalidAlias) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(toLinkResponse(link, h.svc.ShortURL(link.Code)))
}

func (h *Handler) GetLink(c *fiber.Ctx) error {
	code := c.Params("code")
	link, err := h.svc.ResolveLink(c.Context(), code)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(toLinkResponse(link, h.svc.ShortURL(link.Code)))
}

func (h *Handler) DeleteLink(c *fiber.Ctx) error {
	code := c.Params("code")
	err := h.svc.DeleteLink(c.Context(), code)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Redirect(c *fiber.Ctx) error {
	code := c.Params("code")
	url, err := h.svc.ResolveLinkCached(c.Context(), code)
	if err != nil {
		if errors.Is(err, ErrLinkNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "link not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Copy values BEFORE goroutines — Fiber recycles Ctx after handler returns
	event := ClickEvent{
		Code:      code,
		IP:        c.IP(),
		UserAgent: c.Get("User-Agent"),
		Referer:   c.Get("Referer"),
		Timestamp: time.Now().Unix(),
	}

	go h.svc.PublishClick(event)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.svc.IncrementClicks(ctx, code)
	}()

	return c.Redirect(url, fiber.StatusFound)
}

func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func toLinkResponse(link *Link, shortURL string) linkResponse {
	resp := linkResponse{
		ID:        link.ID.String(),
		Code:      link.Code,
		URL:       link.URL,
		ShortURL:  shortURL,
		Clicks:    link.Clicks,
		CreatedAt: link.CreatedAt.Format(time.RFC3339),
	}
	if link.ExpiresAt != nil {
		s := link.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp
}
