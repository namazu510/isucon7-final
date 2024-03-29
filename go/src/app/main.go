package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

var (
	db *sqlx.DB

	roomDataTimeStore *redis.Client
	roomTimeLock      sync.Mutex

	itemCache *cache.Cache
	serverIPs = []string{
		"163.43.29.7",
		"163.43.28.43",
		"153.125.224.115",
		"59.106.219.167",
	}

	knownRoomNameLock sync.Mutex
	knownRoomName     = map[string]int{}
	nextServerID      int
)

func initDB() {
	db_host := os.Getenv("ISU_DB_HOST")
	if db_host == "" {
		db_host = "127.0.0.1"
	}
	db_port := os.Getenv("ISU_DB_PORT")
	if db_port == "" {
		db_port = "3306"
	}
	db_user := os.Getenv("ISU_DB_USER")
	if db_user == "" {
		db_user = "root"
	}
	db_password := os.Getenv("ISU_DB_PASSWORD")
	if db_password != "" {
		db_password = ":" + db_password
	}

	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isudb?parseTime=true&loc=Local&charset=utf8mb4",
		db_user, db_password, db_host, db_port)

	log.Printf("Connecting to db: %q", dsn)
	db, _ = sqlx.Connect("mysql", dsn)
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 3)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Printf("Succeeded to connect db.")
}

func resetRedis() {
	err := roomDataTimeStore.Del("*").Err()
	if err != nil {
		panic(err)
	}
}

func getInitializeHandler(w http.ResponseWriter, r *http.Request) {
	db.MustExec("TRUNCATE TABLE adding")
	db.MustExec("TRUNCATE TABLE buying")
	db.MustExec("TRUNCATE TABLE room_time")

	resetRedis()

	w.WriteHeader(204)
}

func getRoomHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roomName := vars["room_name"]

	// ラウンドロビンで振り分ける
	knownRoomNameLock.Lock()
	hashValue, ok := knownRoomName[roomName]
	if !ok {
		nextServerID = (nextServerID + 1) % len(serverIPs)
		knownRoomName[roomName] = nextServerID
		hashValue = nextServerID
	}
	knownRoomNameLock.Unlock()

	host := serverIPs[hashValue%len(serverIPs)]
	path := "/ws/" + url.PathEscape(roomName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Host string `json:"host"`
		Path string `json:"path"`
	}{
		Host: host,
		Path: path,
	})
}

func wsGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	roomName := vars["room_name"]

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		log.Println("Failed to upgrade", err)
		return
	}
	go serveGameConn(ws, roomName)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	initDB()

	// init redis
	roomDataTimeStore = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	pong, err := roomDataTimeStore.Ping().Result()
	fmt.Println(pong, err)

	// init cache
	itemCache = cache.New(5*time.Minute, 10*time.Minute)

	r := mux.NewRouter()
	r.HandleFunc("/initialize", getInitializeHandler)
	r.HandleFunc("/room/", getRoomHandler)
	r.HandleFunc("/room/{room_name}", getRoomHandler)
	r.HandleFunc("/ws/", wsGameHandler)
	r.HandleFunc("/ws/{room_name}", wsGameHandler)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../public/")))

	log.Fatal(http.ListenAndServe(":5000", handlers.LoggingHandler(os.Stderr, r)))
}
