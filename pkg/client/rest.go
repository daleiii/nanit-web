package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/indiefan/home_assistant_nanit/pkg/baby"
	"github.com/indiefan/home_assistant_nanit/pkg/message"
	"github.com/indiefan/home_assistant_nanit/pkg/session"
	"github.com/indiefan/home_assistant_nanit/pkg/utils"
	"github.com/rs/zerolog/log"
)

var myClient = &http.Client{Timeout: 10 * time.Second}
var ErrExpiredRefreshToken = errors.New("Refresh token has expired. Relogin required.")

// ------------------------------------------

type authResponsePayload struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"` // We can store this to renew a session, avoiding the need to re-auth with MFA
}

type babiesResponsePayload struct {
	Babies []baby.Baby `json:"babies"`
}

type messagesResponsePayload struct {
	Messages []message.Message `json:"messages"`
}

// ------------------------------------------

// NanitClient - client context
type NanitClient struct {
	Email        string
	Password     string
	RefreshToken string
	SessionStore *session.Store
}

// MaybeAuthorize - Performs authorization if we don't have token or we assume it is expired
func (c *NanitClient) MaybeAuthorize(force bool) error {
	if force || c.SessionStore.Session.AuthToken == "" || time.Since(c.SessionStore.Session.AuthTime) > AuthTokenTimelife {
		return c.Authorize()
	}
	return nil
}

// Authorize - performs authorization attempt, returns error if it fails
func (c *NanitClient) Authorize() error {
	if len(c.SessionStore.Session.RefreshToken) == 0 {
		c.SessionStore.Session.RefreshToken = c.RefreshToken
	}

	if len(c.SessionStore.Session.RefreshToken) > 0 {
		err := c.RenewSession() // We have a refresh token, so we'll use that to extend our session
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrExpiredRefreshToken) {
			log.Error().Err(err).Msg("Unknown error occurred while trying to refresh the session")
			return fmt.Errorf("session renewal failed: %w", err)
		}
	}

	return c.Login() // We don't have a refresh token, e.g. initial login so we need to supply username/password
}

// Renews an existing session using a valid refresh token
// If the refresh token has also expired, we need to perform a full re-login
func (c *NanitClient) RenewSession() error {
	requestBody, requestBodyErr := json.Marshal(map[string]string{
		"refresh_token": c.SessionStore.Session.RefreshToken,
	})

	if requestBodyErr != nil {
		log.Error().Err(requestBodyErr).Msg("Unable to marshal auth body")
		return fmt.Errorf("failed to marshal refresh token request: %w", requestBodyErr)
	}

	r, clientErr := myClient.Post("https://api.nanit.com/tokens/refresh", "application/json", bytes.NewBuffer(requestBody))
	if clientErr != nil {
		log.Error().Err(clientErr).Msg("Unable to renew session")
		return fmt.Errorf("session renewal request failed: %w", clientErr)
	}

	defer r.Body.Close()
	if r.StatusCode == 404 {
		log.Warn().Msg("Server responded with code 404. This typically means your refresh token has expired. Will try to login with username/password")
		return ErrExpiredRefreshToken
	} else if r.StatusCode > 299 || r.StatusCode < 200 {
		log.Error().Int("code", r.StatusCode).Msg("Server responded with an error")
		return fmt.Errorf("session renewal failed with status code: %d", r.StatusCode)
	}

	authResponse := new(authResponsePayload)

	jsonErr := json.NewDecoder(r.Body).Decode(authResponse)
	if jsonErr != nil {
		log.Error().Err(jsonErr).Msg("Unable to decode response")
		return fmt.Errorf("failed to decode refresh token response: %w", jsonErr)
	}

	log.Info().Str("token", utils.AnonymizeToken(authResponse.AccessToken, 4)).Msg("Authorized")
	log.Info().Str("refresh_token", utils.AnonymizeToken(authResponse.RefreshToken, 4)).Msg("Retreived")
	c.SessionStore.Session.AuthToken = authResponse.AccessToken
	c.SessionStore.Session.RefreshToken = authResponse.RefreshToken
	c.SessionStore.Session.AuthTime = time.Now()
	if err := c.SessionStore.Save(); err != nil {
		log.Warn().Err(err).Msg("Failed to save session after token refresh")
	}

	return nil
}

