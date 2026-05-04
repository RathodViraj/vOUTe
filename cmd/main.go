package main

import (
	"context"
	"log"
	"net/http"
	"voute/db"
	"voute/pkg/bloom"
	"voute/pkg/bookmarks"
	"voute/pkg/comments"
	"voute/pkg/mailing"
	"voute/pkg/middleware"
	"voute/pkg/user"
	"voute/pkg/vote"
	"voute/pkg/ws"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	mongoClinet, err := db.ConnectMongoDB()
	if err != nil {
		panic(err)
	}

	redisClient, err := db.ConnectRedis()
	if err != nil {
		db.CloseMongoDB(mongoClinet)
		panic(err)
	}

	timescaleDB, err := db.ConnectTimescaleDB()
	if err != nil {
		db.CloseMongoDB(mongoClinet)
		db.CloseRedis(redisClient)
		panic(err)
	}

	defer func() {
		db.CloseMongoDB(mongoClinet)
		db.CloseRedis(redisClient)
		db.CloseTimescaleDB(timescaleDB)
	}()

	r := gin.Default()

	bloom, err := bloom.InitBloomFilter(ctx, redisClient, mongoClinet.Database("voute").Collection("users"))
	if err != nil {
		panic(err)
	}

	middleware.AddDBInMiddleware(mongoClinet.Database("voute"), "users")
	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
	}
	r.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.POST("/auth/login", middleware.Login)
	r.GET("/auth/google/login", middleware.GoogleLogin)
	r.GET("/auth/google/callback", middleware.GoogleCallback)
	r.POST("/auth/refresh", middleware.RefershToken)
	r.POST("/auth/logout", middleware.Logout)
	r.POST("/auth/reset-password", middleware.ResetPassword)

	mailRepo := mailing.NewMailingRepository(redisClient, mongoClinet.Database("voute"), "users")
	mailSvc := mailing.NewEmailService(mailRepo)
	mailHandler := mailing.NewMailingHandler(mailSvc)
	mailHandler.RegisterRoutes(r)

	middleware.AddMailingServiceInMiddleware(mailSvc)

	r.POST("/auth/signup-otp", middleware.SignupWithOTP)
	r.POST("/auth/login-otp", middleware.LoginWithOTP)
	r.POST("/auth/login-otp-username", middleware.LoginWithOTPUsername)

	userRepo := user.NewUserRepository(mongoClinet.Database("voute"))
	userSvc := user.NewUserService(userRepo)
	userHandler := user.NewHandler(userSvc, bloom)
	userHandler.AddUserRoutes(r)

	voteRepo := vote.NewVoteRepository(mongoClinet.Database("voute"), "votes", "options", redisClient, timescaleDB)
	voteSvc := vote.NewVoteService(voteRepo)
	voteHandler := vote.NewVoteHandler(voteSvc)
	voteHandler.AddVoteRoutes(r)

	commentRepo := comments.NewCommentRepository(mongoClinet.Database("voute"), "comments")
	commentSvc := comments.NewCommentService(commentRepo)
	commentHandler := comments.NewCommentHandler(commentSvc)
	commentHandler.AddCommentsRoutes(r)

	bookmarksRepo := bookmarks.NewBookmarkRepository(mongoClinet.Database("voute"), "bookmarks")
	bookmarksSvc := bookmarks.NewBookmarkService(bookmarksRepo)
	bookmarksHandler := bookmarks.NewBookmarksHandler(bookmarksSvc)
	bookmarksHandler.AddBookmarksRoutes(r)

	hub := ws.NewHub(voteRepo)
	r.GET("/ws/polls", ws.WSHandler(hub))

	r.Run(":8080")
}
