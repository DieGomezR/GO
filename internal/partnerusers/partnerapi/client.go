package partnerapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tienda-go/internal/partnerusers"
)

// Config define como hablar con la API externa del partner.
type Config struct {
	BaseURL       string
	Partner       string
	Password      string
	Timeout       time.Duration
	SkipTLSVerify bool
}

// Client implementa partnerusers.PartnerAPIClient usando HTTP.
type Client struct {
	baseURL    string
	partner    string
	password   string
	httpClient *http.Client
}

// New construye un cliente listo para consumir la API del partner.
func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(strings.Trim(cfg.BaseURL, `"'`))
	if baseURL == "" {
		return nil, fmt.Errorf("partnerapi: base url is required")
	}
	if strings.TrimSpace(cfg.Partner) == "" {
		return nil, fmt.Errorf("partnerapi: partner user is required")
	}
	if strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("partnerapi: partner password is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.SkipTLSVerify {
		// El codigo PHP original usaba verify=false para este proveedor.
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		partner:  cfg.Partner,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}

// RegisterUser queda pendiente porque el endpoint exacto no aparece en el controller compartido.
func (c *Client) RegisterUser(context.Context, partnerusers.RegisterRequest, string, string) (partnerusers.RegisterAPIResult, error) {
	return partnerusers.RegisterAPIResult{}, fmt.Errorf("partnerapi: register user endpoint is not implemented yet")
}

// ModifyUser actualiza nombre o password del usuario en la API externa.
func (c *Client) ModifyUser(ctx context.Context, partnerID, name, password string) error {
	params := map[string]string{
		"partnerid": partnerID,
	}
	if strings.TrimSpace(name) != "" {
		params["fullName"] = name
	}
	if strings.TrimSpace(password) != "" {
		params["userPassword"] = password
	}

	var response operationResponse
	if err := c.postQuery(ctx, "/modifyuser", params, &response); err != nil {
		return err
	}
	if response.Status != 1 {
		return apiError(response.Error)
	}

	return nil
}

// ActivatePackage cambia el paquete activo del usuario.
func (c *Client) ActivatePackage(ctx context.Context, partnerID, packageCode string, to *time.Time) error {
	params := map[string]string{
		"partnerid": partnerID,
		"package":   packageCode,
	}
	if to != nil {
		params["to"] = to.UTC().Format("2006-01-02")
	}

	var response operationResponse
	if err := c.postQuery(ctx, "/activate-user", params, &response); err != nil {
		return err
	}
	if response.Status != 1 {
		return apiError(response.Error)
	}

	return nil
}

// ActivateServices activa canales o bundles adicionales.
func (c *Client) ActivateServices(ctx context.Context, partnerID, bundle string, to *time.Time) error {
	params := map[string]string{
		"userId":   partnerID,
		"services": bundle,
	}
	if to != nil {
		params["to"] = to.UTC().Format("2006-01-02")
	}

	var response operationResponse
	if err := c.postQuery(ctx, "/activate-services", params, &response); err != nil {
		return err
	}
	if response.Status != 1 {
		return apiError(response.Error)
	}

	return nil
}

// RemoveUser elimina el usuario en la API externa.
func (c *Client) RemoveUser(ctx context.Context, partnerID string, userID int64) (partnerusers.OperationStatus, error) {
	params := map[string]string{}
	if strings.TrimSpace(partnerID) != "" {
		params["partnerid"] = partnerID
	} else if userID > 0 {
		params["userId"] = strconv.FormatInt(userID, 10)
	} else {
		return partnerusers.OperationStatus{}, fmt.Errorf("%w: missing partner id or user id", partnerusers.ErrValidation)
	}

	var response operationResponse
	if err := c.get(ctx, "/remove-user", params, &response); err != nil {
		return partnerusers.OperationStatus{}, err
	}

	return partnerusers.OperationStatus{
		OK:     response.Status == 1,
		Status: response.Status,
		Error:  response.Error,
	}, nil
}

