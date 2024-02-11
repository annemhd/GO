package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

// / TYPES
type Client struct {
	ID_client int    `json:"id_client"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type Salon struct {
	ID_salon int    `json:"id_salon"`
	Name     string `json:"name"`
}

type Coiffeur struct {
	ID_coiffeur int    `json:"id_coiffeur"`
	ID_salon    int    `json:"id_salon"`
	Firstname   string `json:"firstname"`
	Lastname    string `json:"lastname"`
}

type Reservation struct {
	ID_reservation int    `json:"id_reservation"`
	ID_salon       int    `json:"id_salon"`
	ID_coiffeur    int    `json:"id_coiffeur"`
	Date           string `json:"date"`
}

type Creneau struct {
	ID_crenau    int    `json:"id_creanau"`
	ID_coiffeur  int    `json:"id_coiffeur"`
	Date         string `json:"date"`
	Availability bool   `json:"availability"`
}

var (
	db        *sql.DB
	clientsMu sync.RWMutex
	salonsMu  sync.RWMutex
	nextID    = 1
)

// / MAIN
func main() {
	/// BASE DE DONNÉES
	var err error
	db, err = sql.Open("mysql", "goteam:root@tcp(localhost:3306)/golang")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	/// TABLES
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id_client INT AUTO_INCREMENT PRIMARY KEY,
			firstname VARCHAR(150),
			lastname VARCHAR(150),
			email VARCHAR(150),
			password VARCHAR(255)
		);
    `)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS salons (
			id_salon INT AUTO_INCREMENT PRIMARY KEY,
			lastname VARCHAR(150)
		);
    `)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS coiffeurs (
			id_coiffeur INT AUTO_INCREMENT PRIMARY KEY,
			firstname VARCHAR(150),
			lastname VARCHAR(150)
		);
    `)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS reservations (
			id_reservation INT AUTO_INCREMENT PRIMARY KEY,
			id_salon INT,
			id_coiffeur INT,
			date_reservation DATETIME,
			FOREIGN KEY (id_salon) REFERENCES salons(id_salon),
			FOREIGN KEY (id_coiffeur) REFERENCES coiffeurs(id_coiffeur)
		);
    `)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS creneaux (
			id_creneau INT AUTO_INCREMENT PRIMARY KEY,
			id_coiffeur INT,
			date_creneau DATETIME,
			availability BOOLEAN,
			FOREIGN KEY (id_coiffeur) REFERENCES coiffeurs(id_coiffeur)
		);
    `)
	if err != nil {
		log.Fatal(err)
	}

	/// ROUTES
	http.HandleFunc("/api/clients", getClientsHandler)
	http.HandleFunc("/api/clients/add", addClientHandler)
	http.HandleFunc("/api/clients/update", updateclientHandler)
	http.HandleFunc("/api/clients/delete", deleteClientHandler)

	port := 8080
	fmt.Printf("Server is running on port %d...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// CLIENTS
func addClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var newClient Client
	err := json.NewDecoder(r.Body).Decode(&newClient)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := db.Exec("INSERT INTO clients (firstname, lastname, email, password) VALUES (?, ?, ?, ?)", newClient.Firstname, newClient.Lastname, newClient.Email, newClient.Password)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newClient.ID_client = int(id)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newClient)
}

func getClientsHandler(w http.ResponseWriter, r *http.Request) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	// Fetch users from the database
	rows, err := db.Query("SELECT * FROM clients")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var clientList []Client
	for rows.Next() {
		var client Client
		err := rows.Scan(&client.ID_client, &client.Firstname, &client.Lastname, &client.Email, &client.Password)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		clientList = append(clientList, client)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientList)
}

func updateclientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var updatedClient Client
	err := json.NewDecoder(r.Body).Decode(&updatedClient)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clientsMu.RLock()
	defer clientsMu.RUnlock()
	row := db.QueryRow("SELECT id FROM clients WHERE id_client=?", updatedClient.ID_client)
	if err := row.Scan(&updatedClient.ID_client); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("UPDATE users SET firstname=?, lastname=?, email=?, password=? WHERE id_client=?", updatedClient.Firstname, updatedClient.Lastname, updatedClient.Email, updatedClient.Password, updatedClient.ID_client)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedClient)
}

func deleteClientHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	idParam := r.URL.Query().Get("id_client")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	clientsMu.Lock()
	defer clientsMu.Unlock()
	row := db.QueryRow("SELECT id_client FROM clients WHERE id_client=?", id)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM clients WHERE id_client=?", id)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

/// RESERVATION
