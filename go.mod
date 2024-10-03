// Declare the module path and list the versions of direct package dependencies
module github.com/davidwiese/fleet-tracker-backend

go 1.21.4

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
)

require filippo.io/edwards25519 v1.1.0 // indirect

// go.sum contains checksums of the module versions to verify integrity of downloads
