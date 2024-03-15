package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	db "slack-pr-lambda/dynamodb"
	"slack-pr-lambda/logger"
	"slack-pr-lambda/slack"
	"slack-pr-lambda/types"
	"syscall"

	"go.uber.org/zap"
)

func PullRequestHandler(w http.ResponseWriter, r *http.Request) {
	l := logger.LoggerConfig()
	zapLog, _ := l.Build()
	defer func() {
		err := r.Body.Close()
		if err != nil {
			log.Fatalf("error close req body. %v\n", err)
		}
	}()

	defer func() {
		if err := zapLog.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			log.Fatalf("error closing the logger. %v\n", err)
		}
	}()

	// read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		zapLog.Fatal("error read request body",
			zap.Error(err),
		)
	}

	// partial parse into map string JSON
	var result map[string]json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		zapLog.Error("error unmarshal JSON raw message",
			zap.Error(err),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// get unique action key
	var action string
	if err := json.Unmarshal(result["action"], &action); err != nil {
		log.Fatal(err)
	}

	if action == "opened" {
		// parse request
		var input types.OpenPullRequest
		err = json.Unmarshal(body, &input)
		if err != nil {
			zapLog.Error("error unmarshal JSON",
				zap.Error(err),
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		timeStamp, err := slack.SlackSendMessage(input)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if len(input.PullRequest.RequestedReviewers) > 0 {
			reviewers := []string{}
			for _, reviewer := range input.PullRequest.RequestedReviewers {
				reviewers = append(reviewers, reviewer.Login)
			}
			err := slack.SlackSendMessageThreadReviewers(timeStamp, reviewers)

			if err != nil {
				zapLog.Error("error slack send message",
					zap.Error(err),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		svc := db.DynamoDbConnection()
		item := &types.TablePullRequestData{
			ID:             fmt.Sprintf("%d", input.PullRequest.ID),
			PullRequestId:  input.Number,
			SlackTimeStamp: timeStamp,
		}

		err = db.InsertItem(svc, item)
		if err != nil {
			zapLog.Error("error insert data",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if action == "review_requested" {
		// parse request
		var input types.ReviewRequestPullRequest
		err = json.Unmarshal(body, &input)
		if err != nil {
			zapLog.Error("error unmarshal JSON",
				zap.Error(err),
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		svc := db.DynamoDbConnection()
		timeStamp, err := db.GetSlackTimeStampReviewRequest(svc, &input)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		reviewers := []string{input.RequestedReviewer.Login}
		err = slack.SlackSendMessageThreadReviewers(timeStamp, reviewers)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if action == "created" {
		// parse request
		var input types.CommentPullRequest
		err = json.Unmarshal(body, &input)
		if err != nil {
			zapLog.Error("error unmarshal JSON",
				zap.Error(err),
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		svc := db.DynamoDbConnection()
		timeStamp, err := db.GetSlackTimeStampIssue(svc, &input)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = slack.SlackSendMessageThreadComment(timeStamp, &input)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if action == "closed" {
		// parse request
		var input types.ClosedPullRequest
		err = json.Unmarshal(body, &input)
		if err != nil {
			zapLog.Error("error unmarshal JSON",
				zap.Error(err),
			)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		svc := db.DynamoDbConnection()
		timeStamp, err := db.GetSlackTimeStampClose(svc, &input)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = slack.SlackSendMessageThreadClosed(timeStamp)
		if err != nil {
			zapLog.Error("error slack send message",
				zap.Error(err),
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	bodyBytes := Response{
		Message: "Webhook done.",
	}

	j, err := json.Marshal(bodyBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(j)
}
