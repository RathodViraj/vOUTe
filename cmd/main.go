package main

import (
	"context"
	"log"
	"voute/db"
	"voute/pkg/bloom"
	"voute/pkg/bookmarks"
	"voute/pkg/comments"
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
	r.POST("/ws/polls", ws.WSHandler(hub))

	r.Run(":8080")
}
