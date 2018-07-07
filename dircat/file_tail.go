package dircat

import (
	"bufio"
	"os"
	"strings"
)

func TailFile(filename string, cnt int) (string, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return "", err
	}
	size := fi.Size()
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	cursor := size - 1
	var text []string
	if cnt < 1 {
		cnt = 10
	}
	for i := 0; i < cnt; i++ {
		if cursor < 0 {
			break
		}
		head, str, err := GetLineByPosition(cursor, file)
		if err != nil {
			return "", err
		}
		text = append([]string{str}, text...)
		cursor = head - 1
	}
	return strings.Join(text, "\n"), nil
}
func GetLineByPosition(cursor int64, file *os.File) (int64, string, error) {
	_, err := file.Seek(cursor, 0)
	if err != nil {
		return cursor, "", err
	}
	step, now, linelen := int64(50), cursor, 0
	//backtrack file pointer
	for {
		if now-step < 0 {
			now = 0
		} else {
			now = now - step
		}
		file.Seek(now, 0)
		scanner := bufio.NewScanner(file)
		scanner.Scan()
		linelen = len(scanner.Bytes())
		if int64(linelen)+now >= cursor && now != 0 {
			//backtrack until now+linelen<cursor
			continue
		}
		if now != 0 {
			now += int64(linelen) + 1
		}
		for {
			scanner = bufio.NewScanner(file)
			file.Seek(now, 0)
			scanner.Scan()
			linelen = len(scanner.Bytes())
			if now+int64(linelen) < cursor {
				now += int64(linelen) + 1
				continue
			} else {
				break
			}
		}
		break
	}
	data := make([]byte, linelen)
	file.ReadAt(data, now)
	return now, string(data), nil
}
