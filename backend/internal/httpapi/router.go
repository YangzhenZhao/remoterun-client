package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"remoterun-backend/internal/config"
	"remoterun-backend/internal/db"
)

const (
	sessionUserIDKey   = "user_id"
	sessionUsernameKey = "username"
	sessionCSRFKey     = "csrf_token"
)

type Handler struct {
	config     config.Config
	db         *pgxpool.Pool
	httpClient *http.Client
}

type authUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type sessionResponse struct {
	Authenticated bool      `json:"authenticated"`
	User          *authUser `json:"user,omitempty"`
	CSRFToken     string    `json:"csrfToken,omitempty"`
}

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type runCommandRequest struct {
	ServerID     string `json:"serverId"`
	CommandAlias string `json:"commandAlias"`
}

type createCommandRequest struct {
	Alias   string `json:"alias"`
	Command string `json:"command"`
}

func NewRouter(cfg config.Config, pool *pgxpool.Pool) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())
	engine.Use(ginSecurityHeaders())
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.AllowedOrigin},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	store := cookie.NewStore([]byte(cfg.SessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60 * 60 * 24,
		Secure:   cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	engine.Use(sessions.Sessions(cfg.SessionName, store))

	handler := &Handler{
		config:     cfg,
		db:         pool,
		httpClient: &http.Client{Timeout: cfg.UpstreamTimeout},
	}

	api := engine.Group("/api")
	authRoutes := api.Group("/auth")
	authRoutes.GET("/session", handler.getSession)
	authRoutes.POST("/register", handler.register)
	authRoutes.POST("/login", handler.login)
	authRoutes.POST("/logout", handler.requireAuth(), handler.requireCSRF(), handler.logout)

	protected := api.Group("")
	protected.Use(handler.requireAuth())
	protected.GET("/servers", handler.listServers)
	protected.GET("/servers/:id", handler.getServer)
	protected.POST("/servers", handler.requireCSRF(), handler.createServer)
	protected.POST("/servers/:id/commands", handler.requireCSRF(), handler.createCommand)
	protected.POST("/run", handler.requireCSRF(), handler.runCommand)

	return engine
}

func ginSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Cache-Control", "no-store")
		headers.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		c.Next()
	}
}

func (h *Handler) getSession(c *gin.Context) {
	session := sessions.Default(c)
	user := sessionUser(session)
	if user == nil {
		c.JSON(http.StatusOK, sessionResponse{Authenticated: false})
		return
	}

	csrfToken := stringValue(session.Get(sessionCSRFKey))
	if csrfToken == "" {
		var err error
		csrfToken, err = randomToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create csrf token"})
			return
		}

		session.Set(sessionCSRFKey, csrfToken)
		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist session"})
			return
		}
	}

	c.JSON(http.StatusOK, sessionResponse{
		Authenticated: true,
		User:          user,
		CSRFToken:     csrfToken,
	})
}

func (h *Handler) login(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login payload"})
		return
	}

	request.Username = strings.TrimSpace(request.Username)
	if !db.ValidateUsername(request.Username) || strings.TrimSpace(request.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username or password"})
		return
	}

	user, err := db.FindUserByUsername(c.Request.Context(), h.db, request.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "username or password is incorrect"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query user"})
		return
	}

	passwordErr := db.CheckPassword(user.PasswordHash, request.Password)
	if passwordErr != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "username or password is incorrect"})
		return
	}

	h.startSession(c, user, http.StatusOK)
}

func (h *Handler) register(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid register payload"})
		return
	}

	user, err := db.CreateUser(c.Request.Context(), h.db, request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, db.ErrUsernameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	h.startSession(c, user, http.StatusCreated)
}

