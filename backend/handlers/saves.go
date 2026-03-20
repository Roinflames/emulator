package handlers

import (
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pokemon-web/database"
	"pokemon-web/models"

	"github.com/gofiber/fiber/v2"
)

func detectSystem(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".gb":
		return "gb"
	case ".gbc":
		return "gb"
	case ".gba":
		return "gba"
	case ".nds":
		return "nds"
	default:
		return "unknown"
	}
}

func ListRoms(c *fiber.Ctx) error {
	rows, err := database.DB.Query("SELECT id, filename, size, system FROM roms ORDER BY filename")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to query roms"})
	}
	defer rows.Close()

	var roms []models.RomInfo
	for rows.Next() {
		var r models.RomInfo
		if err := rows.Scan(&r.ID, &r.Filename, &r.Size, &r.System); err != nil {
			continue
		}
		roms = append(roms, r)
	}
	if roms == nil {
		roms = []models.RomInfo{}
	}
	return c.JSON(roms)
}

func UploadRom(c *fiber.Ctx) error {
	file, err := c.FormFile("rom")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rom file required"})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	validExts := map[string]bool{".gb": true, ".gbc": true, ".gba": true, ".nds": true}
	if !validExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid file type, must be .gb, .gbc, .gba or .nds"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read file"})
	}

	system := detectSystem(file.Filename)
	_, err = database.DB.Exec(
		"INSERT INTO roms (filename, size, system, data) VALUES ($1, $2, $3, $4) ON CONFLICT (filename) DO UPDATE SET data = $4, size = $2",
		file.Filename, len(data), system, data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to store rom"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "rom uploaded", "filename": file.Filename})
}

func ServeRom(c *fiber.Ctx) error {
	filename := c.Query("name")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name query param required"})
	}

	var data []byte
	err := database.DB.QueryRow("SELECT data FROM roms WHERE filename = $1", filename).Scan(&data)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rom not found"})
	}

	c.Set("Content-Type", "application/octet-stream")
	return c.Send(data)
}

func DeleteRom(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rom id"})
	}

	result, err := database.DB.Exec("DELETE FROM roms WHERE id = $1", id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete rom"})
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rom not found"})
	}
	return c.JSON(fiber.Map{"message": "rom deleted"})
}

func ListSaves(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	rows, err := database.DB.Query(
		`SELECT id, user_id, rom_name, save_name, is_nuzlocke, save_data IS NOT NULL, created_at, updated_at
		 FROM saves WHERE user_id = $1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to query saves"})
	}
	defer rows.Close()

	var saves []models.Save
	for rows.Next() {
		var s models.Save
		if err := rows.Scan(&s.ID, &s.UserID, &s.RomName, &s.SaveName, &s.IsNuzlocke, &s.HasSaveData, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		saves = append(saves, s)
	}

	if saves == nil {
		saves = []models.Save{}
	}
	return c.JSON(saves)
}

func CreateSave(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req models.CreateSaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.RomName == "" || req.SaveName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "rom_name and save_name are required"})
	}

	var id int
	err := database.DB.QueryRow(
		"INSERT INTO saves (user_id, rom_name, save_name, is_nuzlocke) VALUES ($1, $2, $3, $4) RETURNING id",
		userID, req.RomName, req.SaveName, req.IsNuzlocke).Scan(&id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create save"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id, "message": "save created"})
}

func GetSaveData(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	var data []byte
	err = database.DB.QueryRow("SELECT save_data FROM saves WHERE id = $1 AND user_id = $2", saveID, userID).Scan(&data)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}
	if data == nil {
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	c.Set("Content-Type", "application/octet-stream")
	return c.Send(data)
}

func UpdateSaveData(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	body := c.Body()
	if len(body) == 0 {
		bodyReader := c.Request().BodyStream()
		if bodyReader != nil {
			body, err = io.ReadAll(bodyReader)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to read body"})
			}
		}
	}

	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "empty save data"})
	}

	result, err := database.DB.Exec(
		"UPDATE saves SET save_data = $1, updated_at = $2 WHERE id = $3 AND user_id = $4",
		body, time.Now(), saveID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update save"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	return c.JSON(fiber.Map{"message": "save updated"})
}

func DeleteSave(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	saveID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid save id"})
	}

	result, err := database.DB.Exec("DELETE FROM saves WHERE id = $1 AND user_id = $2", saveID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete save"})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "save not found"})
	}

	return c.JSON(fiber.Map{"message": "save deleted"})
}
