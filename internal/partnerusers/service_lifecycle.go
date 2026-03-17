package partnerusers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Update porta el metodo update del controller PHP.
func (s *Service) Update(ctx context.Context, req UpdateRequest, authUser AuthUser) (UpdateResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return UpdateResult{}, err
	}
	if strings.TrimSpace(req.Target) == "" {
		return UpdateResult{}, fmt.Errorf("%w: target is required", ErrValidation)
	}

	targetPartnerID, err := s.resolveTargetPartnerID(ctx, req.Target)
	if err != nil {
		return UpdateResult{}, err
	}

	if req.Name != "" || req.Password != "" {
		if err := s.partnerAPI.ModifyUser(ctx, targetPartnerID, req.Name, req.Password); err != nil {
			return UpdateResult{}, err
		}
	}

	wpTable, attempted, err := s.pickTableForPartner(ctx, authUser.PartnerID)
	if err != nil {
		return UpdateResult{}, err
	}
	if wpTable == "" {
		return UpdateResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	baseRef := s.basePartner(authUser.PartnerID)
	baseTarget := s.basePartner(targetPartnerID)
	if baseRef != baseTarget {
		return UpdateResult{}, fmt.Errorf("%w: target partner does not belong to the authenticated ISP", ErrForbidden)
	}

	record, err := s.subscribers.FindByPartnerID(ctx, wpTable, targetPartnerID)
	if err != nil {
		return UpdateResult{}, err
	}
	if record == nil {
		return UpdateResult{}, fmt.Errorf("%w: subscriber not found", ErrNotFound)
	}

	updatePayload := SubscriberUpdate{
		LastUpdatedAt: timePtr(time.Now().UTC()),
	}
	if req.Name != "" {
		updatePayload.Name = stringPtr(req.Name)
	}

	if req.Package != "" {
		code, ok := s.packages.OriginalCode(req.Package)
		if !ok {
			return UpdateResult{}, fmt.Errorf("%w: unknown package %q", ErrValidation, req.Package)
		}

		var packageTo *time.Time
		if req.PackageActivationMode == "Fecha definida" {
			if req.DefinedPackageDate == nil {
				return UpdateResult{}, fmt.Errorf("%w: package defined date is required", ErrValidation)
			}
			check := s.vigenciaPolicy.ValidateDefinedDateForSwitch(record.StartDate, *req.DefinedPackageDate, record.EndDate == nil)
			if !check.OK {
				return UpdateResult{}, fmt.Errorf("%w: %s", ErrValidation, check.Error)
			}
			packageTo = copyTime(req.DefinedPackageDate)
		}

		if err := s.partnerAPI.ActivatePackage(ctx, targetPartnerID, code, packageTo); err != nil {
			return UpdateResult{}, err
		}

		updatePayload.Package = stringPtr(s.packages.ReadableName(code))
		updatePayload.EndDate = packageTo
	}

	channels := make([]ChannelActivation, 0, len(req.AdditionalChannels))
	for _, channel := range req.AdditionalChannels {
		if strings.TrimSpace(channel.Name) == "" {
			continue
		}

		channelDate := copyTime(channel.Date)
		if channelDate == nil && req.ChannelsActivationMode == "Fecha definida" {
			channelDate = copyTime(req.DefinedChannelsDate)
		}

		bundles := s.packages.ChannelsToBundles([]string{channel.Name})
		if len(bundles) == 0 {
			continue
		}

		if err := s.partnerAPI.ActivateServices(ctx, targetPartnerID, bundles[0], channelDate); err != nil {
			return UpdateResult{}, err
		}
		channels = append(channels, ChannelActivation{Name: channel.Name, Date: channelDate})
	}
	if len(channels) > 0 {
		updatePayload.AdditionalChannels = channels
	}

	if _, err := s.subscribers.UpdateByPartnerID(ctx, wpTable, targetPartnerID, updatePayload); err != nil {
		return UpdateResult{}, err
	}

	return UpdateResult{
		Message:   "Usuario actualizado correctamente",
		PartnerID: targetPartnerID,
		UsedTable: wpTable,
		Channels:  channels,
	}, nil
}