// DeactivateUser desactiva el usuario en la API externa.
func (c *Client) DeactivateUser(ctx context.Context, partnerID string, all bool) (partnerusers.OperationStatus, error) {
	params := map[string]string{
		"partnerid": partnerID,
	}
	if all {
		params["all"] = "1"
	}

	var response operationResponse
	if err := c.get(ctx, "/deactivate-user", params, &response); err != nil {
		return partnerusers.OperationStatus{}, err
	}

	return partnerusers.OperationStatus{
		OK:     response.Status == 1,
		Status: response.Status,
		Error:  response.Error,
	}, nil
}

// GetUser consulta el detalle de un usuario en la API externa.
func (c *Client) GetUser(ctx context.Context, partnerID string, userID int64) (partnerusers.ExternalUserDetails, error) {
	if strings.TrimSpace(partnerID) == "" && userID <= 0 {
		return partnerusers.ExternalUserDetails{}, fmt.Errorf("%w: user identifier is required", partnerusers.ErrValidation)
	}

	candidates := make([]map[string]string, 0, 2)
	if strings.TrimSpace(partnerID) != "" {
		candidates = append(candidates,
			map[string]string{"userId": partnerID},
			map[string]string{"partnerid": partnerID},
		)
	} else {
		candidates = append(candidates, map[string]string{"userId": strconv.FormatInt(userID, 10)})
	}

	var lastErr error
	for _, params := range candidates {
		var response getUserResponse
		if err := c.get(ctx, "/get-user", params, &response); err != nil {
			lastErr = err
			continue
		}
		if response.Status == 1 && response.User != nil {
			return normalizeExternalUser(*response.User), nil
		}
		if response.Error == "" {
			response.Error = "invalid get-user response"
		}
		lastErr = apiError(response.Error)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%w: invalid get-user response", partnerusers.ErrConflict)
	}
	return partnerusers.ExternalUserDetails{}, lastErr
}

// GetSTBs devuelve los STBs asociados al usuario.
func (c *Client) GetSTBs(ctx context.Context, partnerID string) ([]partnerusers.STBDevice, error) {
	var response devicesResponse
	if err := c.get(ctx, "/get-stbs", map[string]string{"partnerid": partnerID}, &response); err != nil {
		return nil, err
	}
	if response.Status != 1 && len(response.STBs) == 0 {
		return nil, apiError(response.Error)
	}
	return normalizeSTBs(response.STBs), nil
}

// GetSmartTVs devuelve los Smart TVs asociados al usuario.
func (c *Client) GetSmartTVs(ctx context.Context, partnerID string) ([]partnerusers.STBDevice, error) {
	var response devicesResponse
	if err := c.get(ctx, "/get-smart-tvs", map[string]string{"partnerid": partnerID}, &response); err != nil {
		return nil, err
	}
	if response.Status != 1 && len(response.STBs) == 0 {
		return nil, apiError(response.Error)
	}
	return normalizeSTBs(response.STBs), nil
}

// RemoveDevice expulsa un dispositivo concreto.
func (c *Client) RemoveDevice(ctx context.Context, deviceID string) error {
	var response operationResponse
	if err := c.get(ctx, "/remove-device", map[string]string{"deviceId": deviceID}, &response); err != nil {
		return err
	}
	if response.Status != 1 {
		return apiError(response.Error)
	}
	return nil
}

// GenerateLoginToken obtiene un token de acceso directo para el usuario final.
func (c *Client) GenerateLoginToken(ctx context.Context, userID int64, redirectToServices bool) (string, error) {
	params := map[string]string{
		"userId": strconv.FormatInt(userID, 10),
	}
	if redirectToServices {
		params["redirectToServices"] = "true"
	}

	var response generateTokenResponse
	if err := c.postForm(ctx, "/generate-user-login-token", params, &response); err != nil {
		return "", err
	}
	if response.Status != 1 || strings.TrimSpace(response.Token) == "" {
		return "", apiError(response.Error)
	}

	return response.Token, nil
}

func (c *Client) get(ctx context.Context, path string, params map[string]string, dst any) error {
	query := url.Values{}
	query.Set("partner", c.partner)
	query.Set("password", c.password)
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}

	endpoint := c.baseURL + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	return c.doJSON(req, dst)
}

