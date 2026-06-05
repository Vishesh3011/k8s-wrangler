package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection

func main() {
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collection = initDB(appCtx)
	defer func() {
		if err := collection.Database().Client().Disconnect(appCtx); err != nil {
			fmt.Printf("Error disconnecting from MongoDB: %v\n", err)
		}
	}()

	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/tasks", getTasksHandler)
	http.HandleFunc("/tasks/add", addTaskHandler)

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
	defer func() {
		fmt.Println("Server is shutting down")
	}()
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	responseStr := "i am alive!!!"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthCheckResponse{Status: responseStr})
}

func getTasksFromDB(ctx context.Context, collection *mongo.Collection) ([]Task, error) {
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []Task
	for cursor.Next(ctx) {
		var task Task
		if err := cursor.Decode(&task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func getTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasks, err := getTasksFromDB(ctx, collection)
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		fmt.Printf("Error fetching tasks from DB: %v\n", err)
		return
	}

	taskNames := make([]string, len(tasks))
	for i, task := range tasks {
		taskNames[i] = task.Name
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasksResponse{Tasks: taskNames})
}

func addTaskToDB(ctx context.Context, collection *mongo.Collection, taskName string) error {
	task := Task{
		Name:      taskName,
		CreatedAt: fmt.Sprintf("%d", time.Now().Unix()),
	}
	_, err := collection.InsertOne(ctx, task)
	return err
}

func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req addTaskRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Add the task to the database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = addTaskToDB(ctx, collection, req.Task)
	if err != nil {
		http.Error(w, "Failed to add task", http.StatusInternalServerError)
		fmt.Printf("Error adding task to DB: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type addTaskRequest struct {
	Task string `json:"task"`
}

type healthCheckResponse struct {
	Status string `json:"status"`
}

type tasksResponse struct {
	Tasks []string `json:"tasks"`
}

type Task struct {
	ID        string `json:"id" bson:"_id,omitempty"`
	Name      string `json:"name" bson:"name"`
	CreatedAt string `json:"created_at" bson:"created_at"`
}

func initDB(ctx context.Context) *mongo.Collection {
	mongoDBUrl, dbName := getMongoDBUrl()
	fmt.Printf("Connecting to MongoDB at: %s\n", mongoDBUrl)

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoDBUrl))
	if err != nil {
		fmt.Printf("Error connecting to MongoDB: %v\n", err)
		os.Exit(1)
	}
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		fmt.Printf("Error pinging MongoDB: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully connected to MongoDB")

	tasksCollection := mongoClient.Database(dbName).Collection("tasks")
	_, err = tasksCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.M{"created_at": 1},
		Options: options.Index().SetExpireAfterSeconds(3600),
	})
	if err != nil {
		fmt.Printf("Error creating MongoDB index: %v\n", err)
		os.Exit(1)
	}
	return tasksCollection
}

func getMongoDBUrl() (string, string) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "27017"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "admin"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "tasksdb"
	}

	return fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=admin&authMechanism=SCRAM-SHA-256", user, password, host, port, dbName), dbName
}
