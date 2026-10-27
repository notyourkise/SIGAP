package routes

import (
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestWebSocketHandshakeRejectsMissingToken(t *testing.T) {
	app := fiber.New()
	SetupWebSocketRoutes(app)

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWebSocketHandshakeAcceptsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "websocket-test-secret")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := "22222222-2222-2222-2222-222222222222"
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	token, err := utils.GenerateToken(userID, "admin", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupWebSocketRoutes(app)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	url := "ws://" + listener.Addr().String() + "/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	if err == nil {
		_ = conn.Close()
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestWebSocketHandshakeLoadsTechnicianSiteMembership memastikan koneksi
// teknisi memuat penugasan user_sites sebelum didaftarkan ke Hub, sehingga
// broadcast per site dapat difilter dengan benar.
func TestWebSocketHandshakeLoadsTechnicianSiteMembership(t *testing.T) {
	t.Setenv("JWT_SECRET", "websocket-test-secret")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := "33333333-3333-3333-3333-333333333333"
	siteID := uuid.New()
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`^SELECT "site_id" FROM "user_sites" WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"site_id"}).AddRow(siteID.String()))

	token, err := utils.GenerateToken(userID, "teknisi", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupWebSocketRoutes(app)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	// Pastikan koneksi dari test sebelumnya sudah dilepas agar penambahan
	// koneksi baru dapat diamati dengan andal.
	assert.Eventually(t, func() bool {
		return utils.Hub.ActiveConnections() == 0
	}, time.Second, 10*time.Millisecond)

	url := "ws://" + listener.Addr().String() + "/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.NoError(t, err)
	assert.Eventually(t, func() bool {
		return utils.Hub.ActiveConnections() == 1
	}, time.Second, 10*time.Millisecond)

	if err == nil {
		_ = conn.Close()
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}
