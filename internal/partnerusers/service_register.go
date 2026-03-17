package partnerusers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Register porta el metodo register del controller PHP, pero ahora
// separado de HTTP y expresado como caso de uso.
func (s *Service) Register(ctx context.Context, req RegisterRequest, authUser AuthUser) (RegisterResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return RegisterResult{}, err
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Package) == "" ||
		strings.TrimSpace(req.City) == "" {
		return RegisterResult{}, fmt.Errorf("%w: email, password, name, package and city are required", ErrValidation)
	}

	if req.PackageActivationMode == "Fecha definida" {
		if req.DefinedPackageDate == nil {
			return RegisterResult{}, fmt.Errorf("%w: package defined date is required", ErrValidation)
		}
		minDate := s.vigenciaPolicy.MinDefinedDate(nil)
		if req.DefinedPackageDate.Before(minDate) {
			return RegisterResult{}, fmt.Errorf("%w: package defined date must be at least %s", ErrValidation, minDate.Format("2006-01-02"))
		}
	}

	newPartnerID, wpTable, attempted, err := s.computeNextPartnerIDFor(ctx, authUser.PartnerID)
	if err != nil {
		return RegisterResult{}, err
	}
	if newPartnerID == "" || wpTable == "" {
		return RegisterResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	country := authUser.Country
	if country == "" {
		country = "co"
	}

	apiResult, err := s.partnerAPI.RegisterUser(ctx, req, newPartnerID, country)
	if err != nil {
		if strings.Contains(err.Error(), "EMAIL_ALREADY_EXISTS") {
			return RegisterResult{}, fmt.Errorf("%w: email already exists", ErrConflict)
		}
		return RegisterResult{}, err
	}

	var channels []ChannelActivation
	var packageEndDate *time.Time
	if req.PackageActivationMode == "Fecha definida" {
		packageEndDate = copyTime(req.DefinedPackageDate)
	}

	for _, channel := range req.AdditionalChannels {
		bundleNames := s.packages.ChannelsToBundles([]string{channel})
		if len(bundleNames) == 0 {
			continue
		}
		if err := s.partnerAPI.ActivateServices(ctx, newPartnerID, bundleNames[0], packageEndDate); err != nil {
			return RegisterResult{}, err
		}
		channels = append(channels, ChannelActivation{Name: channel, Date: copyTime(packageEndDate)})
	}

	if err := s.subscribers.CreateMirroredUser(
		ctx,
		wpTable,
		req,
		newPartnerID,
		apiResult.ReadablePackage,
		channels,
		apiResult.ExternalID,
		packageEndDate,
	); err != nil {
		return RegisterResult{}, err
	}

	if err := s.users.SaveRegisteredUser(ctx, req, newPartnerID, apiResult.ReadablePackage, country, channels); err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		Message:           "Usuario registrado correctamente",
		AssignedPartnerID: newPartnerID,
		UsedTable:         wpTable,
		ExternalID:        apiResult.ExternalID,
	}, nil
}

// ReactivateDeletedUser porta el metodo reactivateDeletedUser del controller.
func (s *Service) ReactivateDeletedUser(ctx context.Context, req ReactivateDeletedUserRequest, authUser AuthUser) (RegisterResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return RegisterResult{}, err
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Package) == "" ||
		strings.TrimSpace(req.City) == "" {
		return RegisterResult{}, fmt.Errorf("%w: email, password, name, package and city are required", ErrValidation)
	}

	if req.PackageActivationMode == "Fecha definida" {
		if req.DefinedPackageDate == nil {
			return RegisterResult{}, fmt.Errorf("%w: package defined date is required", ErrValidation)
		}
		minDate := time.Now().UTC().AddDate(0, 1, 0)
		if req.DefinedPackageDate.Before(minDate) {
			return RegisterResult{}, fmt.Errorf("%w: package defined date must be at least %s", ErrValidation, minDate.Format("2006-01-02"))
		}
	}

	newPartnerID, wpTable, attempted, err := s.computeNextPartnerIDFor(ctx, authUser.PartnerID)
	if err != nil {
		return RegisterResult{}, err
	}
	if newPartnerID == "" || wpTable == "" {
		return RegisterResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	record, err := s.subscribers.FindByEmailAndStates(ctx, wpTable, req.Email, []string{"suspendido", "inactivo"})
	if err != nil {
		return RegisterResult{}, err
	}
	if record == nil {
		return RegisterResult{}, fmt.Errorf("%w: deleted subscriber not found", ErrNotFound)
	}

	registerReq := RegisterRequest{
		Email:                 req.Email,
		Password:              req.Password,
		Name:                  req.Name,
		Package:               req.Package,
		AdditionalChannels:    req.AdditionalChannels,
		PackageActivationMode: req.PackageActivationMode,
		DefinedPackageDate:    copyTime(req.DefinedPackageDate),
		Department:            req.Department,
		City:                  req.City,
	}

	country := authUser.Country
	if country == "" {
		country = "co"
	}

	apiResult, err := s.partnerAPI.RegisterUser(ctx, registerReq, newPartnerID, country)
	if err != nil {
		return RegisterResult{}, err
	}

	var channels []ChannelActivation
	var packageEndDate *time.Time
	if req.PackageActivationMode == "Fecha definida" {
		packageEndDate = copyTime(req.DefinedPackageDate)
	}

	for _, channel := range req.AdditionalChannels {
		bundleNames := s.packages.ChannelsToBundles([]string{channel})
		if len(bundleNames) == 0 {
			continue
		}
		if err := s.partnerAPI.ActivateServices(ctx, newPartnerID, bundleNames[0], packageEndDate); err != nil {
			return RegisterResult{}, err
		}
		channels = append(channels, ChannelActivation{Name: channel, Date: copyTime(packageEndDate)})
	}

	now := time.Now().UTC()
	activeStatus := "activo"
	if _, err := s.subscribers.UpdateByEmailAndStates(ctx, wpTable, req.Email, []string{"inactivo", "suspendido"}, SubscriberUpdate{
		PartnerID:          stringPtr(newPartnerID),
		Status:             stringPtr(activeStatus),
		Name:               stringPtr(req.Name),
		Package:            stringPtr(apiResult.ReadablePackage),
		StartDate:          &now,
		EndDate:            packageEndDate,
		Department:         stringPtr(req.Department),
		City:               stringPtr(req.City),
		AdditionalChannels: channels,
		LastUpdatedAt:      &now,
	}); err != nil {
		return RegisterResult{}, err
	}

	if err := s.users.SaveRegisteredUser(ctx, registerReq, newPartnerID, apiResult.ReadablePackage, country, channels); err != nil {
		return RegisterResult{}, err
	}

	return RegisterResult{
		Message:           "Usuario reactivado correctamente",
		AssignedPartnerID: newPartnerID,
		UsedTable:         wpTable,
		ExternalID:        apiResult.ExternalID,
	}, nil
}