func (c *NanitClient) Login() error {
	log.Info().Str("email", c.Email).Str("password", utils.AnonymizeToken(c.Password, 0)).Msg("Authorizing using user credentials")
	requestBody, requestBodyErr := json.Marshal(map[string]string{
		"email":    c.Email,
		"password": c.Password,
	})

	if requestBodyErr != nil {
		log.Error().Err(requestBodyErr).Msg("Unable to marshal auth body")
		return fmt.Errorf("failed to marshal login request: %w", requestBodyErr)
	}

	//nanit-api-version: 1
	req, reqErr := http.NewRequest("POST", "https://api.nanit.com/login", bytes.NewBuffer(requestBody))
	if reqErr != nil {
		log.Error().Err(reqErr).Msg("Unable to create request")
		return fmt.Errorf("failed to create login request: %w", reqErr)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("nanit-api-version", "1") // required if you have MFA enabled or it'll reject the request
	r, clientErr := myClient.Do(req)
	if clientErr != nil {
		log.Error().Err(clientErr).Msg("Unable to fetch auth token")
		return fmt.Errorf("login request failed: %w", clientErr)
	}

	defer r.Body.Close()

	if r.StatusCode == 401 {
		errMsg := "Server responded with code 401. Provided credentials has not been accepted by the server. Please check if your e-mail address and password is entered correctly and that 2FA is disabled on your account."
		log.Error().Msg(errMsg)
		return errors.New(errMsg)
	} else if r.StatusCode != 201 {
		errMsg := fmt.Sprintf("Server responded with unexpected status code: %d", r.StatusCode)
		log.Error().Int("code", r.StatusCode).Msg("Server responded with unexpected status code")
		return errors.New(errMsg)
	}

	authResponse := new(authResponsePayload)

	jsonErr := json.NewDecoder(r.Body).Decode(authResponse)
	if jsonErr != nil {
		log.Error().Err(jsonErr).Msg("Unable to decode response")
		return fmt.Errorf("failed to decode login response: %w", jsonErr)
	}

	log.Info().Str("token", utils.AnonymizeToken(authResponse.AccessToken, 4)).Msg("Authorized")
	log.Info().Str("refresh_token", utils.AnonymizeToken(authResponse.RefreshToken, 4)).Msg("Retreived")
	c.SessionStore.Session.AuthToken = authResponse.AccessToken
	c.SessionStore.Session.RefreshToken = authResponse.RefreshToken
	c.SessionStore.Session.AuthTime = time.Now()
	if err := c.SessionStore.Save(); err != nil {
		log.Warn().Err(err).Msg("Failed to save session after login")
	}
	
	return nil
}

// FetchAuthorized - makes authorized http request
func (c *NanitClient) FetchAuthorized(req *http.Request, data interface{}) error {
	for i := 0; i < 2; i++ {
		if c.SessionStore.Session.AuthToken != "" {
			req.Header.Set("Authorization", c.SessionStore.Session.AuthToken)

			res, clientErr := myClient.Do(req)
			if clientErr != nil {
				log.Error().Err(clientErr).Msg("HTTP request failed")
				return fmt.Errorf("HTTP request failed: %w", clientErr)
			}

			defer res.Body.Close()

			if res.StatusCode != 401 {
				if res.StatusCode != 200 {
					log.Error().Int("code", res.StatusCode).Msg("Server responded with unexpected status code")
					return fmt.Errorf("server responded with unexpected status code: %d", res.StatusCode)
				}

				jsonErr := json.NewDecoder(res.Body).Decode(data)
				if jsonErr != nil {
					log.Error().Err(jsonErr).Msg("Unable to decode response")
					return fmt.Errorf("failed to decode response: %w", jsonErr)
				}

				return nil
			}

			log.Info().Msg("Token might be expired. Will try to re-authenticate.")
		}

		if err := c.Authorize(); err != nil {
			log.Error().Err(err).Msg("Re-authorization failed")
			return fmt.Errorf("authorization failed on attempt %d: %w", i+1, err)
		}
	}

	errMsg := "Unable to make request due to failed authorization (2 attempts)"
	log.Error().Msg(errMsg)
	return errors.New(errMsg)
}

// FetchBabies - fetches baby list
func (c *NanitClient) FetchBabies() ([]baby.Baby, error) {
	log.Info().Msg("Fetching babies list")
	req, reqErr := http.NewRequest("GET", "https://api.nanit.com/babies", nil)

	if reqErr != nil {
		log.Error().Err(reqErr).Msg("Unable to create request")
		return nil, fmt.Errorf("failed to create babies request: %w", reqErr)
	}

	data := new(babiesResponsePayload)
	if err := c.FetchAuthorized(req, data); err != nil {
		return nil, fmt.Errorf("failed to fetch babies: %w", err)
	}

	c.SessionStore.Session.Babies = data.Babies
	if err := c.SessionStore.Save(); err != nil {
		log.Warn().Err(err).Msg("Failed to save session after fetching babies")
	}
	return data.Babies, nil
}

// FetchMessages - fetches message list
func (c *NanitClient) FetchMessages(babyUID string, limit int) ([]message.Message, error) {
	req, reqErr := http.NewRequest("GET", fmt.Sprintf("https://api.nanit.com/babies/%s/messages?limit=%d", babyUID, limit), nil)

	if reqErr != nil {
		log.Error().Err(reqErr).Msg("Unable to create request")
		return nil, fmt.Errorf("failed to create messages request: %w", reqErr)
	}

	data := new(messagesResponsePayload)
	if err := c.FetchAuthorized(req, data); err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return data.Messages, nil
}

// EnsureBabies - fetches baby list if not fetched already
func (c *NanitClient) EnsureBabies() ([]baby.Baby, error) {
	if len(c.SessionStore.Session.Babies) == 0 {
		return c.FetchBabies()
	}

	return c.SessionStore.Session.Babies, nil
}

// FetchNewMessages - fetches 10 newest messages, ignores any messages which were already fetched or which are older than 5 minutes
func (c *NanitClient) FetchNewMessages(babyUID string, defaultMessageTimeout time.Duration) ([]message.Message, error) {
	fetchedMessages, err := c.FetchMessages(babyUID, 10)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch messages")
		return nil, fmt.Errorf("failed to fetch new messages: %w", err)
	}
	newMessages := make([]message.Message, 0)

	// return empty [] if there are no fetchedMessages
	if len(fetchedMessages) == 0 {
		log.Debug().Msg("No messages fetched")
		return newMessages, nil
	}

	// sort fetechedMessages starting with most recent
	sort.Slice(fetchedMessages, func(i, j int) bool {
		return fetchedMessages[i].Time.Time().After(fetchedMessages[j].Time.Time())
	})

	lastSeenMessageTime := c.SessionStore.Session.LastSeenMessageTime
	messageTimeoutTime := lastSeenMessageTime
	log.Debug().Msgf("Last seen message time was %s", lastSeenMessageTime)

	// Don't know when last message was, set messageTimeout to default
	if lastSeenMessageTime.IsZero() {
		messageTimeoutTime = time.Now().UTC().Add(-defaultMessageTimeout)
	}

	// lastSeenMessageTime is older than most recent fetchedMessage, or is unset
	if lastSeenMessageTime.Before(fetchedMessages[0].Time.Time()) {
		lastSeenMessageTime = fetchedMessages[0].Time.Time()
		c.SessionStore.Session.LastSeenMessageTime = lastSeenMessageTime
		if err := c.SessionStore.Save(); err != nil {
			log.Warn().Err(err).Msg("Failed to save session after updating last seen message time")
		}
	}

	// Only keep messages that are more recent than messageTimeoutTime
	filteredMessages := message.FilterMessages(fetchedMessages, func(message message.Message) bool {
		return message.Time.Time().After(messageTimeoutTime)
	})

	log.Debug().Msgf("Found %d new messages", len(filteredMessages))
	log.Debug().Msgf("%+v\n", filteredMessages)

	return filteredMessages, nil
}
