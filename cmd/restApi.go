package main

import (
	"context"
	"errors"
	"github.com/mini-e-commerce-microservice/product-service/internal/conf"
	"github.com/mini-e-commerce-microservice/product-service/internal/presentations"
	"github.com/mini-e-commerce-microservice/product-service/internal/services"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"net/http"
	"os/signal"
	"syscall"
)

var restApiCmd = &cobra.Command{
	Use:   "restApi",
	Short: "run rest api",
	Run: func(cmd *cobra.Command, args []string) {
		appConf := conf.LoadAppConf()
		jwtConf := conf.LoadJwtConf()

		dependency, closeFn := services.NewDependency(appConf)

		server := presentations.New(&presentations.Presenter{
			ProductService:     dependency.ProductService,
			JwtAccessTokenConf: jwtConf.AccessToken,
			OutletService:      dependency.OutletService,
			Port:               int(appConf.AppPort),
		})

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go func() {
			if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
				log.Err(err).Msg("failed listen serve")
				stop()
			}
		}()

		<-ctx.Done()
		log.Info().Msg("Received shutdown signal, shutting down server gracefully...")

		if err := server.Shutdown(context.Background()); err != nil {
			log.Err(err).Msg("failed shutdown server")
		}

		_ = closeFn(context.Background())
		log.Info().Msg("Shutdown complete. Exiting.")
		return
	},
}
