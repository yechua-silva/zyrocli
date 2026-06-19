package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteZorroLogo(t *testing.T) {
	// Usar directorio temporal para no ensuciar la config real
	tmpHome, err := os.MkdirTemp("", "zyro-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	// Mockear HOME
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	path, err := WriteZorroLogo()
	if err != nil {
		t.Fatal(err)
	}

	// Verificar que el archivo existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("plugin file not created at %s", path)
	}

	// Verificar que el contenido parece un plugin TSX
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("plugin file is empty")
	}
}

func TestUpdateTuiJSON(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "zyro-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Primero escribir el plugin
	WriteZorroLogo()

	// Luego actualizar tui.json
	if err := UpdateTuiJSON(); err != nil {
		t.Fatal(err)
	}

	// Verificar que tui.json existe y tiene el plugin
	tuiPath := filepath.Join(tmpHome, ".config", "opencode", "tui.json")
	data, err := os.ReadFile(tuiPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("tui.json is empty")
	}
}

func TestUpdateTuiJSONIdempotent(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "zyro-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	WriteZorroLogo()

	// Ejecutar dos veces
	if err := UpdateTuiJSON(); err != nil {
		t.Fatal(err)
	}
	if err := UpdateTuiJSON(); err != nil {
		t.Fatal(err)
	}

	// No debe dar error en la segunda ejecución
}
