// Gateway EduPilot: `gateway serve` (mặc định) | `migrate` | `token` | `admin create` | cờ `-healthcheck`.
package main

import (
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) }

// run chọn lệnh con. Không có lệnh con (hoặc chỉ có cờ) → `serve`.
func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "serve":
			return runServe(args[1:], getenv, stdout, stderr)
		case "migrate":
			return runMigrate(args[1:], getenv, stdout, stderr)
		case "token":
			return runToken(args[1:], getenv, stdout, stderr)
		case "admin":
			return runAdmin(args[1:], getenv, os.Stdin, stdout, stderr)
		}
	}
	return runServe(args, getenv, stdout, stderr)
}
