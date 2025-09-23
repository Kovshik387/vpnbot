package interfaces

import (
	"VpnBot/internal/domain/model"
	"VpnBot/internal/domain/ports/service"
	"VpnBot/internal/interfaces/http/dto"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type marzbanClient struct {
	baseUrl   string
	username  string
	password  string
	client    *http.Client
	token     string
	tokenType string
	expires   time.Time
}

const contentType = "application/x-www-form-urlencoded"

func NewMarzbanClient(baseUrl string, username string, password string) service.MarzbanService {
	return &marzbanClient{
		baseUrl:  baseUrl,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *marzbanClient) Delete(username string) error {
	if err := m.ensureToken(); err != nil {
		return err
	}
	log.Println("Deleting user", username)
	req, err := http.NewRequest(http.MethodDelete, m.baseUrl+"/api/user/"+username, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", m.tokenType+" "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.New(string(body))
	}

	return nil
}

func (m *marzbanClient) AddUser(username string) (model.User, error) {
	if err := m.ensureToken(); err != nil {
		return model.User{}, err
	}

	var reqBody = dto.AddUserRequest{
		Username:               username,
		Status:                 "active",
		Expire:                 0,
		DataLimit:              0,
		DataLimitResetStrategy: "no_reset",
		Proxies: map[string]map[string]string{
			"vless": {},
		},
		Inbounds: map[string][]string{
			"vless": {"TEAMDOCS", "VK", "WHITELIST"},
		},
		Note:                 "",
		OnHoldTimeout:        time.Now().Format(time.RFC3339),
		OnHoldExpireDuration: 0,
		NextPlan: map[string]interface{}{
			"add_remaining_traffic": false,
			"data_limit":            0,
			"expire":                0,
			"fire_on_either":        true,
		},
	}
	bodyStr, err := json.Marshal(reqBody)
	if err != nil {
		return model.User{}, err
	}

	req, err := http.NewRequest("POST", m.baseUrl+"/api/user/", bytes.NewBuffer(bodyStr))
	if err != nil {
		return model.User{}, err
	}

	req.Header.Add("Authorization", m.tokenType+" "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return model.User{}, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return model.User{}, errors.New(string(body))
	}

	var usersResponse model.User
	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return model.User{}, err
	}

	return usersResponse, nil
}

func (m *marzbanClient) GetUser(username string) (model.User, error) {
	if err := m.ensureToken(); err != nil {
		return model.User{}, err
	}

	req, err := http.NewRequest("GET", m.baseUrl+"/api/user/"+username, nil)
	if err != nil {
		return model.User{}, err
	}

	req.Header.Add("Authorization", m.tokenType+" "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return model.User{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return model.User{}, errors.New(string(body))
	}

	var usersResponse model.User
	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return model.User{}, err
	}

	return usersResponse, nil
}

func (m *marzbanClient) ensureToken() error {
	if m.token == "" || time.Now().After(m.expires) {
		return m.Login()
	}
	return nil
}

func (m *marzbanClient) Login() error {
	const grantType = "password"

	form := url.Values{}
	form.Set("grant_type", grantType)
	form.Set("username", m.username)
	form.Set("password", m.password)
	form.Set("scope", "")
	form.Set("client_id", "")
	form.Set("client_secret", "")

	resp, err := m.client.Post(m.baseUrl+"/api/admin/token", contentType, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		response, _ := io.ReadAll(resp.Body)
		return errors.New(string(response))
	}

	var accessModel model.AccessToken

	if err := json.NewDecoder(resp.Body).Decode(&accessModel); err != nil {
		return err
	}

	m.token = accessModel.AccessToken
	m.tokenType = accessModel.TokenType

	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(m.token, claims)
	if err != nil {
		return err
	}
	if exp, ok := claims["exp"].(float64); ok {
		m.expires = time.Unix(int64(exp), 0)
	} else {
		return errors.New("время жизни токена не найдено")
	}

	return nil
}
func (m *marzbanClient) GetUsers() (model.UsersResponse, error) {
	if err := m.ensureToken(); err != nil {
		return model.UsersResponse{}, err
	}

	req, err := http.NewRequest("GET", m.baseUrl+"/api/users", nil)
	if err != nil {
		return model.UsersResponse{}, err
	}

	req.Header.Add("Authorization", m.tokenType+" "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return model.UsersResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return model.UsersResponse{}, errors.New(string(body))
	}

	var usersResponse model.UsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&usersResponse); err != nil {
		return model.UsersResponse{}, err
	}

	return usersResponse, nil
}
