package main

import (
	"context"
	"voute/db"
	"voute/pkg/bloom"
	"voute/pkg/bookmarks"
	"voute/pkg/comments"
	"voute/pkg/user"
	"voute/pkg/vote"

	"github.com/gin-gonic/gin"
)

func main() {
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
		db.CloseTimescaleDB(timescaleDB)
		panic(err)
	}

	defer func() {
		db.CloseMongoDB(mongoClinet)
		db.CloseRedis(redisClient)
		db.CloseTimescaleDB(timescaleDB)
	}()

	r := gin.Default()

	bloom, err := bloom.InitBloomFilter(ctx, redisClient, mongoClinet.Database("voute").Collection("user"))
	if err != nil {
		panic(err)
	}

	userRepo := user.NewUserRepository(mongoClinet.Database("voute"))
	userSvc := user.NewUserService(userRepo)
	userHandler := user.NewHandler(userSvc, bloom)
	userHandler.AddUserRoute(r)

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

	r.Run(":8080")
}
