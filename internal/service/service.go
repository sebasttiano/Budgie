package service

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sebasttiano/Budgie/internal/handlers"
	"github.com/sebasttiano/Budgie/internal/storage"
)

type Service struct {
	Store storage.Store
}

func NewService(store storage.Store) *Service {
	return &Service{
		Store: store,
	}
}

func (s *Service) InitRouter() chi.Router {

	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(handlers.WithLogging, handlers.GzipMiddleware)

	//r.Route("/", func(r chi.Router) {
	//	r.Get("/", s.MainHandle)
	//	r.Get("/ping", s.PingDB)
	//	r.Post("/updates/", s.UpdateMetricsJSON)
	//	r.Route("/value", func(r chi.Router) {
	//		r.Post("/", s.GetMetricJSON)
	//		r.Route("/{metricType}", func(r chi.Router) {
	//			r.Route("/{metricName}", func(r chi.Router) {
	//				r.Get("/", s.GetMetric)
	//			})
	//		})
	//	})
	//	r.Route("/update", func(r chi.Router) {
	//		r.Post("/", s.UpdateMetricJSON)
	//		r.Route("/{metricType}", func(r chi.Router) {
	//			r.Route("/{metricName}", func(r chi.Router) {
	//				r.Route("/{metricValue}", func(r chi.Router) {
	//					r.Post("/", s.UpdateMetric)
	//				})
	//			})
	//		})
	//	})
	//})
	return r
}
