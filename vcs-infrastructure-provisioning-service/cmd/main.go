package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpHandler "github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/api/http"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/databases"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/kafka"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/env"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/middlewares"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/services"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	envConfig, err := env.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load env: %v", err)
	}

	logger, err := logger.LoadLogger(envConfig.LoggerEnv)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	postgresDb, err := databases.ConnectPostgresDb(envConfig.PostgresEnv)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}

	if err := postgresDb.AutoMigrate(
		&entities.Infrastructure{},
		&entities.PostgreSQLInstance{},
		&entities.NginxInstance{},
		&entities.PostgreSQLCluster{},
		&entities.ClusterNode{},
		&entities.EtcdNode{},
		&entities.FailoverEvent{},
		&entities.NginxDomain{},
		&entities.NginxRoute{},
		&entities.NginxUpstream{},
		&entities.NginxUpstreamBackend{},
		&entities.NginxCertificate{},
		&entities.NginxSecurity{},
		&entities.PostgresDatabase{},
		&entities.PostgresBackup{},
		&entities.DockerService{},
		&entities.DockerEnvVar{},
		&entities.DockerPort{},
		&entities.DockerNetwork{},
		&entities.DockerHealthCheck{},
		&entities.Stack{},
		&entities.StackResource{},
		&entities.StackTemplate{},
		&entities.StackOperation{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	dockerService, err := docker.NewDockerService(logger)
	if err != nil {
		log.Fatalf("Failed to create docker service: %v", err)
	}

	kafkaProducer := kafka.NewKafkaProducer(envConfig.KafkaEnv, logger)
	defer kafkaProducer.Close()

	redisFactory := databases.NewRedisFactory(envConfig.RedisEnv)
	redisClient := redisFactory.ConnectRedis()

	infraRepo := repositories.NewInfrastructureRepository(postgresDb)
	pgRepo := repositories.NewPostgreSQLRepository(postgresDb)
	nginxRepo := repositories.NewNginxRepository(postgresDb)
	clusterRepo := repositories.NewPostgreSQLClusterRepository(postgresDb)
	pgDatabaseRepo := repositories.NewPostgresDatabaseRepository(postgresDb)
	dockerRepo := repositories.NewDockerServiceRepository(postgresDb)
	stackRepo := repositories.NewStackRepository(postgresDb)

	cacheService := services.NewCacheService(redisClient)
	pgService := services.NewPostgreSQLService(infraRepo, pgRepo, dockerService, kafkaProducer, logger)
	nginxService := services.NewNginxService(infraRepo, nginxRepo, dockerService, kafkaProducer, logger)
	clusterService := services.NewPostgreSQLClusterService(infraRepo, clusterRepo, dockerService, kafkaProducer, cacheService, logger)
	pgDatabaseService := services.NewPostgresDatabaseService(pgDatabaseRepo, pgRepo, dockerService)
	dockerSvcService := services.NewDockerServiceService(dockerRepo, infraRepo, dockerService)
	stackService := services.NewStackService(stackRepo, infraRepo, nginxService, pgService, clusterService, clusterRepo, pgDatabaseService, dockerSvcService)

	kafkaConsumer := kafka.NewEventConsumer(envConfig.KafkaEnv, cacheService, logger)
	defer kafkaConsumer.Close()

	ctx := context.Background()
	if err := kafkaConsumer.Start(ctx); err != nil {
		logger.Error("failed to start kafka consumer", zap.Error(err))
	}

	// Initialize WebSocket handler
	wsHandler := httpHandler.NewWebSocketHandler(logger)
	go wsHandler.Start()

	// Initialize Docker Event Listener
	eventListenerService := services.NewDockerEventListenerService(
		dockerService,
		kafkaProducer,
		infraRepo,
		logger,
	)
	eventListenerService.SetWebSocketHandler(wsHandler)
	if err := eventListenerService.Start(ctx); err != nil {
		logger.Error("failed to start docker event listener", zap.Error(err))
	}
	defer eventListenerService.Stop()

	jwtMiddleware := middlewares.NewJWTMiddleware(envConfig.AuthEnv.JWTSecret)

	pgHandler := httpHandler.NewPostgreSQLHandler(pgService)
	nginxHandler := httpHandler.NewNginxHandler(nginxService)
	clusterHandler := httpHandler.NewPostgreSQLClusterHandler(clusterService, logger)
	pgDatabaseHandler := httpHandler.NewPostgresDatabaseHandler(pgDatabaseService)
	dockerHandler := httpHandler.NewDockerServiceHandler(dockerSvcService)
	stackHandler := httpHandler.NewStackHandler(stackService)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001", "http://frontend.localhost"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// WebSocket endpoint (no auth required for now, can add later)
	r.GET("/ws", wsHandler.HandleWebSocket)

	apiV1 := r.Group("/api/v1", jwtMiddleware.CheckBearerAuth())
	pgHandler.RegisterRoutes(apiV1)
	nginxHandler.RegisterRoutes(apiV1)
	pgDatabaseHandler.RegisterRoutes(apiV1)
	dockerHandler.RegisterRoutes(apiV1)
	stackHandler.RegisterRoutes(apiV1)

	// PostgreSQL Cluster routes
	clusterGroup := apiV1.Group("/postgres/cluster")
	{
		clusterGroup.POST("", clusterHandler.CreateCluster)
		clusterGroup.GET("/:id", clusterHandler.GetClusterInfo)
		clusterGroup.POST("/:id/start", clusterHandler.StartCluster)
		clusterGroup.POST("/:id/stop", clusterHandler.StopCluster)
		clusterGroup.POST("/:id/restart", clusterHandler.RestartCluster)
		clusterGroup.DELETE("/:id", clusterHandler.DeleteCluster)
		clusterGroup.POST("/:id/scale", clusterHandler.ScaleCluster)
		clusterGroup.POST("/:id/failover", clusterHandler.PromoteReplica)
		clusterGroup.GET("/:id/replication", clusterHandler.GetReplicationStatus)
		clusterGroup.GET("/:id/stats", clusterHandler.GetClusterStats)
		clusterGroup.GET("/:id/logs", clusterHandler.GetClusterLogs)
		// Node management
		clusterGroup.POST("/:id/nodes/stop", clusterHandler.StopNode)
		clusterGroup.POST("/:id/nodes/start", clusterHandler.StartNode)
		clusterGroup.POST("/:id/nodes", clusterHandler.AddNode)
		clusterGroup.DELETE("/:id/nodes", clusterHandler.RemoveNode)
		clusterGroup.GET("/:id/failover-history", clusterHandler.GetFailoverHistory)
		// User management
		clusterGroup.POST("/:id/users", clusterHandler.CreateUser)
		clusterGroup.GET("/:id/users", clusterHandler.ListUsers)
		clusterGroup.DELETE("/:id/users/:username", clusterHandler.DeleteUser)
		clusterGroup.POST("/:id/databases", clusterHandler.CreateDatabase)
		clusterGroup.GET("/:id/databases", clusterHandler.ListDatabases)
		clusterGroup.DELETE("/:id/databases/:dbname", clusterHandler.DeleteDatabase)
		clusterGroup.PUT("/:id/config", clusterHandler.UpdateConfig)
		clusterGroup.GET("/:id/endpoints", clusterHandler.GetEndpoints)

		// Patroni Management routes
		clusterGroup.POST("/:id/patroni/switchover", clusterHandler.PatroniSwitchover)
		clusterGroup.POST("/:id/patroni/reinit", clusterHandler.PatroniReinit)
		clusterGroup.POST("/:id/patroni/pause", clusterHandler.PatroniPause)
		clusterGroup.POST("/:id/patroni/resume", clusterHandler.PatroniResume)
		clusterGroup.GET("/:id/patroni/status", clusterHandler.PatroniStatus)

		// Backup/Restore routes
		clusterGroup.POST("/:id/backup", clusterHandler.BackupCluster)
		clusterGroup.GET("/:id/backups", clusterHandler.ListBackups)
		clusterGroup.POST("/:id/restore", clusterHandler.RestoreCluster)

		// Query & Replication Test
		clusterGroup.POST("/:id/query", clusterHandler.ExecuteQuery)
		clusterGroup.POST("/:id/test-replication", clusterHandler.TestReplication)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":" + envConfig.HTTPEnv.Port,
		Handler: r,
	}

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown failed", zap.Error(err))
		}
		logger.Info("Infrastructure provisioning service stopped gracefully")
	}()

	logger.Info("Infrastructure provisioning service started", zap.String("port", envConfig.HTTPEnv.Port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run service: %v", err)
	}
}
