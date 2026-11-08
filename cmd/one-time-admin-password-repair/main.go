package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/db"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

const targetEmail = "superadmin@gmail.com"

func readHidden(prompt string) (string, error) {
	stateCmd := exec.Command("stty", "-g")
	stateCmd.Stdin = os.Stdin
	state, err := stateCmd.Output()
	if err != nil {
		return "", errors.New("could not read terminal settings")
	}

	setEcho := exec.Command("stty", "-echo")
	setEcho.Stdin = os.Stdin
	if err := setEcho.Run(); err != nil {
		return "", errors.New("could not disable terminal echo")
	}

	restore := func() error {
		cmd := exec.Command("stty", strings.TrimSpace(string(state)))
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	restored := false
	defer func() {
		if !restored {
			_ = restore()
		}
	}()

	fmt.Fprint(os.Stderr, prompt)
	line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')

	if err := restore(); err != nil {
		return "", errors.New("could not restore terminal settings")
	}
	restored = true

	fmt.Fprintln(os.Stderr)

	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", errors.New("could not read password")
	}

	return strings.TrimRight(line, "\r\n"), nil
}

func run() error {
	if err := godotenv.Load(); err != nil {
		return errors.New(".env could not be loaded")
	}

	config.ValidateConfig()

	password, err := readHidden("Enter the new admin password: ")
	if err != nil {
		return err
	}

	confirmation, err := readHidden("Confirm the new admin password: ")
	if err != nil {
		return err
	}

	if password == "" {
		return errors.New("password cannot be empty")
	}
	if password != confirmation {
		return errors.New("passwords do not match")
	}
	if len([]byte(password)) > 72 {
		return errors.New("password exceeds bcrypt's 72-byte limit")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		12,
	)
	if err != nil {
		return errors.New("password hashing failed")
	}

	conn, err := db.OpenDB()
	if err != nil {
		return errors.New("database connection failed")
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return errors.New("could not begin database transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT id::text
		FROM users
		WHERE LOWER(email) = LOWER($1)
		  AND deleted_at IS NULL
		  AND LOWER(BTRIM(role)) = 'admin'
		  AND UPPER(BTRIM(account_status)) = 'ACTIVE'
		FOR UPDATE
	`, targetEmail)
	if err != nil {
		return errors.New("could not verify target admin account")
	}

	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return errors.New("could not read target admin account")
		}
		ids = append(ids, id)
	}

	rowsErr := rows.Err()
	rows.Close()

	if rowsErr != nil {
		return errors.New("error while checking target admin account")
	}

	if len(ids) != 1 {
		return fmt.Errorf(
			"refusing update: expected exactly one active, non-deleted admin; found %d",
			len(ids),
		)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET password = $1
		WHERE id = $2
		  AND LOWER(email) = LOWER($3)
		  AND deleted_at IS NULL
		  AND LOWER(BTRIM(role)) = 'admin'
		  AND UPPER(BTRIM(account_status)) = 'ACTIVE'
	`, string(hash), ids[0], targetEmail)

	if err != nil {
		return errors.New("password update failed")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return errors.New("could not verify update result")
	}

	if affected != 1 {
		return fmt.Errorf(
			"refusing to commit: expected one updated row; got %d",
			affected,
		)
	}

	if err := tx.Commit(); err != nil {
		return errors.New("database transaction could not be committed")
	}

	fmt.Println("SUCCESS: existing admin password updated.")
	fmt.Println("Account:", targetEmail)
	fmt.Println("Role and account status were not changed.")

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "RECOVERY FAILED:", err)
		os.Exit(1)
	}
}