func (c *Client) postQuery(ctx context.Context, path string, params map[string]string, dst any) error {
	query := url.Values{}
	query.Set("partner", c.partner)
	query.Set("password", c.password)
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}

	endpoint := c.baseURL + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}

	return c.doJSON(req, dst)
}

func (c *Client) postForm(ctx context.Context, path string, params map[string]string, dst any) error {
	form := url.Values{}
	form.Set("partner", c.partner)
	form.Set("password", c.password)
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			form.Set(key, value)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.doJSON(req, dst)
}

func (c *Client) doJSON(req *http.Request, dst any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("partnerapi: unexpected status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return err
	}
	return nil
}

func apiError(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("%w: external api error", partnerusers.ErrConflict)
	}
	switch message {
	case "not found", "unknown user":
		return fmt.Errorf("%w: %s", partnerusers.ErrNotFound, message)
	case "missing id":
		return fmt.Errorf("%w: %s", partnerusers.ErrValidation, message)
	default:
		return fmt.Errorf("%w: %s", partnerusers.ErrConflict, message)
	}
}

func normalizeExternalUser(user externalUser) partnerusers.ExternalUserDetails {
	services := make([]partnerusers.ExternalService, 0, len(user.Services))
	for _, service := range user.Services {
		services = append(services, partnerusers.ExternalService{
			Type: service.Type,
			Code: service.Code,
			From: parseAPITimePtr(service.From),
			To:   parseAPITimePtr(service.To),
		})
	}

	devices := make([]partnerusers.ExternalDevice, 0, len(user.Devices))
	for _, device := range user.Devices {
		devices = append(devices, partnerusers.ExternalDevice{
			ID:        firstNonEmpty(device.ID, device.DeviceID),
			Type:      device.Type,
			Title:     firstNonEmpty(device.Title, device.Name),
			LastLogin: parseAPITimePtr(device.LastLogin),
			LastIP:    firstNonEmpty(device.LastIP, device.IP),
			CreatedAt: parseAPITimePtr(device.Created),
			Enabled:   device.Enabled == 1 || strings.EqualFold(device.EnabledText, "true"),
		})
	}

	return partnerusers.ExternalUserDetails{
		FullName: user.FullName,
		Services: services,
		Devices:  devices,
	}
}

func normalizeSTBs(items []stbItem) []partnerusers.STBDevice {
	devices := make([]partnerusers.STBDevice, 0, len(items))
	for _, item := range items {
		deviceID := firstNonEmpty(item.DeviceID, item.ID)
		if strings.TrimSpace(deviceID) == "" {
			continue
		}
		devices = append(devices, partnerusers.STBDevice{DeviceID: deviceID})
	}
	return devices
}

func parseAPITimePtr(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0000-00-00 00:00:00" || raw == "0000-00-00" {
		return nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			value := parsed.UTC()
			return &value
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

type operationResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type generateTokenResponse struct {
	Status int    `json:"status"`
	Token  string `json:"token"`
	Error  string `json:"error"`
}

type getUserResponse struct {
	Status int           `json:"status"`
	Error  string        `json:"error"`
	User   *externalUser `json:"user"`
}

type externalUser struct {
	FullName string            `json:"fullName"`
	Services []externalService `json:"services"`
	Devices  []externalDevice  `json:"devices"`
}

type externalService struct {
	Type string `json:"type"`
	Code string `json:"code"`
	From string `json:"from"`
	To   string `json:"to"`
}

type externalDevice struct {
	ID          string `json:"id"`
	DeviceID    string `json:"deviceId"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Name        string `json:"name"`
	LastLogin   string `json:"lastLogin"`
	LastIP      string `json:"lastIp"`
	IP          string `json:"ip"`
	Created     string `json:"created"`
	Enabled     int    `json:"enabled"`
	EnabledText string `json:"enabled_text"`
}

type devicesResponse struct {
	Status int       `json:"status"`
	Error  string    `json:"error"`
	STBs   []stbItem `json:"stbs"`
}

type stbItem struct {
	ID       string `json:"id"`
	DeviceID string `json:"deviceId"`
}
