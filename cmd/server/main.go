package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron/v2"
)

const portHTTP = ":3000"

func main() {

	r := chi.NewRouter()     //  создакм новый роутер, через него поднимаем сервер
	r.Use(middleware.Logger) // логируем запросы
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("welcome"))
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
