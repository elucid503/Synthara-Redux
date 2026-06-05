package Server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"Synthara-Redux/Globals"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/oauth2"
	"github.com/disgoorg/snowflake/v2"
	"github.com/gin-gonic/gin"
)

const (

	WebSessionCookie = "synthara_web_session"
	WebSessionTTL = 7 * 24 * time.Hour // 7 days

)

var (

	OAuthClient *oauth2.Client

	webSessions = make(map[string]webSession)

	webSessionMu sync.RWMutex
	oauthReturnMu sync.Mutex

	oauthReturns = make(map[string]string)

)

type webSession struct {

	Username string
	UserID snowflake.ID
	ExpiresAt time.Time

}

// InitAuth configures Discord OAuth2 for the web UI. Requires Globals.DiscordClient.
func InitAuth() error {

	if Globals.DiscordClient == nil {

		return nil

	}

	Secret := strings.TrimSpace(os.Getenv("DISCORD_CLIENT_SECRET"))

	if Secret == "" {

		return nil

	}

	ClientID := Globals.DiscordClient.ApplicationID
	OAuthClient = oauth2.New(ClientID, Secret)

	return nil

}

func OAuthEnabled() bool {

	return OAuthClient != nil

}

func OAuthRedirectURI() string {

	Domain := strings.TrimRight(strings.TrimSpace(os.Getenv("DOMAIN")), "/")

	if Domain == "" {

		Domain = "http://localhost:8080"

	}

	return Domain + "/API/Auth/Callback"

}

func newSessionToken() (string, error) {

	Bytes := make([]byte, 32)

	if _, Err := rand.Read(Bytes); Err != nil {

		return "", Err

	}

	return hex.EncodeToString(Bytes), nil

}

func storeSession(Token, Username string, UserID snowflake.ID) {

	webSessionMu.Lock()
	defer webSessionMu.Unlock()

	webSessions[Token] = webSession{

		Username: Username,
		UserID: UserID,
		ExpiresAt: time.Now().Add(WebSessionTTL),

	}

}

func sessionFromToken(Token string) (webSession, bool) {

	webSessionMu.RLock()
	Session, Exists := webSessions[Token]
	webSessionMu.RUnlock()

	if !Exists || time.Now().After(Session.ExpiresAt) {

		if Exists {

			webSessionMu.Lock()
			delete(webSessions, Token)
			webSessionMu.Unlock()

		}

		return webSession{}, false

	}

	return Session, true

}

func deleteSession(Token string) {

	webSessionMu.Lock()
	delete(webSessions, Token)
	webSessionMu.Unlock()

}

func sessionTokenFromRequest(Request *http.Request) string {

	if Request == nil {

		return ""

	}

	Cookie, Err := Request.Cookie(WebSessionCookie)

	if Err != nil || Cookie.Value == "" {

		return ""

	}

	return Cookie.Value

}

// WebUserFromRequest returns the authenticated Discord username for a request, if any.
func WebUserFromRequest(Request *http.Request) (string, bool) {

	Token := sessionTokenFromRequest(Request)

	if Token == "" {

		return "", false

	}

	Session, OK := sessionFromToken(Token)

	if !OK {

		return "", false

	}

	return Session.Username, true

}

func setSessionCookie(Context *gin.Context, Token string) {

	Secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(os.Getenv("DOMAIN"))), "https://")

	Context.SetSameSite(http.SameSiteLaxMode)
	Context.SetCookie(WebSessionCookie, Token, int(WebSessionTTL.Seconds()), "/", "", Secure, true)

}

func clearSessionCookie(Context *gin.Context) {

	Secure := strings.HasPrefix(strings.ToLower(strings.TrimSpace(os.Getenv("DOMAIN"))), "https://")

	Context.SetCookie(WebSessionCookie, "", -1, "/", "", Secure, true)

}

func rememberOAuthReturn(State, ReturnTo string) {

	oauthReturnMu.Lock()
	oauthReturns[State] = sanitizeReturnTo(ReturnTo)
	oauthReturnMu.Unlock()

}

func consumeOAuthReturn(State string) string {

	oauthReturnMu.Lock()
	ReturnTo := oauthReturns[State]
	delete(oauthReturns, State)
	oauthReturnMu.Unlock()

	return ReturnTo

}

