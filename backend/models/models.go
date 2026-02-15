package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Save struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	RomName     string    `json:"rom_name"`
	SaveName    string    `json:"save_name"`
	IsNuzlocke  bool      `json:"is_nuzlocke"`
	HasSaveData bool      `json:"has_save_data"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NuzlockePokemon struct {
	ID           int    `json:"id"`
	SaveID       int    `json:"save_id"`
	PokemonName  string `json:"pokemon_name"`
	Nickname     string `json:"nickname"`
	Route        string `json:"route"`
	IsAlive      bool   `json:"is_alive"`
	CauseOfDeath string `json:"cause_of_death,omitempty"`
	LevelCaught  int    `json:"level_caught"`
	Notes        string `json:"notes,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

type CreateSaveRequest struct {
	RomName    string `json:"rom_name"`
	SaveName   string `json:"save_name"`
	IsNuzlocke bool   `json:"is_nuzlocke"`
}

type RomInfo struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	System   string `json:"system"`
}
