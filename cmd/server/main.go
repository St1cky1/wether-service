package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/St1cky1/wether-service/internal/client/http/geocoding"
	openmeteo "github.com/St1cky1/wether-service/internal/client/http/open_meteo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
	// "github.com/golang/protobuf/ptypes/timestamp"
)

const (
	portHTTP = ":3000"
	city     = "Moscow"
)

type Reading struct { // показания погоды
	Timestamp   time.Time
	Temperature float64
}

type Storage struct { // временное хранилище показаний
	data map[string][]Reading
	mu   sync.RWMutex
}

func main() {

	r := chi.NewRouter()     //  создакм новый роутер, через него поднимаем сервер
	r.Use(middleware.Logger) // логируем запросы

	storage := &Storage{
		data: make(map[string][]Reading),
	}

	r.Get("/{city}", func(w http.ResponseWriter, r *http.Request) {

		cityName := chi.URLParam(r, "city")          // получаем параметр из урла
		fmt.Printf("Requested city: %s\n", cityName) // лонируем запрос

		storage.mu.RLock()
		defer storage.mu.RUnlock()

		reading, ok := storage.data[cityName] // получаем показания из хранилища
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
			return
		}

		raw, err := json.Marshal(reading)
		if err != nil {
			log.Println(err)
		}

		w.Header().Set("Content-Type", "application/json") // укзываем, что данные вернем в формате json
		_, err = w.Write(raw)
		if err != nil {
			log.Println(err)
		}
	})

	s, err := gocron.NewScheduler() // создаем новый экземпляр планировщика
	if err != nil {
		panic(err)
	}

	jobs, err := initJobs(s, storage) // вызываем "cron"
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		fmt.Println("Starting HTTP server on port", portHTTP)

		if err := http.ListenAndServe(portHTTP, r); err != nil { // поднимаем сервер
			panic(err)
		}
	}()

	go func() {
		defer wg.Done()

		fmt.Printf("Starting job: %v\n", jobs[0].ID()) // проверяем, что все работает (возвращает уникальный идентификатор задания.)

		s.Start() // стартуем приложение cron
	}()

	wg.Wait()
}

func initJobs(sheduler gocron.Scheduler, storage *Storage) ([]gocron.Job, error) { // принемаем сам планировщик, возвращаем слайс job и ошибку

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	geocodingClient := geocoding.NewClient(httpClient) // создаем клиента для получекния коорлинат
	openmeteoClient := openmeteo.NewClient(httpClient) // создаем клиента для получения температуры

	j, err := sheduler.NewJob( // j содержит job`у (новое задание в планировщике)
		gocron.DurationJob(
			10*time.Second, // задаем интервал срабатывания
		),
		gocron.NewTask(
			func() {
				geoResp, err := geocodingClient.GetCoords(city) // эндпоинт для получения координат
				if err != nil {
					log.Println(err)
					return
				}

				openMeteoResp, err := openmeteoClient.GetTemperature(geoResp.Latitude, geoResp.Longitude)
				if err != nil {
					log.Println(err)
					return
				}

				storage.mu.Lock()
				defer storage.mu.Unlock()

				timestamp, err := time.Parse("2006-01-02T15:04", openMeteoResp.Current.Time)
				if err != nil {
					log.Println(err)
				}

				storage.data[city] = append(storage.data[city], Reading{
					Timestamp:   timestamp,
					Temperature: openMeteoResp.Current.Temperature2m,
				})

				fmt.Printf("%v Update data for city: %s\n", time.Now(), city)
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return []gocron.Job{j}, nil
}
