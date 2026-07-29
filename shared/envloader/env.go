package envloader

import (
	"bufio"
	"os"
	"strings"
)

// Load читает .env-файлы и выставляет переменные окружения,
// не перезаписывая уже заданные значения (os.Getenv / compose / CI имеют приоритет).
// Отсутствие файла — не ошибка.
func Load(paths ...string) {
	if len(paths) == 0 {
		paths = []string{".env"}
	}
	for _, path := range paths {
		_ = loadFile(path)
	}
}

func loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := parseLine(line)
		if !ok {
			continue
		}
		if os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

func parseLine(line string) (key, value string, ok bool) {
	if strings.HasPrefix(line, "export ") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
	}
	idx := strings.IndexByte(line, '=')
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", false
	}
	value = strings.TrimSpace(line[idx+1:])
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return key, value, true
}
