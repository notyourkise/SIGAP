package utils

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// fakeRedisCommander adalah mock redisCommander untuk menguji
// RedisTokenBlacklist tanpa server Redis fisik (task verifikasi: mockable).
type fakeRedisCommander struct {
	mu     sync.Mutex
	set    map[string]map[string]struct{}
	expiry map[string]time.Duration
}

func newFakeRedisCommander() *fakeRedisCommander {
	return &fakeRedisCommander{
		set:    make(map[string]map[string]struct{}),
		expiry: make(map[string]time.Duration),
	}
}

func (f *fakeRedisCommander) SAdd(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.set[key]; !ok {
		f.set[key] = make(map[string]struct{})
	}
	added := int64(0)
	for _, m := range members {
		s, _ := m.(string)
		if _, ok := f.set[key][s]; !ok {
			f.set[key][s] = struct{}{}
			added++
		}
	}
	cmd := redis.NewIntCmd(context.Background(), "SADD")
	cmd.SetVal(added)
	return cmd
}

func (f *fakeRedisCommander) SIsMember(_ context.Context, key string, member interface{}) *redis.BoolCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.set[key][member.(string)]
	cmd := redis.NewBoolCmd(context.Background(), "SISMEMBER")
	cmd.SetVal(ok)
	return cmd
}

func (f *fakeRedisCommander) SRem(_ context.Context, key string, members ...interface{}) *redis.IntCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	removed := int64(0)
	for _, m := range members {
		s, _ := m.(string)
		if _, ok := f.set[key][s]; ok {
			delete(f.set[key], s)
			removed++
		}
	}
	cmd := redis.NewIntCmd(context.Background(), "SREM")
	cmd.SetVal(removed)
	return cmd
}

func (f *fakeRedisCommander) TTL(_ context.Context, key string) *redis.DurationCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	cmd := redis.NewDurationCmd(context.Background(), time.Second, "TTL")
	ttl, ok := f.expiry[key]
	if !ok {
		ttl = -1 * time.Second
	}
	cmd.SetVal(ttl)
	return cmd
}

func (f *fakeRedisCommander) Expire(_ context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.expiry[key] = expiration
	cmd := redis.NewBoolCmd(context.Background(), "EXPIRE")
	cmd.SetVal(true)
	return cmd
}

// TestTokenBlacklist_AddCheckRemoveViaInterface menguji kontrak
// TokenBlacklist (add → check → remove) melalui interface — dijalankan untuk
// varian Redis (mock) dan in-memory (fallback).
func TestTokenBlacklist_AddCheckRemoveViaInterface(t *testing.T) {
	tests := []struct {
		name    string
		factory func() TokenBlacklist
	}{
		{
			name: "Redis (mock redisCommander)",
			factory: func() TokenBlacklist {
				return &RedisTokenBlacklist{client: newFakeRedisCommander()}
			},
		},
		{
			name: "In-memory (fallback)",
			factory: func() TokenBlacklist {
				return newMemoryTokenBlacklist()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.factory()
			tokenA := "token-aaa"
			tokenB := "token-bbb"

			// Belum ada yang di-revoke
			revoked, err := store.IsRevoked(tokenA)
			assert.NoError(t, err)
			assert.False(t, revoked)
			revoked, err = store.IsRevoked(tokenB)
			assert.NoError(t, err)
			assert.False(t, revoked)

			// Add token A
			err = store.Add(tokenA, time.Now().Add(24*time.Hour))
			assert.NoError(t, err)
			revoked, err = store.IsRevoked(tokenA)
			assert.NoError(t, err)
			assert.True(t, revoked, "token yang sudah di-add harus terdeteksi revoked")
			revoked, err = store.IsRevoked(tokenB)
			assert.NoError(t, err)
			assert.False(t, revoked, "token lain tidak boleh terpengaruh")

			// Remove token A → tidak lagi revoked
			err = store.Remove(tokenA)
			assert.NoError(t, err)
			revoked, err = store.IsRevoked(tokenA)
			assert.NoError(t, err)
			assert.False(t, revoked, "token yang di-remove tidak boleh terdeteksi revoked")
		})
	}
}

