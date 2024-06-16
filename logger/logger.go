package logger

import (
	"encoding/json"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const fileNameLog string = "log.json"

type RingMessStructLog struct {
	Mess      string    `json:"mess"`
	InputData string    `json:"inputData"`
	Datetime  time.Time `json:"datetime"`
	Success   bool      `json:"success"`
}

func SetupLogger() {
	// Настраиваем zerolog
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.000"
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: false})
}

func SetLog(mess string, inputData string, datetime time.Time, success bool) {
	// Создаем новый экземпляр структуры
	r := RingMessStructLog{
		Mess:      mess,
		InputData: inputData,
		Datetime:  datetime,
		Success:   success,
	}

	// Записываем данные в JSON-формате
	logData, err := json.Marshal(r)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal log data")
		return
	}

	// Открываем файл для записи
	f, err := os.OpenFile(fileNameLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open log file")
		return
	}
	defer f.Close()

	// Записываем данные в файл
	_, err = f.Write(append(logData, '\n'))
	if err != nil {
		log.Error().Err(err).Msg("Failed to write to log file")
		return
	}

}

func GetLog() {
	// Открываем файл для чтения
	f, err := os.Open(fileNameLog)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open log file")
		return
	}
	defer f.Close()

	// Читаем данные из файла
	var logs []RingMessStructLog
	decoder := json.NewDecoder(f)
	for {
		var r RingMessStructLog
		err := decoder.Decode(&r)
		if err != nil {
			break
		}
		logs = append(logs, r)
	}

	// Выводим данные в консоль
	for _, l := range logs {
		log.Info().
			Str("mess", l.Mess).
			Str("inputData", l.InputData).
			Time("datetime", l.Datetime).
			Bool("success", l.Success).
			Msg("Log entry")
	}
}