func (h *Handler) logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Secure:   h.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) listServers(c *gin.Context) {
	servers, err := db.ListServers(c.Request.Context(), h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	publicServers := make([]db.PublicServer, 0, len(servers))
	for _, server := range servers {
		publicServers = append(publicServers, db.ToPublicServer(server))
	}

	c.JSON(http.StatusOK, publicServers)
}

func (h *Handler) getServer(c *gin.Context) {
	server, err := db.FindServerByID(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	c.JSON(http.StatusOK, db.ToPublicServer(server))
}

func (h *Handler) createServer(c *gin.Context) {
	var request db.CreateServerInput
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server payload"})
		return
	}

	server, err := db.CreateServer(c.Request.Context(), h.db, request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, db.ToPublicServer(server))
}

func (h *Handler) createCommand(c *gin.Context) {
	var request createCommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid command payload"})
		return
	}

	server, err := db.AddCommandToServer(c.Request.Context(), h.db, c.Param("id"), db.CreateCommandInput{
		Alias:   request.Alias,
		Command: request.Command,
	})
	if err != nil {
		if errors.Is(err, db.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, db.ToPublicServer(server))
}

func (h *Handler) runCommand(c *gin.Context) {
	var request runCommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run payload"})
		return
	}

	server, err := db.FindServerByID(c.Request.Context(), h.db, strings.TrimSpace(request.ServerID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	var commandToRun *db.CommandConfig
	for i := range server.Commands {
		if server.Commands[i].Alias == strings.TrimSpace(request.CommandAlias) {
			commandToRun = &server.Commands[i]
			break
		}
	}

	if commandToRun == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
		return
	}

	upstreamPayload, err := json.Marshal(gin.H{
		"command":  commandToRun.Command,
		"password": server.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode upstream payload"})
		return
	}

	upstreamURL := fmt.Sprintf("http://%s:%d/run", server.Host, server.Port)
	upstreamRequest, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		upstreamURL,
		bytes.NewReader(upstreamPayload),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build upstream request"})
		return
	}

	upstreamRequest.Header.Set("Content-Type", "application/json")

	upstreamResponse, err := h.httpClient.Do(upstreamRequest)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reach remoterun-server"})
		return
	}
	defer upstreamResponse.Body.Close()

	body, err := io.ReadAll(upstreamResponse.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read upstream response"})
		return
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream returned invalid json"})
		return
	}

	c.JSON(upstreamResponse.StatusCode, payload)
}

func (h *Handler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if sessionUser(sessions.Default(c)) == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		c.Next()
	}
}

func (h *Handler) requireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" && !sameOrigin(origin, h.config.AllowedOrigin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid request origin"})
			return
		}

		session := sessions.Default(c)
		sessionToken := stringValue(session.Get(sessionCSRFKey))
		requestToken := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
		if sessionToken == "" || requestToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token is required"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(sessionToken), []byte(requestToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token mismatch"})
			return
		}

		c.Next()
	}
}

func sessionUser(session sessions.Session) *authUser {
	userID := stringValue(session.Get(sessionUserIDKey))
	username := stringValue(session.Get(sessionUsernameKey))
	if userID == "" || username == "" {
		return nil
	}

	return &authUser{ID: userID, Username: username}
}

func stringValue(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}

func sameOrigin(left string, right string) bool {
	leftURL, leftErr := url.Parse(left)
	rightURL, rightErr := url.Parse(right)
	if leftErr != nil || rightErr != nil {
		return false
	}

	return strings.EqualFold(leftURL.Scheme, rightURL.Scheme) && strings.EqualFold(leftURL.Host, rightURL.Host)
}

func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func (h *Handler) startSession(c *gin.Context, user db.User, statusCode int) {
	csrfToken, err := randomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create csrf token"})
		return
	}

	session := sessions.Default(c)
	session.Clear()
	session.Set(sessionUserIDKey, fmt.Sprintf("%d", user.ID))
	session.Set(sessionUsernameKey, user.Username)
	session.Set(sessionCSRFKey, csrfToken)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist session"})
		return
	}

	c.JSON(statusCode, sessionResponse{
		Authenticated: true,
		User: &authUser{
			ID:       fmt.Sprintf("%d", user.ID),
			Username: user.Username,
		},
		CSRFToken: csrfToken,
	})
}
