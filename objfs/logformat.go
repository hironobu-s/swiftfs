package objfs

import (
	"bytes"

	"fmt"

	log "github.com/Sirupsen/logrus"
)

type LogfileFormatter struct {
}

func (l *LogfileFormatter) Format(entry *log.Entry) ([]byte, error) {

	b := &bytes.Buffer{}

	var keys []string = make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		fmt.Fprintf(b, "[%s] %s: %s", entry.Time.Format("Jan 02 15:04:05"), entry.Level, entry.Message)
	} else {
		var kv string
		for _, key := range keys {
			kv += fmt.Sprint("%s=%s", key, entry.Data[key])
		}
		fmt.Fprintf(b, "[%s] %s: %s (%s)", entry.Time.Format("Jan 02 15:04:05"), entry.Level, entry.Message, kv)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
