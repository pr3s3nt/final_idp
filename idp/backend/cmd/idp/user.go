package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"

	"idp/internal/authentication"
	"idp/internal/config"
	"idp/internal/persistence"
)

func runUser(ctx context.Context, cfg *config.Config, db *persistence.DB, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: idp user <create|reset-password|set-status> ...")
	}
	authKey := sha256.Sum256([]byte("login-rate:" + cfg.AuthHMACKey))
	svc, err := authentication.NewService(&persistence.AuthenticationRepository{DB: db}, authKey[:])
	if err != nil {
		return err
	}
	svc.AccountAttemptLimit = cfg.AuthAccountLimit
	svc.SourceAttemptLimit = cfg.AuthSourceLimit
	svc.AttemptWindow = cfg.AuthAttemptWindow
	switch args[0] {
	case "create":
		if len(args) != 3 {
			return errors.New("usage: idp user create <username> <display-name>")
		}
		password, err := promptConfirmedPassword("Password: ", "Confirm password: ")
		if err != nil {
			return err
		}
		return svc.CreateLocalUser(ctx, args[1], args[2], password)
	case "reset-password":
		if len(args) != 2 {
			return errors.New("usage: idp user reset-password <username>")
		}
		password, err := promptConfirmedPassword("New password: ", "Confirm new password: ")
		if err != nil {
			return err
		}
		return svc.ResetLocalPassword(ctx, args[1], password)
	case "set-status":
		if len(args) != 3 {
			return errors.New("usage: idp user set-status <username> <ACTIVE|DISABLED>")
		}
		return svc.SetLocalUserStatus(ctx, args[1], args[2])
	default:
		return fmt.Errorf("unknown user command %q", args[0])
	}
}

func promptConfirmedPassword(prompt, confirm string) (string, error) {
	first, err := readHidden(prompt)
	if err != nil {
		return "", err
	}
	second, err := readHidden(confirm)
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("passwords do not match")
	}
	return first, nil
}

func readHidden(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		body, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(body), err
	}
	return "", errors.New("password must be entered through an interactive terminal")
}
