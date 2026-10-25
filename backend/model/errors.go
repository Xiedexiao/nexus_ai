package model

import "errors"

// Common errors
var (
	ErrDatabase = errors.New("database error")
)

// User auth errors
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserEmptyCredentials = errors.New("empty credentials")
	ErrEmailAlreadyTaken    = errors.New("email already taken")
	ErrEmailNotFound        = errors.New("email not found")
	ErrEmailAmbiguous       = errors.New("email matches multiple users")
	// ErrRegistrationIpLimitReached reports that the calling network already
	// holds the configured maximum number of self-registered accounts.
	ErrRegistrationIpLimitReached = errors.New("registration ip limit reached")
	// ErrRegistrationWindowReached reports that the deployment-wide window
	// already admitted its configured maximum number of self-registered
	// accounts and more must wait for the window to slide.
	ErrRegistrationWindowReached = errors.New("registration window limit reached")
)

// Token auth errors
var (
	ErrTokenNotProvided = errors.New("token not provided")
	ErrTokenInvalid     = errors.New("token invalid")
)

// Redemption errors
var ErrRedeemFailed = errors.New("redeem.failed")

// 2FA errors
var ErrTwoFANotEnabled = errors.New("2fa not enabled")
var ErrTwoFAAlreadyEnabled = errors.New("2fa already enabled")