func sanitizeReturnTo(ReturnTo string) string {

	ReturnTo = strings.TrimSpace(ReturnTo)

	if ReturnTo == "" || !strings.HasPrefix(ReturnTo, "/") || strings.HasPrefix(ReturnTo, "//") {

		return "/"

	}

	return ReturnTo

}

func HandleAuthLogin(Context *gin.Context) {

	if !OAuthEnabled() {

		Context.JSON(http.StatusServiceUnavailable, gin.H{"Error": "Discord login is not configured."})
		return

	}

	ReturnTo := sanitizeReturnTo(Context.Query("returnTo"))

	AuthURL, State := OAuthClient.GenerateAuthorizationURLState(oauth2.AuthorizationURLParams{

		RedirectURI: OAuthRedirectURI(),
		Scopes: []discord.OAuth2Scope{discord.OAuth2ScopeIdentify},

	})

	rememberOAuthReturn(State, ReturnTo)

	Context.Redirect(http.StatusFound, AuthURL)

}

func HandleAuthCallback(Context *gin.Context) {

	if !OAuthEnabled() {

		Context.String(http.StatusServiceUnavailable, "Discord login is not configured.")
		return

	}

	Code := Context.Query("code")
	State := Context.Query("state")

	if Code == "" || State == "" {

		Context.String(http.StatusBadRequest, "Missing OAuth code or state.")
		return

	}

	Session, _, Err := OAuthClient.StartSession(Code, State)

	if Err != nil {

		Context.String(http.StatusBadRequest, "Failed to complete Discord login.")
		return

	}

	User, Err := OAuthClient.GetUser(Session)

	if Err != nil || User == nil || strings.TrimSpace(User.Username) == "" {

		Context.String(http.StatusBadRequest, "Failed to load Discord profile.")
		return

	}

	Token, Err := newSessionToken()

	if Err != nil {

		Context.String(http.StatusInternalServerError, "Failed to create session.")
		return

	}

	storeSession(Token, User.Username, User.ID)
	setSessionCookie(Context, Token)

	Context.Redirect(http.StatusFound, consumeOAuthReturn(State))

}

func HandleAuthMe(Context *gin.Context) {

	Enabled := OAuthEnabled()

	Username, OK := WebUserFromRequest(Context.Request)

	if !OK {

		Context.JSON(http.StatusOK, gin.H{

			"Authenticated": false,
			"OAuthEnabled": Enabled,

		})

		return

	}

	Context.JSON(http.StatusOK, gin.H{

		"Authenticated": true,
		"OAuthEnabled":  Enabled,
		"Username": Username,

	})

}

func HandleAuthLogout(Context *gin.Context) {

	Token := sessionTokenFromRequest(Context.Request)

	if Token != "" {

		deleteSession(Token)

	}

	clearSessionCookie(Context)

	Context.JSON(http.StatusOK, gin.H{"OK": true})

}

// WebAuthenticated reports whether the request has a valid Discord OAuth session.
func WebAuthenticated(Request *http.Request) bool {

	if !OAuthEnabled() || Request == nil {

		return false

	}

	_, OK := WebUserFromRequest(Request)

	return OK

}

// WebControlsLocked reports whether web controls should be rejected.
func WebControlsLocked(GuildLocked bool, Request *http.Request) bool {

	if !WebAuthenticated(Request) {

		return true

	}

	return GuildLocked

}

// WebControlsLockMessage returns the error shown when controls are locked.
func WebControlsLockMessage(GuildLocked bool, Request *http.Request) string {

	if !OAuthEnabled() {

		return "Discord OAuth is not configured. Web controls are unavailable."

	}

	if !WebAuthenticated(Request) {

		return ""

	}

	if GuildLocked {

		return "Web controls are locked. Use /unlock to re-enable."

	}

	return ""

}

// WebUserForControls returns the Discord username used for web operations.
func WebUserForControls(Request *http.Request) string {

	Username, OK := WebUserFromRequest(Request)

	if !OK {

		return ""

	}

	return Username

}

func webControlsLockStatus(GuildLocked bool, Request *http.Request) int {

	if !OAuthEnabled() {

		return http.StatusServiceUnavailable

	}

	if !WebAuthenticated(Request) {

		return http.StatusUnauthorized

	}

	if GuildLocked {

		return http.StatusForbidden

	}

	return http.StatusForbidden

}
