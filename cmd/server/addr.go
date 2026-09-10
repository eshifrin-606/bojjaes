package main

// The platform decides where we listen; 8080 is only the local convenience.
const defaultPort = "8080"

// resolveAddr takes the environment lookup as an argument rather than calling
// os.Getenv, so a test can supply a fake without touching the process
// environment.
func resolveAddr(getenv func(string) string) string {
	port := getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return ":" + port
}
