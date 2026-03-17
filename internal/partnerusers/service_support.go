package partnerusers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// QueueBulkUpload porta los metodos upload y bulkUsers a una forma desacoplada del framework.
func (s *Service) QueueBulkUpload(ctx context.Context, req QueueBulkUploadRequest, authUser AuthUser) (QueueBulkUploadResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return QueueBulkUploadResult{}, err
	}
	if s.uploads == nil || s.scheduler == nil {
		return QueueBulkUploadResult{}, fmt.Errorf("%w: upload pipeline is not configured", ErrConflict)
	}
	if req.Content == nil {
		return QueueBulkUploadResult{}, fmt.Errorf("%w: file content is required", ErrValidation)
	}

	fileName := req.FileName
	if fileName == "" {
		fileName = fmt.Sprintf("bulk_%d.json", time.Now().UnixNano())
	}
	path, err := s.uploads.Save(ctx, filepath.Base(fileName), req.Content)
	if err != nil {
		return QueueBulkUploadResult{}, err
	}

	if err := s.scheduler.DispatchBulkUsers(ctx, path, authUser.ActorEmail()); err != nil {
		return QueueBulkUploadResult{}, err
	}

	return QueueBulkUploadResult{
		Message: "Archivo recibido correctamente. Se procesara en segundo plano.",
		Path:    path,
	}, nil
}

// DeactivateMultiple porta deactivateMultiple iterando sobre el caso de uso Deactivate.
func (s *Service) DeactivateMultiple(ctx context.Context, req BulkOperationRequest, authUser AuthUser, clientIP string) ([]BulkResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return nil, err
	}

	results := make([]BulkResult, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		_, err := s.Deactivate(ctx, DeactivateRequest{UserID: userID}, authUser, clientIP)
		if err != nil {
			results = append(results, BulkResult{UserID: userID, Status: "error", Message: err.Error()})
			continue
		}
		results = append(results, BulkResult{UserID: userID, Status: "ok"})
	}

	return results, nil
}

// ChangePackageMultiple porta changePackageMultiple delegando en ActivatePackage.
func (s *Service) ChangePackageMultiple(ctx context.Context, req ChangePackageMultipleRequest, authUser AuthUser) ([]BulkResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.NewPackage) == "" {
		return nil, fmt.Errorf("%w: new_package is required", ErrValidation)
	}

	results := make([]BulkResult, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		_, err := s.ActivatePackage(ctx, ActivatePackageRequest{
			UserID:         userID,
			Package:        req.NewPackage,
			DefinedEndDate: copyTime(req.PackageEndDate),
		}, authUser)
		if err != nil {
			results = append(results, BulkResult{UserID: userID, Status: "error", Message: err.Error()})
			continue
		}
		results = append(results, BulkResult{UserID: userID, Status: "ok"})
	}

	return results, nil
}

// RemoveMultiple porta removeMultiple delegando en Remove.
func (s *Service) RemoveMultiple(ctx context.Context, req BulkOperationRequest, authUser AuthUser, clientIP string) ([]BulkResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return nil, err
	}

	results := make([]BulkResult, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		_, err := s.Remove(ctx, RemoveRequest{UserID: userID}, authUser, clientIP)
		if err != nil {
			results = append(results, BulkResult{UserID: userID, Status: "error", Message: err.Error()})
			continue
		}
		results = append(results, BulkResult{UserID: userID, Status: "ok"})
	}

	return results, nil
}