// ActivatePackage porta el metodo activate del controller PHP.
func (s *Service) ActivatePackage(ctx context.Context, req ActivatePackageRequest, authUser AuthUser) (ActivatePackageResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return ActivatePackageResult{}, err
	}
	if req.UserID <= 0 || strings.TrimSpace(req.Package) == "" {
		return ActivatePackageResult{}, fmt.Errorf("%w: user_id and package are required", ErrValidation)
	}

	partnerID, err := s.users.FindPartnerIDByID(ctx, req.UserID)
	if err != nil {
		return ActivatePackageResult{}, err
	}
	if !strings.Contains(partnerID, "_") {
		return ActivatePackageResult{}, fmt.Errorf("%w: invalid partner id for user %d", ErrNotFound, req.UserID)
	}

	wpTable, attempted, err := s.pickTableForPartner(ctx, partnerID)
	if err != nil {
		return ActivatePackageResult{}, err
	}
	if wpTable == "" {
		return ActivatePackageResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	record, err := s.subscribers.FindBySubscriberID(ctx, wpTable, req.UserID)
	if err != nil {
		return ActivatePackageResult{}, err
	}

	if req.DefinedEndDate != nil {
		minDate := s.vigenciaPolicy.BarrierFromStart(record.StartDate)
		if req.DefinedEndDate.Before(minDate) {
			return ActivatePackageResult{}, fmt.Errorf("%w: defined end date must be at least %s", ErrValidation, minDate.Format("2006-01-02"))
		}
	}

	if err := s.partnerAPI.ActivatePackage(ctx, partnerID, req.Package, copyTime(req.DefinedEndDate)); err != nil {
		return ActivatePackageResult{}, err
	}

	now := time.Now().UTC()
	if _, err := s.subscribers.UpdateBySubscriberID(ctx, wpTable, req.UserID, SubscriberUpdate{
		Package:       stringPtr(s.packages.ReadableName(req.Package)),
		EndDate:       copyTime(req.DefinedEndDate),
		LastUpdatedAt: &now,
	}); err != nil {
		return ActivatePackageResult{}, err
	}

	return ActivatePackageResult{Message: "Paquete activado correctamente"}, nil
}

// ActivateChannels porta el metodo activateChannels.
func (s *Service) ActivateChannels(ctx context.Context, req ActivateChannelsRequest, authUser AuthUser) (ActivatePackageResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return ActivatePackageResult{}, err
	}
	if req.PartnerID == "" || req.Channel == "" {
		return ActivatePackageResult{}, fmt.Errorf("%w: partner_id and channel are required", ErrValidation)
	}

	if err := s.partnerAPI.ActivateServices(ctx, req.PartnerID, req.Channel, copyTime(req.To)); err != nil {
		return ActivatePackageResult{}, err
	}

	return ActivatePackageResult{Message: "Canales adicionales activados correctamente."}, nil
}

// GetUserDetails porta el metodo getUserDetails.
func (s *Service) GetUserDetails(ctx context.Context, req UserDetailsRequest, authUser AuthUser) (UserDetailsResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return UserDetailsResult{}, err
	}
	if strings.TrimSpace(req.PartnerID) == "" {
		return UserDetailsResult{}, fmt.Errorf("%w: partner_id is required", ErrValidation)
	}

	externalUser, err := s.partnerAPI.GetUser(ctx, req.PartnerID, 0)
	if err != nil {
		return UserDetailsResult{}, err
	}

	var activePackage ExternalService
	for _, service := range externalUser.Services {
		if service.Type == "bundle" {
			activePackage = service
			break
		}
	}

	var activeDays *int
	if activePackage.From != nil {
		end := time.Now().UTC()
		if activePackage.To != nil {
			end = *activePackage.To
		}
		days := int(end.Sub(*activePackage.From).Hours() / 24)
		activeDays = &days
	}

	devices := make([]UserDevice, 0, len(externalUser.Devices))
	for _, device := range externalUser.Devices {
		devices = append(devices, UserDevice{
			Type:           defaultString(device.Type, "unknown"),
			Name:           defaultString(device.Title, "Sin nombre"),
			LastConnection: copyTime(device.LastLogin),
			IP:             device.LastIP,
			CreatedAt:      copyTime(device.CreatedAt),
		})
	}

	country := s.resolveCountryForPartnerID(ctx, req.PartnerID, authUser)
	earlySignal := s.earlyPolicy.Evaluate(country, strings.ToLower(strings.TrimSpace(authUser.Role)))

	return UserDetailsResult{
		FullName:                externalUser.FullName,
		ActivationDate:          copyTime(activePackage.From),
		ActiveDays:              activeDays,
		CurrentPackage:          s.packages.ReadableName(activePackage.Code),
		DeactivationDate:        copyTime(activePackage.To),
		Devices:                 devices,
		CanForceDeactivateEarly: earlySignal.CanForceDeactivateEarly,
		ForceDeactivatePolicy:   earlySignal,
	}, nil
}

