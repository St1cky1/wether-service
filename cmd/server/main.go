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
)

const portHTTP = ":3000"

func main() {

	r := chi.NewRouter()     //  создакм новый роутер, через него поднимаем сервер
	r.Use(middleware.Logger) // логируем запросы

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	geocodingClient := geocoding.NewClient(httpClient) // создаем клиента для получекния коорлинат
	openmeteoClient := openmeteo.NewClient(httpClient) // создаем клиента для получения температуры

	r.Get("/{city}", func(w http.ResponseWriter, r *http.Request) {

		city := chi.URLParam(r, "city")          // получаем параметр из урла
		fmt.Printf("Requested city: %s\n", city) // лонируем запрос

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

		raw, err := json.Marshal(openMeteoResp)
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

	jobs, err := initJobs(s) // вызываем "cron"
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

func initJobs(sheduler gocron.Scheduler) ([]gocron.Job, error) { // принемаем сам планировщик, возвращаем слайс job и ошибку

	j, err := sheduler.NewJob( // j содержит job`у (новое задание в планировщике)
		gocron.DurationJob(
			10*time.Second, // задаем интервал срабатывания
		),
		gocron.NewTask(
			func() {
				fmt.Println("Hello from cron!") // предоставляет функцию задачи и параметры задания
			},
		),
	)
	if err != nil {
		return nil, err
	}

	return []gocron.Job{j}, nil
}
