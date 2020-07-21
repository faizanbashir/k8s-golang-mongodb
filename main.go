package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Config struct {
	Server struct {
		Host         string        `envconfig:"SERVER_HOST"`
		Port         string        `envconfig:"SERVER_PORT"`
		ReadTimeout  time.Duration `envconfig:"SERVER_READ_TIMEOUT"`
		WriteTimeout time.Duration `envconfig:"SERVER_WRITE_TIMEOUT"`
	}
	Database struct {
		DB       string `envconfig:"DB_DATABASE"`
		Username string `envconfig:"DB_USERNAME"`
		Password string `envconfig:"DB_PASSWORD"`
		Hostname string `envconfig:"DB_HOSTNAME"`
		Port     int    `envconfig:"DB_PORT"`
	}
}

type Todo struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Todo        string             `bson:"todo,omitempty" json:"todo"`
	CreatedAt   time.Time          `bson:"created_at,omitempty" json:"created_at"`
	CompletedAt time.Time          `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}

var db *mongo.Database

var cfg Config

func init() {
	loadENV(&cfg)
	err := connectDB()
	if err != nil {
		log.Fatal(err)
	}
}

func loadENV(cfg *Config) {
	err := envconfig.Process("", cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	fmt.Println("Initializing web server!")
	log.Fatal(run())
}

func run() error {
	mux := makeMuxRouter()
	httpAddr := cfg.Server.Port
	log.Println("Listening on:", httpAddr)
	server := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    cfg.Server.ReadTimeout * time.Second,
		WriteTimeout:   cfg.Server.WriteTimeout * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}

func makeMuxRouter() http.Handler {
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message": "Welcome to Golang TODO app!"}`))
	}).Methods(http.MethodGet)
	api.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status": "OK"}`))
	}).Methods(http.MethodGet)
	api.HandleFunc("/todo/list", getTodo).Methods(http.MethodGet)
	api.HandleFunc("/todo/create", addTodo).Methods(http.MethodPost)
	return api
}

func connectDB() error {
	connectionString := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?ssl=true&authSource=admin", cfg.Database.Username, cfg.Database.Password, cfg.Database.Hostname, cfg.Database.Port, cfg.Database.DB)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		fmt.Println(" [-] Failed connecting to MongoDB!")
		return err
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println(" [-] Failed pinging MongoDB!")
		return err
	}

	fmt.Println(" [+] Connected to MongoDB!")
	db = client.Database(cfg.Database.DB)

	return nil
}

func getTodo(w http.ResponseWriter, r *http.Request) {
	collection := db.Collection("todo")

	var todos []Todo
	findOptions := options.Find()
	findOptions.SetLimit(20)

	cur, err := collection.Find(context.TODO(), bson.D{}, findOptions)
	if err != nil {
		log.Fatal(err)
	}

	for cur.Next(context.TODO()) {
		var elem Todo
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		todos = append(todos, elem)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	cur.Close(context.TODO())
	fmt.Printf("Found multiple documents: %+v\n", todos)
	out, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to get todos: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func addTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	err := json.NewDecoder(r.Body).Decode(&todo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = insert(todo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to add todo: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "OK", "message": "Added TODO!"}`))
}

func insert(todo Todo) error {
	collection := db.Collection("todo")

	todo.CreatedAt = time.Now()
	fmt.Println(todo.Todo)

	insertResult, err := collection.InsertOne(context.TODO(), todo)
	if err != nil {
		return err
	}
	fmt.Println(insertResult.InsertedID)
	return nil
}