// GetEarlyDeactivationStatus porta el metodo getEarlyDeactivationStatus.
func (s *Service) GetEarlyDeactivationStatus(_ context.Context, req EarlyDeactivationStatusRequest, authUser AuthUser) (EarlyDeactivationSignal, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return EarlyDeactivationSignal{}, err
	}
	country := req.Country
	if country == "" {
		country = authUser.Country
	}
	return s.earlyPolicy.Evaluate(strings.ToLower(country), strings.ToLower(authUser.Role)), nil
}

// Remove porta el metodo remove del controller PHP.
func (s *Service) Remove(ctx context.Context, req RemoveRequest, authUser AuthUser, clientIP string) (RemoveResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return RemoveResult{}, err
	}
	if req.UserID <= 0 {
		return RemoveResult{}, fmt.Errorf("%w: user_id is required", ErrValidation)
	}

	userRow, err := s.users.FindByID(ctx, req.UserID)
	if err != nil {
		return RemoveResult{}, err
	}
	if userRow == nil || strings.TrimSpace(userRow.Email) == "" {
		return RemoveResult{}, fmt.Errorf("%w: local user not found or missing email", ErrNotFound)
	}

	emailNorm := strings.ToLower(strings.TrimSpace(userRow.Email))
	wpTable, attempted, err := s.pickTableForPartner(ctx, userRow.PartnerID)
	if err != nil {
		return RemoveResult{}, err
	}
	if wpTable == "" {
		return RemoveResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	record, err := s.subscribers.FindByPartnerID(ctx, wpTable, userRow.PartnerID)
	if err != nil {
		return RemoveResult{}, err
	}

	runAt := s.vigenciaPolicy.RunAtForDeactivation(record.StartDate, record.EndDate)
	if runAt == nil {
		external, err := s.partnerAPI.RemoveUser(ctx, userRow.PartnerID, userRow.ID)
		if err != nil {
			return RemoveResult{}, err
		}

		now := time.Now().UTC()
		suspended := "suspendido"
		eliminado := "[ELIMINADO]"
		affected, err := s.subscribers.UpdateByEmailAndStates(ctx, wpTable, emailNorm, []string{record.Status, "activo", "inactivo", "suspendido"}, SubscriberUpdate{
			RemovedAt:     &now,
			LastUpdatedAt: &now,
			Status:        &suspended,
			PartnerID:     &eliminado,
		})
		if err != nil {
			return RemoveResult{}, err
		}

		return RemoveResult{
			Message:  "Usuario eliminado inmediatamente por VigenciaPolicy.",
			External: external,
			Local: LocalUpdateStatus{
				OK:       affected >= 1,
				Affected: affected,
				Table:    wpTable,
				By:       "email",
			},
		}, nil
	}

	if err := s.scheduler.ScheduleDeactivation(ctx, userRow.ID, authUser.ActorEmail(), clientIP, *runAt); err != nil {
		return RemoveResult{}, err
	}

	now := time.Now().UTC()
	if _, err := s.subscribers.UpdateByPartnerID(ctx, wpTable, userRow.PartnerID, SubscriberUpdate{
		EndDate:       copyTime(runAt),
		LastUpdatedAt: &now,
	}); err != nil {
		return RemoveResult{}, err
	}

	return RemoveResult{Message: "Eliminacion programada segun VigenciaPolicy."}, nil
}

