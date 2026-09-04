package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"marmara-cli/internal/api"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// loginCmd authenticates and caches tokens. Credentials come from flags/env or
// an interactive masked prompt; the raw password is never persisted.
func loginCmd() *cobra.Command {
	var username, password string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and cache tokens (~/.marmara/token.json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, m, err := deps()
			if err != nil {
				return err
			}
			if username == "" {
				username = os.Getenv("MARMARA_USERNAME")
			}
			if password == "" {
				password = os.Getenv("MARMARA_PASSWORD")
			}
			if username == "" {
				// Students log in with an "o" prefix before the student number
				// (e.g. o000000000), the same as BYS.
				username, err = prompt("Username (student: o + number, e.g. o000000000): ")
				if err != nil {
					return err
				}
			}
			if password == "" {
				password, err = promptSecret("Password: ")
				if err != nil {
					return err
				}
			}
			if username == "" || password == "" {
				return errors.New("username and password are required")
			}
			if err := m.Login(cmd.Context(), username, password); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "logged in; tokens cached.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "username; students use o + student number, e.g. o000000000 (or MARMARA_USERNAME)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password (or MARMARA_PASSWORD)")
	return cmd
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke token and clear local cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, m, err := deps()
			if err != nil {
				return err
			}
			if err := m.Logout(cmd.Context()); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "logged out.")
			return nil
		},
	}
}

// gradesCmd prints grades, with a `detail` subcommand for one course.
func gradesCmd() *cobra.Command {
	cmd := dataCmd("grades", "Your grade list", func(ctx context.Context, a *api.API) (json0, error) { return a.Grades(ctx) }, renderGrades)

	var dersID string
	detail := &cobra.Command{
		Use:   "detail",
		Short: "Grade component breakdown for one course (--ders-id)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dersID == "" {
				return errors.New("--ders-id is required (OgrenciDersId from `marmara grades`)")
			}
			a, _, err := deps()
			if err != nil {
				return err
			}
			raw, err := a.GradeDetail(cmd.Context(), dersID)
			if err != nil {
				return err
			}
			return emit(raw)
		},
	}
	detail.Flags().StringVar(&dersID, "ders-id", "", "OgrenciDersId")
	cmd.AddCommand(detail)
	return cmd
}

func campusMapsCmd() *cobra.Command {
	var types bool
	cmd := &cobra.Command{
		Use:   "campus-maps",
		Short: "Campus GPS markers and buildings (--types for categories)",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, _, err := deps()
			if err != nil {
				return err
			}
			var raw json0
			if types {
				raw, err = a.CampusMapTypes(cmd.Context())
			} else {
				raw, err = a.CampusMaps(cmd.Context())
			}
			if err != nil {
				return err
			}
			return emit(raw)
		},
	}
	cmd.Flags().BoolVar(&types, "types", false, "list building categories instead of markers")
	return cmd
}

func prompt(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return "", errors.New("no input")
	}
	return strings.TrimSpace(sc.Text()), nil
}

func promptSecret(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
