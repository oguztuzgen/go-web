package main

import "os"

func GetEnv() map[string]string {
	env := map[string]string{}
	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		panic("env JWT_SECRET not set")
	}

	env["JWT_SECRET"] = jwtSecret
	return env
}