// Deactivate porta el metodo deactivate del controller PHP.
func (s *Service) Deactivate(ctx context.Context, req DeactivateRequest, authUser AuthUser, clientIP string) (DeactivateResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return DeactivateResult{}, err
	}
	if req.UserID <= 0 {
		return DeactivateResult{}, fmt.Errorf("%w: user_id is required", ErrValidation)
	}

	targetUser, err := s.users.FindByID(ctx, req.UserID)
	if err != nil {
		return DeactivateResult{}, err
	}
	if targetUser == nil || !strings.Contains(targetUser.PartnerID, "_") {
		return DeactivateResult{}, fmt.Errorf("%w: could not resolve partner id", ErrNotFound)
	}

	wpTable, attempted, err := s.pickTableForPartner(ctx, targetUser.PartnerID)
	if err != nil {
		return DeactivateResult{}, err
	}
	if wpTable == "" {
		return DeactivateResult{}, fmt.Errorf("%w: no subscriber table found, attempted %v", ErrNotFound, attempted)
	}

	record, err := s.subscribers.FindByPartnerID(ctx, wpTable, targetUser.PartnerID)
	if err != nil {
		return DeactivateResult{}, err
	}

	targetCountry := s.resolveCountryForPartnerID(ctx, targetUser.PartnerID, authUser)
	earlySignal := s.earlyPolicy.Evaluate(targetCountry, strings.ToLower(strings.TrimSpace(authUser.Role)))

	if req.ForceEarly && !earlySignal.CanForceDeactivateEarly {
		s.auditEarlyDeactivationAttempt(ctx, req.UserID, targetUser.PartnerID, targetCountry, earlySignal, false, earlySignal.Reason, authUser)
		return DeactivateResult{}, fmt.Errorf("%w: early deactivation is not allowed", ErrForbidden)
	}
	if req.ForceEarly {
		s.auditEarlyDeactivationAttempt(ctx, req.UserID, targetUser.PartnerID, targetCountry, earlySignal, true, "", authUser)
	}

	var runAt *time.Time
	if !req.ForceEarly {
		runAt = s.vigenciaPolicy.RunAtForDeactivation(record.StartDate, record.EndDate)
	}

	if runAt == nil {
		external, err := s.partnerAPI.DeactivateUser(ctx, targetUser.PartnerID, true)
		if err != nil {
			return DeactivateResult{}, err
		}
		if !external.OK {
			return DeactivateResult{}, fmt.Errorf("%w: external API did not confirm deactivation", ErrConflict)
		}

		yesterday := time.Now().UTC().AddDate(0, 0, -1)
		now := time.Now().UTC()
		inactive := "inactivo"
		update := SubscriberUpdate{
			EndDate:       &yesterday,
			LastUpdatedAt: &now,
			Status:        &inactive,
		}
		if len(record.AdditionalChannels) > 0 {
			updatedChannels := make([]ChannelActivation, 0, len(record.AdditionalChannels))
			for _, channel := range record.AdditionalChannels {
				updatedChannels = append(updatedChannels, ChannelActivation{Name: channel.Name, Date: &yesterday})
			}
			update.AdditionalChannels = updatedChannels
		}

		if _, err := s.subscribers.UpdateByPartnerID(ctx, wpTable, targetUser.PartnerID, update); err != nil {
			return DeactivateResult{}, err
		}

		return DeactivateResult{Message: "Usuario desactivado correctamente (inmediato)."}, nil
	}

	if err := s.scheduler.ScheduleDeactivation(ctx, req.UserID, authUser.ActorEmail(), clientIP, *runAt); err != nil {
		return DeactivateResult{}, err
	}

	now := time.Now().UTC()
	if _, err := s.subscribers.UpdateByPartnerID(ctx, wpTable, targetUser.PartnerID, SubscriberUpdate{
		EndDate:       copyTime(runAt),
		LastUpdatedAt: &now,
	}); err != nil {
		return DeactivateResult{}, err
	}

	return DeactivateResult{
		Message: "Desactivacion programada.",
		UserID:  req.UserID,
		RunAt:   copyTime(runAt),
	}, nil
}