// GenerateUserLoginToken porta generateUserLoginToken.
func (s *Service) GenerateUserLoginToken(ctx context.Context, req GenerateLoginTokenRequest, authUser AuthUser) (GenerateLoginTokenResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return GenerateLoginTokenResult{}, err
	}
	if req.UserID <= 0 {
		return GenerateLoginTokenResult{}, fmt.Errorf("%w: user_id is required", ErrValidation)
	}

	token, err := s.partnerAPI.GenerateLoginToken(ctx, req.UserID, req.RedirectToServices)
	if err != nil {
		return GenerateLoginTokenResult{}, err
	}
	if token == "" {
		return GenerateLoginTokenResult{}, fmt.Errorf("%w: could not generate login token", ErrConflict)
	}

	base := s.partnerLoginBaseURL
	if base == "" {
		base = "https://nuplin.com"
	}
	return GenerateUserLoginTokenResult(base, token), nil
}

// GenerateUserLoginTokenByEmail porta generateUserLoginTokenEmail.
func (s *Service) GenerateUserLoginTokenByEmail(ctx context.Context, req GenerateLoginTokenByEmailRequest, authUser AuthUser) (LoginTokenByEmailResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return LoginTokenByEmailResult{}, err
	}
	if strings.TrimSpace(req.Email) == "" {
		return LoginTokenByEmailResult{}, fmt.Errorf("%w: email is required", ErrValidation)
	}

	userPartnerID, err := s.users.FindPartnerIDByEmail(ctx, req.Email)
	if err != nil {
		return LoginTokenByEmailResult{}, err
	}
	localUser, err := s.users.FindByPartnerID(ctx, userPartnerID)
	if err != nil {
		return LoginTokenByEmailResult{}, err
	}
	if localUser == nil {
		return LoginTokenByEmailResult{}, fmt.Errorf("%w: user not found", ErrNotFound)
	}

	result, err := s.GenerateUserLoginToken(ctx, GenerateLoginTokenRequest{UserID: localUser.ID}, authUser)
	if err != nil {
		return LoginTokenByEmailResult{}, err
	}
	return LoginTokenByEmailResult{RedirectURL: result.TokenURL}, nil
}

// LogoutAllDevices porta logoutAllDevices.
func (s *Service) LogoutAllDevices(ctx context.Context, req LogoutAllDevicesRequest, authUser AuthUser) (map[string]any, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return nil, err
	}
	if req.UserID <= 0 {
		return nil, fmt.Errorf("%w: user_id is required", ErrValidation)
	}

	user, err := s.users.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !strings.Contains(user.PartnerID, "_") {
		return nil, fmt.Errorf("%w: invalid partner id", ErrNotFound)
	}

	removed := make([]string, 0, 8)
	errorsByDevice := make([]map[string]any, 0, 4)

	removeDevice := func(deviceID string) {
		if deviceID == "" {
			return
		}
		if err := s.partnerAPI.RemoveDevice(ctx, deviceID); err != nil {
			errorsByDevice = append(errorsByDevice, map[string]any{
				"device_id": deviceID,
				"error":     err.Error(),
			})
			return
		}
		removed = append(removed, deviceID)
	}

	stbs, err := s.partnerAPI.GetSTBs(ctx, user.PartnerID)
	if err == nil {
		for _, stb := range stbs {
			removeDevice(stb.DeviceID)
		}
	}

	smartTVs, err := s.partnerAPI.GetSmartTVs(ctx, user.PartnerID)
	if err == nil {
		for _, tv := range smartTVs {
			removeDevice(tv.DeviceID)
		}
	}

	details, err := s.partnerAPI.GetUser(ctx, user.PartnerID, user.ID)
	if err == nil {
		for _, device := range details.Devices {
			if device.Enabled {
				removeDevice(device.ID)
			}
		}
	}

	return map[string]any{
		"message":    "Dispositivos procesados",
		"partner_id": user.PartnerID,
		"removed":    removed,
		"errors":     errorsByDevice,
	}, nil
}

func GenerateUserLoginTokenResult(baseURL, token string) GenerateLoginTokenResult {
	baseURL = strings.TrimRight(baseURL, "/")
	return GenerateLoginTokenResult{
		Message:  "Token generado correctamente",
		Token:    token,
		TokenURL: baseURL + "/partner-login?token=" + token,
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