// TestRedisBlacklist_TTLMengikutiTokenTerpanjang memverifikasi bahwa TTL key
// Redis mengikuti sisa masa hidup token dengan expiry terjauh (menambah token
// ber-TTL pendek tidak mempersingkat masa berlaku token lain).
func TestRedisBlacklist_TTLMengikutiTokenTerpanjang(t *testing.T) {
	fake := newFakeRedisCommander()
	store := &RedisTokenBlacklist{client: fake}

	// TTL yang tersimpan = time.Until(expiry), sedikit kurang dari TTL awal
	// karena waktu berlalu antara time.Now().Add(...) dan pemanggilan Add.
	// Karena itu perbandingan memakai toleransi ±1 detik.
	longTTL := 5 * time.Hour
	shortTTL := 30 * time.Minute
	extendedTTL := 10 * time.Hour

	assertTTLClose := func(label string, expected, actual time.Duration) {
		t.Helper()
		diff := expected - actual
		if diff < 0 {
			diff = -diff
		}
		assert.LessOrEqual(t, diff, time.Second, "%s: TTL expected ~%s, actual %s", label, expected, actual)
	}

	assert.NoError(t, store.Add("token-lama", time.Now().Add(longTTL)))
	assertTTLClose("setelah token long", longTTL, fake.expiry[tokenBlacklistKey])

	// Token baru dengan TTL lebih pendek → TTL key TIDAK berkurang
	assert.NoError(t, store.Add("token-baru", time.Now().Add(shortTTL)))
	assertTTLClose("setelah token short (tidak berkurang)", longTTL, fake.expiry[tokenBlacklistKey])

	// Token baru dengan TTL lebih panjang → TTL key diperpanjang
	assert.NoError(t, store.Add("token-panjang", time.Now().Add(extendedTTL)))
	assertTTLClose("setelah token extended", extendedTTL, fake.expiry[tokenBlacklistKey])
}

// TestBlacklist_RevokeAndCheck menguji facade publik RevokeToken/IsTokenRevoked
// dengan backend default in-memory (dipakai handler logout & middleware auth).
func TestBlacklist_RevokeAndCheck(t *testing.T) {
	tokenA := "token-aaa"
	tokenB := "token-bbb"

	assert.False(t, IsTokenRevoked(tokenA))
	assert.False(t, IsTokenRevoked(tokenB))

	RevokeToken(tokenA, time.Now().Add(24*time.Hour))

	assert.True(t, IsTokenRevoked(tokenA))
	assert.False(t, IsTokenRevoked(tokenB))
}

// TestBlacklist_CleanupExpiredTokens memverifikasi pembersihan token kadaluarsa
// pada backend in-memory (fallback).
func TestBlacklist_CleanupExpiredTokens(t *testing.T) {
	expiredToken := "token-expired"
	validToken := "token-valid"

	// Revoke dengan expiry di masa lalu → harus dibersihkan
	RevokeToken(expiredToken, time.Now().Add(-time.Minute))
	RevokeToken(validToken, time.Now().Add(24*time.Hour))

	assert.True(t, IsTokenRevoked(expiredToken))

	cleanupExpiredTokens()

	assert.False(t, IsTokenRevoked(expiredToken), "token yang sudah expire harus dibersihkan")
	assert.True(t, IsTokenRevoked(validToken), "token yang belum expire harus tetap ada")
}

// TestBlacklist_SetTokenBlacklist memverifikasi pergantian backend (main.go
// memanggil SetTokenBlacklist setelah ConnectRedis).
func TestBlacklist_SetTokenBlacklist(t *testing.T) {
	original := blacklistStore
	defer func() { blacklistStore = original }()

	fake := newFakeRedisCommander()
	store := &RedisTokenBlacklist{client: fake}
	SetTokenBlacklist(store)

	assert.NoError(t, store.Add("token-switch", time.Now().Add(time.Hour)))
	assert.True(t, IsTokenRevoked("token-switch"))
}
